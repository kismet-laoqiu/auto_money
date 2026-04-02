package bitget

import (
	"fmt"
	"strings"
	"time"

	exbitget "quantlab/internal/exchange/bitget"
	"quantlab/internal/market"
	common "quantlab/internal/venue/common"
)

const Venue = "bitget"

func ProductTypeToMarketType(productType string) common.MarketType {
	if strings.Contains(strings.ToUpper(strings.TrimSpace(productType)), "SPOT") {
		return common.MarketTypeSpot
	}
	return common.MarketTypePerp
}

func AdaptCandles(events []market.BarClosedEvent, mapper *common.Mapper) ([]common.Candle, error) {
	out := make([]common.Candle, 0, len(events))
	for _, event := range events {
		instrument, err := resolveInstrument(event.SymbolValue, event.MarketType, mapper)
		if err != nil {
			return nil, err
		}
		closeTime := event.Ts.UTC().Add(intervalDuration(event.Interval))
		out = append(out, common.Candle{
			Venue:           Venue,
			MarketType:      instrument.MarketType,
			CanonicalSymbol: instrument.CanonicalSymbol,
			VenueSymbol:     instrument.VenueSymbol,
			Interval:        event.Interval,
			OpenTime:        event.Ts.UTC(),
			CloseTime:       closeTime,
			Open:            event.Open,
			High:            event.High,
			Low:             event.Low,
			Close:           event.Close,
			Volume:          event.Volume,
		})
	}
	return out, nil
}

func AdaptTicker(symbol, productType string, lastPrice, markPrice float64, ts time.Time, mapper *common.Mapper) (common.PriceSnapshot, error) {
	instrument, err := resolveInstrument(symbol, string(ProductTypeToMarketType(productType)), mapper)
	if err != nil {
		return common.PriceSnapshot{}, err
	}
	return common.PriceSnapshot{
		Venue:           Venue,
		MarketType:      instrument.MarketType,
		CanonicalSymbol: instrument.CanonicalSymbol,
		VenueSymbol:     instrument.VenueSymbol,
		Timestamp:       ts.UTC(),
		LastPrice:       lastPrice,
		MarkPrice:       markPrice,
	}, nil
}

func AdaptContractRules(rules map[string]exbitget.ContractRule, productType string, ts time.Time, mapper *common.Mapper) ([]common.ContractSpec, error) {
	specs := make([]common.ContractSpec, 0, len(rules))
	for symbol, rule := range rules {
		instrument, err := resolveInstrument(symbol, string(ProductTypeToMarketType(productType)), mapper)
		if err != nil {
			return nil, err
		}
		specs = append(specs, common.ContractSpec{
			Venue:           Venue,
			MarketType:      instrument.MarketType,
			CanonicalSymbol: instrument.CanonicalSymbol,
			VenueSymbol:     instrument.VenueSymbol,
			Timestamp:       ts.UTC(),
			MinTradeSize:    rule.MinTradeNum,
			SizeStep:        rule.SizeMultiplier,
			MaxLeverage:     rule.MaxLeverage,
		})
	}
	return specs, nil
}

func AdaptPositionEvent(event market.PositionEvent, leaderID string, mapper *common.Mapper) (common.LeaderStateSnapshot, error) {
	instrument, err := resolveInstrument(event.SymbolValue, event.MarketType, mapper)
	if err != nil {
		return common.LeaderStateSnapshot{}, err
	}
	return common.LeaderStateSnapshot{
		Venue:           Venue,
		MarketType:      instrument.MarketType,
		LeaderID:        leaderID,
		CanonicalSymbol: instrument.CanonicalSymbol,
		VenueSymbol:     instrument.VenueSymbol,
		Timestamp:       event.Ts.UTC(),
		PositionSize:    event.Qty,
	}, nil
}

func AdaptOrderEvent(event market.OrderEvent, leaderID string, mapper *common.Mapper) (common.OrderUpdate, error) {
	instrument, err := resolveInstrument(event.SymbolValue, event.MarketType, mapper)
	if err != nil {
		return common.OrderUpdate{}, err
	}
	status := strings.TrimSpace(event.Status)
	filledSize := 0.0
	if strings.Contains(strings.ToLower(status), "fill") {
		filledSize = event.Size
	}
	return common.OrderUpdate{
		Venue:           Venue,
		MarketType:      instrument.MarketType,
		LeaderID:        leaderID,
		CanonicalSymbol: instrument.CanonicalSymbol,
		VenueSymbol:     instrument.VenueSymbol,
		Timestamp:       event.Ts.UTC(),
		Status:          status,
		Price:           event.Price,
		Size:            event.Size,
		FilledSize:      filledSize,
		ClientOrderID:   event.ClientOID,
		VenueOrderID:    event.OrderID,
	}, nil
}

func resolveInstrument(symbol string, marketType string, mapper *common.Mapper) (common.Instrument, error) {
	if mapper == nil {
		return common.Instrument{}, fmt.Errorf("bitget adapter mapper is nil")
	}
	instrument, ok := mapper.Resolve(Venue, common.MarketType(marketType), symbol)
	if !ok {
		return common.Instrument{}, fmt.Errorf("bitget adapter missing mapping for %s %s", marketType, symbol)
	}
	return instrument, nil
}

func intervalDuration(interval string) time.Duration {
	switch strings.ToLower(strings.TrimSpace(interval)) {
	case "1m":
		return time.Minute
	case "5m":
		return 5 * time.Minute
	case "15m":
		return 15 * time.Minute
	case "1h":
		return time.Hour
	case "4h":
		return 4 * time.Hour
	case "1d":
		return 24 * time.Hour
	case "1w":
		return 7 * 24 * time.Hour
	default:
		return 0
	}
}
