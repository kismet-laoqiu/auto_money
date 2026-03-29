package bitget

import (
	"context"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"

	"quantlab/internal/market"
)

const defaultPrivateWSURL = "wss://ws.bitget.com/v2/ws/private"

type PrivateCredentials struct {
	Key        string
	Secret     string
	Passphrase string
}

type PrivateSubscription struct {
	InstType string
	Channel  string
	InstID   string
	Coin     string
}

type PrivateWSSource struct {
	url           string
	creds         PrivateCredentials
	subscriptions []PrivateSubscription
	dialer        *websocket.Dialer
}

type loginFrame struct {
	Op   string     `json:"op"`
	Args []loginArg `json:"args"`
}

type loginArg struct {
	APIKey     string `json:"apiKey"`
	Passphrase string `json:"passphrase"`
	Timestamp  string `json:"timestamp"`
	Sign       string `json:"sign"`
}

type privateMessage struct {
	Event string           `json:"event"`
	Code  privateCode      `json:"code"`
	Msg   string           `json:"msg"`
	Arg   privateArg       `json:"arg"`
	Data  []privatePayload `json:"data"`
}

type privateCode string

func (code *privateCode) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*code = ""
		return nil
	}
	var text string
	if err := json.Unmarshal(data, &text); err == nil {
		*code = privateCode(text)
		return nil
	}
	var number json.Number
	if err := json.Unmarshal(data, &number); err == nil {
		*code = privateCode(number.String())
		return nil
	}
	return fmt.Errorf("decode private code: %s", string(data))
}

type privateArg struct {
	InstType string `json:"instType,omitempty"`
	Channel  string `json:"channel"`
	InstID   string `json:"instId,omitempty"`
	Coin     string `json:"coin,omitempty"`
}

type privateSubscribeFrame struct {
	Op   string       `json:"op"`
	Args []privateArg `json:"args"`
}

type privatePayload struct {
	ClientOID    string `json:"clientOid"`
	OrderID      string `json:"orderId"`
	Status       string `json:"status"`
	Size         string `json:"size"`
	Price        string `json:"price"`
	PriceAvg     string `json:"priceAvg"`
	BaseVolume   string `json:"baseVolume"`
	Symbol       string `json:"symbol"`
	InstID       string `json:"instId"`
	HoldSide     string `json:"holdSide"`
	Total        string `json:"total"`
	UTime        string `json:"uTime"`
	MarginCoin   string `json:"marginCoin"`
	Available    string `json:"available"`
	Equity       string `json:"equity"`
	AccountEq    string `json:"accountEquity"`
	USDTEq       string `json:"usdtEquity"`
	UnrealizedPL string `json:"unrealizedPL"`
}

func BuildLoginFrame(ts string, creds PrivateCredentials, signer *Signer) loginFrame {
	sign := signer.Sign(ts, "GET", "/user/verify", "")
	return loginFrame{Op: "login", Args: []loginArg{{APIKey: creds.Key, Passphrase: creds.Passphrase, Timestamp: ts, Sign: sign}}}
}

func NewPrivateWSSource(url string, creds PrivateCredentials, subscriptions ...PrivateSubscription) *PrivateWSSource {
	if strings.TrimSpace(url) == "" {
		url = defaultPrivateWSURL
	}
	return &PrivateWSSource{
		url:           url,
		creds:         creds,
		subscriptions: append([]PrivateSubscription(nil), subscriptions...),
		dialer:        websocket.DefaultDialer,
	}
}

