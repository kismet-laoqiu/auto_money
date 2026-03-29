package bitget

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"

	"quantlab/internal/market"
)

const defaultPublicWSURL = "wss://ws.bitget.com/v2/ws/public"

type PublicSubscription struct {
	InstType string
	Channel  string
	InstID   string
}

type PublicWSSource struct {
	url           string
	subscriptions []PublicSubscription
	dialer        *websocket.Dialer
}

type publicMessage struct {
	Event  string          `json:"event"`
	Action string          `json:"action"`
	Arg    publicArg       `json:"arg"`
	Data   json.RawMessage `json:"data"`
}

type publicArg struct {
	InstType string `json:"instType,omitempty"`
	InstID   string `json:"instId"`
	Channel  string `json:"channel"`
}

type publicSubscribeFrame struct {
	Op   string      `json:"op"`
	Args []publicArg `json:"args"`
}

type publicTradeRow struct {
	TS    string `json:"ts"`
	Price string `json:"price"`
	Size  string `json:"size"`
	Side  string `json:"side"`
}

type PublicWSDecoder struct {
	lastCandle map[string]market.BarClosedEvent
}

func NewPublicWSSource(url string, subscriptions ...PublicSubscription) *PublicWSSource {
	if strings.TrimSpace(url) == "" {
		url = defaultPublicWSURL
	}
	return &PublicWSSource{
		url:           url,
		subscriptions: append([]PublicSubscription(nil), subscriptions...),
		dialer:        websocket.DefaultDialer,
	}
}

func (source *PublicWSSource) Events(ctx context.Context) <-chan []byte {
	out := make(chan []byte)
	go func() {
		defer close(out)
		conn, _, err := source.dialer.DialContext(ctx, source.url, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		go func() {
			<-ctx.Done()
			_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "context canceled"), time.Now().Add(time.Second))
			_ = conn.Close()
		}()

		args := make([]publicArg, 0, len(source.subscriptions))
		for _, sub := range source.subscriptions {
			args = append(args, publicArg{InstType: sub.InstType, Channel: sub.Channel, InstID: sub.InstID})
		}
		if err := conn.WriteJSON(publicSubscribeFrame{Op: "subscribe", Args: args}); err != nil {
			return
		}
		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				return
			}
			select {
			case <-ctx.Done():
				return
			case out <- data:
			}
		}
	}()
	return out
}

func NewPublicWSDecoder() *PublicWSDecoder {
	return &PublicWSDecoder{lastCandle: map[string]market.BarClosedEvent{}}
}

func DecodePublicEvents(raw []byte) ([]market.MarketEvent, error) {
	return NewPublicWSDecoder().Decode(raw)
}

func (decoder *PublicWSDecoder) Decode(raw []byte) ([]market.MarketEvent, error) {
	var msg publicMessage
	if err := json.Unmarshal(raw, &msg); err != nil {
		return nil, err
	}
	if msg.Event != "" {
		return nil, nil
	}
	switch msg.Arg.Channel {
	case "trade":
		return decodeTrades(msg)
	case "ticker":
		return nil, nil
	default:
		if strings.HasPrefix(msg.Arg.Channel, "candle") {
			return decoder.decodeCandles(msg)
		}
		return nil, nil
	}
}

