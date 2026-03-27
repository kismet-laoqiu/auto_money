package bitget

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"quantlab/internal/market"
)

type PrivateCredentials struct {
	Key        string
	Secret     string
	Passphrase string
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
	Arg  privateArg       `json:"arg"`
	Data []privatePayload `json:"data"`
}

type privateArg struct {
	Channel string `json:"channel"`
	InstID  string `json:"instId"`
}

type privatePayload struct {
	ClientOID string `json:"clientOid"`
	OrderID   string `json:"orderId"`
	Status    string `json:"status"`
	Size      string `json:"size"`
	PriceAvg  string `json:"priceAvg"`
}

func BuildLoginFrame(ts string, creds PrivateCredentials, signer *Signer) loginFrame {
	sign := signer.Sign(ts, "GET", "/user/verify", "")
	return loginFrame{Op: "login", Args: []loginArg{{APIKey: creds.Key, Passphrase: creds.Passphrase, Timestamp: ts, Sign: sign}}}
}

func DecodePrivateEvents(raw []byte) ([]market.MarketEvent, error) {
	var msg privateMessage
	if err := json.Unmarshal(raw, &msg); err != nil {
		return nil, err
	}
	switch msg.Arg.Channel {
	case "orders":
		return decodeOrderEvents(msg)
	case "positions":
		return decodePositionEvents(msg)
	default:
		return decodeAccountEvents(msg)
	}
}

func decodeOrderEvents(msg privateMessage) ([]market.MarketEvent, error) {
	events := make([]market.MarketEvent, 0, len(msg.Data))
	for index, row := range msg.Data {
		size, err := strconv.ParseFloat(row.Size, 64)
		if err != nil {
			return nil, fmt.Errorf("parse order size: %w", err)
		}
		price, err := strconv.ParseFloat(row.PriceAvg, 64)
		if err != nil {
			return nil, fmt.Errorf("parse order avg price: %w", err)
		}
		kind := "order_update"
		if strings.EqualFold(row.Status, "filled") {
			kind = "order_fill"
		}
		events = append(events, market.OrderEvent{
			EventIDValue: fmt.Sprintf("order:%s:%d", row.OrderID, index),
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

func decodePositionEvents(privateMessage) ([]market.MarketEvent, error) {
	return nil, nil
}

func decodeAccountEvents(privateMessage) ([]market.MarketEvent, error) {
	return nil, nil
}