func (source *PrivateWSSource) Events(ctx context.Context) <-chan []byte {
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

		ts := fmt.Sprintf("%d", time.Now().UTC().Unix())
		if err := conn.WriteJSON(BuildLoginFrame(ts, source.creds, NewSigner(source.creds.Secret))); err != nil {
			return
		}
		if err := waitForPrivateEvent(ctx, conn, "login"); err != nil {
			return
		}
		args := make([]privateArg, 0, len(source.subscriptions))
		for _, sub := range source.subscriptions {
			args = append(args, privateArg{
				InstType: sub.InstType,
				Channel:  sub.Channel,
				InstID:   sub.InstID,
				Coin:     sub.Coin,
			})
		}
		if err := conn.WriteJSON(privateSubscribeFrame{Op: "subscribe", Args: args}); err != nil {
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

func waitForPrivateEvent(ctx context.Context, conn *websocket.Conn, want string) error {
	for {
		_ = conn.SetReadDeadline(time.Now().Add(15 * time.Second))
		_, data, err := conn.ReadMessage()
		if err != nil {
			return err
		}
		var msg privateMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			continue
		}
		if msg.Event == want {
			if msg.Code != "" && msg.Code != "0" {
				return fmt.Errorf("private websocket %s failed: code=%s msg=%s", want, msg.Code, msg.Msg)
			}
			_ = conn.SetReadDeadline(time.Time{})
			return nil
		}
		if err := ctx.Err(); err != nil {
			return err
		}
	}
}

func DecodePrivateEvents(raw []byte) ([]market.MarketEvent, error) {
	var msg privateMessage
	if err := json.Unmarshal(raw, &msg); err != nil {
		return nil, err
	}
	if msg.Event != "" {
		return nil, nil
	}
	switch msg.Arg.Channel {
	case "orders":
		return decodeOrderEvents(msg)
	case "fill":
		return decodeFillEvents(msg)
	case "positions":
		return decodePositionEvents(msg)
	default:
		return decodeAccountEvents(msg)
	}
}

func decodeFillEvents(msg privateMessage) ([]market.MarketEvent, error) {
	events := make([]market.MarketEvent, 0, len(msg.Data))
	for index, row := range msg.Data {
		sizeRaw := row.BaseVolume
		if sizeRaw == "" {
			sizeRaw = row.Size
		}
		size, err := strconv.ParseFloat(sizeRaw, 64)
		if err != nil {
			return nil, fmt.Errorf("parse fill size: %w", err)
		}
		priceRaw := row.Price
		if priceRaw == "" {
			priceRaw = row.PriceAvg
		}
		price, err := strconv.ParseFloat(priceRaw, 64)
		if err != nil {
			return nil, fmt.Errorf("parse fill price: %w", err)
		}
		symbol := row.Symbol
		if symbol == "" {
			symbol = row.InstID
		}
		if symbol == "" {
			symbol = msg.Arg.InstID
		}
		events = append(events, market.OrderEvent{
			EventIDValue: buildPrivateStreamEventID("fill_ws", symbol, row.OrderID, row.ClientOID, priceRaw, row.BaseVolume, strconv.Itoa(index)),
			SymbolValue:  symbol,
			Ts:           time.Time{},
			ClientOID:    row.ClientOID,
			OrderID:      row.OrderID,
			Status:       "filled",
			Size:         size,
			Price:        price,
			KindValue:    "order_fill",
		})
	}
	return events, nil
}

func parseOptionalFloat(value string) (float64, error) {
	if strings.TrimSpace(value) == "" {
		return 0, nil
	}
	return strconv.ParseFloat(value, 64)
}

func buildPrivateStreamEventID(channel string, fields ...string) string {
	payload := channel
	for _, field := range fields {
		payload += "|" + strings.TrimSpace(field)
	}
	sum := sha1.Sum([]byte(payload))
	return fmt.Sprintf("%s:%x", channel, sum[:8])
}

func decodeOrderEvents(msg privateMessage) ([]market.MarketEvent, error) {
	events := make([]market.MarketEvent, 0, len(msg.Data))
	for index, row := range msg.Data {
		size, err := parseOptionalFloat(row.Size)
		if err != nil {
			return nil, fmt.Errorf("parse order size: %w", err)
		}
		price, err := parseOptionalFloat(row.PriceAvg)
		if err != nil {
			return nil, fmt.Errorf("parse order avg price: %w", err)
		}
		kind := "order_update"
		if strings.EqualFold(row.Status, "filled") {
			kind = "order_fill"
		}
		events = append(events, market.OrderEvent{
			EventIDValue: buildPrivateStreamEventID("order_ws", msg.Arg.InstID, row.OrderID, row.ClientOID, row.Status, row.Size, row.PriceAvg, strconv.Itoa(index)),
			SymbolValue:  msg.Arg.InstID,
			Ts:           time.Time{},
			ClientOID:    row.ClientOID,
			OrderID:      row.OrderID,
			Status:       row.Status,
			Size:         size,
			Price:        price,
			KindValue:    kind,
		})
	}
	return events, nil
}

func decodePositionEvents(msg privateMessage) ([]market.MarketEvent, error) {
	events := make([]market.MarketEvent, 0, len(msg.Data))
	for index, row := range msg.Data {
		qty, err := parseSignedPositionQty(row.HoldSide, row.Total)
		if err != nil {
			return nil, err
		}
		ts, err := parseMillis(row.UTime)
		if err != nil {
			return nil, err
		}
		symbol := row.InstID
		if symbol == "" {
			symbol = msg.Arg.InstID
		}
		events = append(events, market.PositionEvent{
			EventIDValue: buildPrivateStreamEventID("position_ws", symbol, row.HoldSide, row.Total, row.UTime, strconv.Itoa(index)),
			SymbolValue:  symbol,
			Ts:           ts,
			Qty:          qty,
		})
	}
	return events, nil
}

func decodeAccountEvents(msg privateMessage) ([]market.MarketEvent, error) {
	events := make([]market.MarketEvent, 0, len(msg.Data))
	for index, row := range msg.Data {
		equityRaw := row.Equity
		if equityRaw == "" {
			equityRaw = row.AccountEq
		}
		available, err := strconv.ParseFloat(row.Available, 64)
		if err != nil {
			return nil, fmt.Errorf("parse account available: %w", err)
		}
		equity, err := strconv.ParseFloat(equityRaw, 64)
		if err != nil {
			return nil, fmt.Errorf("parse account equity: %w", err)
		}
		usdtEq, err := strconv.ParseFloat(row.USDTEq, 64)
		if err != nil {
			return nil, fmt.Errorf("parse account usdt equity: %w", err)
		}
		unrealizedPL, err := strconv.ParseFloat(row.UnrealizedPL, 64)
		if err != nil {
			return nil, fmt.Errorf("parse account unrealized pnl: %w", err)
		}
		events = append(events, market.AccountEvent{
			EventIDValue: buildPrivateStreamEventID("account_ws", row.MarginCoin, row.Available, equityRaw, row.USDTEq, row.UnrealizedPL, strconv.Itoa(index)),
			SymbolValue:  row.MarginCoin,
			Ts:           time.Time{},
			MarginCoin:   row.MarginCoin,
			Available:    available,
			Equity:       equity,
			USDTEq:       usdtEq,
			UnrealizedPL: unrealizedPL,
		})
	}
	return events, nil
}
