package hyperliquid

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	common "quantlab/internal/venue/common"
)

const Venue = "hyperliquid"

type Fill struct {
	Coin          string  `json:"coin"`
	Side          string  `json:"side"`
	Px            string  `json:"px"`
	Sz            string  `json:"sz"`
	Fee           string  `json:"fee"`
	ClosedPnl     string  `json:"closedPnl"`
	Time          int64   `json:"time"`
	Oid           string  `json:"oid"`
	Tid           string  `json:"tid"`
	StartPosition float64 `json:"startPosition,omitempty"`
}

type Candle struct {
	Coin  string `json:"coin"`
	Open  string `json:"o"`
	High  string `json:"h"`
	Low   string `json:"l"`
	Close string `json:"c"`
	Vol   string `json:"v"`
	Time  int64  `json:"t"`
	End   int64  `json:"T"`
}

type AssetPosition struct {
	Type     string `json:"type"`
	Position struct {
		Coin          string `json:"coin"`
		Szi           string `json:"szi"`
		EntryPx       string `json:"entryPx"`
		LeverageValue struct {
			Value string `json:"value"`
		} `json:"leverage"`
		PositionValue string `json:"positionValue"`
	} `json:"position"`
}

type OrderStatus struct {
	Coin       string `json:"coin"`
	OrderID    string `json:"oid"`
	ClientOID  string `json:"cloid"`
	Side       string `json:"side"`
	Status     string `json:"status"`
	LimitPx    string `json:"limitPx"`
	Sz         string `json:"sz"`
	Filled     string `json:"filled"`
	Timestamp  int64  `json:"timestamp"`
	ReduceOnly bool   `json:"reduceOnly"`
}

func AdaptUserFillsByTime(body []byte, leaderID string, mapper *common.Mapper) ([]common.LeaderFill, error) {
	var fills []Fill
	if err := json.Unmarshal(body, &fills); err != nil {
		return nil, fmt.Errorf("decode user fills by time: %w", err)
	}
	out := make([]common.LeaderFill, 0, len(fills))
	for _, fill := range fills {
		instrument, err := resolveInstrument(fill.Coin, common.MarketTypePerp, mapper)
		if err != nil {
			return nil, err
		}
		out = append(out, common.LeaderFill{
			Venue:           Venue,
			MarketType:      instrument.MarketType,
			LeaderID:        leaderID,
			CanonicalSymbol: instrument.CanonicalSymbol,
			VenueSymbol:     instrument.VenueSymbol,
			Timestamp:       time.UnixMilli(fill.Time).UTC(),
			Side:            strings.ToLower(strings.TrimSpace(fill.Side)),
			Price:           parseFloat(fill.Px),
			Size:            parseFloat(fill.Sz),
			Fee:             parseFloat(fill.Fee),
			RealizedPnL:     parseFloat(fill.ClosedPnl),
			VenueOrderID:    fill.Oid,
			VenueTradeID:    fill.Tid,
		})
	}
	return out, nil
}

func AdaptCandleSnapshot(body []byte, mapper *common.Mapper) ([]common.Candle, error) {
	var candles []Candle
	if err := json.Unmarshal(body, &candles); err != nil {
		return nil, fmt.Errorf("decode candle snapshot: %w", err)
	}
	out := make([]common.Candle, 0, len(candles))
	for _, candle := range candles {
		instrument, err := resolveInstrument(candle.Coin, common.MarketTypePerp, mapper)
		if err != nil {
			return nil, err
		}
		out = append(out, common.Candle{
			Venue:           Venue,
			MarketType:      instrument.MarketType,
			CanonicalSymbol: instrument.CanonicalSymbol,
			VenueSymbol:     instrument.VenueSymbol,
			Interval:        "15m",
			OpenTime:        time.UnixMilli(candle.Time).UTC(),
			CloseTime:       time.UnixMilli(candle.End).UTC(),
			Open:            parseFloat(candle.Open),
			High:            parseFloat(candle.High),
			Low:             parseFloat(candle.Low),
			Close:           parseFloat(candle.Close),
			Volume:          parseFloat(candle.Vol),
		})
	}
	return out, nil
}

func AdaptClearinghouseState(body []byte, leaderID string, mapper *common.Mapper) ([]common.LeaderStateSnapshot, error) {
	var payload struct {
		AssetPositions []AssetPosition `json:"assetPositions"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("decode clearinghouse state: %w", err)
	}
	out := make([]common.LeaderStateSnapshot, 0, len(payload.AssetPositions))
	for _, item := range payload.AssetPositions {
		instrument, err := resolveInstrument(item.Position.Coin, common.MarketTypePerp, mapper)
		if err != nil {
			return nil, err
		}
		out = append(out, common.LeaderStateSnapshot{
			Venue:           Venue,
			MarketType:      instrument.MarketType,
			LeaderID:        leaderID,
			CanonicalSymbol: instrument.CanonicalSymbol,
			VenueSymbol:     instrument.VenueSymbol,
			PositionSize:    parseFloat(item.Position.Szi),
			Leverage:        parseFloat(item.Position.LeverageValue.Value),
			EntryPrice:      parseFloat(item.Position.EntryPx),
			AccountValue:    parseFloat(item.Position.PositionValue),
		})
	}
	return out, nil
}

func AdaptOrderStatus(body []byte, leaderID string, mapper *common.Mapper) ([]common.OrderUpdate, error) {
	var orders []OrderStatus
	if err := json.Unmarshal(body, &orders); err != nil {
		return nil, fmt.Errorf("decode order status: %w", err)
	}
	out := make([]common.OrderUpdate, 0, len(orders))
	for _, order := range orders {
		instrument, err := resolveInstrument(order.Coin, common.MarketTypePerp, mapper)
		if err != nil {
			return nil, err
		}
		out = append(out, common.OrderUpdate{
			Venue:           Venue,
			MarketType:      instrument.MarketType,
			LeaderID:        leaderID,
			CanonicalSymbol: instrument.CanonicalSymbol,
			VenueSymbol:     instrument.VenueSymbol,
			Timestamp:       time.UnixMilli(order.Timestamp).UTC(),
			Side:            strings.ToLower(strings.TrimSpace(order.Side)),
			Status:          order.Status,
			Price:           parseFloat(order.LimitPx),
			Size:            parseFloat(order.Sz),
			FilledSize:      parseFloat(order.Filled),
			ReduceOnly:      order.ReduceOnly,
			ClientOrderID:   order.ClientOID,
			VenueOrderID:    order.OrderID,
		})
	}
	return out, nil
}

func resolveInstrument(raw string, marketType common.MarketType, mapper *common.Mapper) (common.Instrument, error) {
	if mapper == nil {
		return common.Instrument{}, fmt.Errorf("hyperliquid adapter mapper is nil")
	}
	instrument, ok := mapper.Resolve(Venue, marketType, raw)
	if !ok {
		return common.Instrument{}, fmt.Errorf("hyperliquid adapter missing mapping for %s %s", marketType, raw)
	}
	return instrument, nil
}

func parseFloat(raw string) float64 {
	if strings.TrimSpace(raw) == "" {
		return 0
	}
	number, _ := strconv.ParseFloat(raw, 64)
	return number
}