func decodeTrades(msg publicMessage) ([]market.MarketEvent, error) {
	var payload []publicTradeRow
	if err := json.Unmarshal(msg.Data, &payload); err == nil && len(payload) > 0 {
		events := make([]market.MarketEvent, 0, len(payload))
		for index, row := range payload {
			ts, err := strconv.ParseInt(row.TS, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("parse trade ts: %w", err)
			}
			price, err := strconv.ParseFloat(row.Price, 64)
			if err != nil {
				return nil, fmt.Errorf("parse trade price: %w", err)
			}
			size, err := strconv.ParseFloat(row.Size, 64)
			if err != nil {
				return nil, fmt.Errorf("parse trade size: %w", err)
			}
			events = append(events, market.TradeTickEvent{
				EventIDValue: fmt.Sprintf("trade:%s:%s:%d", msg.Arg.InstID, row.TS, index),
				SymbolValue:  msg.Arg.InstID,
				Ts:           time.UnixMilli(ts).UTC(),
				Price:        price,
				Size:         size,
				Side:         row.Side,
			})
		}
		return events, nil
	}

	var rows [][]string
	if err := json.Unmarshal(msg.Data, &rows); err != nil {
		return nil, err
	}
	events := make([]market.MarketEvent, 0, len(rows))
	for index, row := range rows {
		if len(row) < 4 {
			continue
		}
		ts, err := strconv.ParseInt(row[0], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse trade ts: %w", err)
		}
		price, err := strconv.ParseFloat(row[1], 64)
		if err != nil {
			return nil, fmt.Errorf("parse trade price: %w", err)
		}
		size, err := strconv.ParseFloat(row[2], 64)
		if err != nil {
			return nil, fmt.Errorf("parse trade size: %w", err)
		}
		events = append(events, market.TradeTickEvent{
			EventIDValue: fmt.Sprintf("trade:%s:%s:%d", msg.Arg.InstID, row[0], index),
			SymbolValue:  msg.Arg.InstID,
			Ts:           time.UnixMilli(ts).UTC(),
			Price:        price,
			Size:         size,
			Side:         row[3],
		})
	}
	return events, nil
}

func (decoder *PublicWSDecoder) decodeCandles(msg publicMessage) ([]market.MarketEvent, error) {
	var rows [][]string
	if err := json.Unmarshal(msg.Data, &rows); err != nil {
		return nil, err
	}
	interval := strings.TrimPrefix(msg.Arg.Channel, "candle")
	bars := make([]market.BarClosedEvent, 0, len(rows))
	for _, row := range rows {
		if len(row) < 6 {
			continue
		}
		ts, err := strconv.ParseInt(row[0], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse candle ts: %w", err)
		}
		openValue, err := strconv.ParseFloat(row[1], 64)
		if err != nil {
			return nil, fmt.Errorf("parse candle open: %w", err)
		}
		highValue, err := strconv.ParseFloat(row[2], 64)
		if err != nil {
			return nil, fmt.Errorf("parse candle high: %w", err)
		}
		lowValue, err := strconv.ParseFloat(row[3], 64)
		if err != nil {
			return nil, fmt.Errorf("parse candle low: %w", err)
		}
		closeValue, err := strconv.ParseFloat(row[4], 64)
		if err != nil {
			return nil, fmt.Errorf("parse candle close: %w", err)
		}
		volumeValue, err := strconv.ParseFloat(row[5], 64)
		if err != nil {
			return nil, fmt.Errorf("parse candle volume: %w", err)
		}
		bars = append(bars, market.BarClosedEvent{
			EventIDValue: fmt.Sprintf("candle:%s:%s:%d", msg.Arg.InstID, interval, ts),
			SymbolValue:  msg.Arg.InstID,
			Interval:     interval,
			Ts:           time.UnixMilli(ts).UTC(),
			Open:         openValue,
			High:         highValue,
			Low:          lowValue,
			Close:        closeValue,
			Volume:       volumeValue,
		})
	}
	sort.Slice(bars, func(i, j int) bool {
		return bars[i].Ts.Before(bars[j].Ts)
	})

	key := msg.Arg.InstID + ":" + msg.Arg.Channel
	prev, hasPrev := decoder.lastCandle[key]
	events := make([]market.MarketEvent, 0, len(bars))
	for _, bar := range bars {
		if hasPrev && bar.Ts.After(prev.Ts) {
			events = append(events, prev)
		}
		prev = bar
		hasPrev = true
	}
	if hasPrev {
		decoder.lastCandle[key] = prev
	}
	return events, nil
}
