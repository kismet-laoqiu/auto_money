package bitget

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"quantlab/internal/market"
)

type publicMessage struct {
	Arg  publicArg  `json:"arg"`
	Data [][]string `json:"data"`
}

type publicArg struct {
	InstID  string `json:"instId"`
	Channel string `json:"channel"`
}

func DecodePublicEvents(raw []byte) ([]market.MarketEvent, error) {
	var msg publicMessage
	if err := json.Unmarshal(raw, &msg); err != nil {
		return nil, err
	}
	switch msg.Arg.Channel {
	case "trade":
		return decodeTrades(msg)
	case "ticker":
		return decodeTickers(msg)
	default:
		return decodeCandles(msg)
	}
}

func decodeTrades(msg publicMessage) ([]market.MarketEvent, error) {
	events := make([]market.MarketEvent, 0, len(msg.Data))
	for index, row := range msg.Data {
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

func decodeTickers(publicMessage) ([]market.MarketEvent, error) {
	return nil, nil
}

func decodeCandles(publicMessage) ([]market.MarketEvent, error) {
	return nil, nil
}
