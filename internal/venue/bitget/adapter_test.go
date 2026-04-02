package bitget

import (
	"path/filepath"
	"testing"
	"time"

	exbitget "quantlab/internal/exchange/bitget"
	"quantlab/internal/market"
	common "quantlab/internal/venue/common"
)

func testMapper(t *testing.T) *common.Mapper {
	t.Helper()
	entries, err := common.LoadMappings(filepath.Join("..", "common", "testdata", "instrument_map.json"))
	if err != nil {
		t.Fatalf("load mappings: %v", err)
	}
	return common.NewMapper(entries)
}

func TestAdaptCandlesUsesCanonicalSymbolAndVenueIdentity(t *testing.T) {
	mapper := testMapper(t)
	candles, err := AdaptCandles([]market.BarClosedEvent{{
		EventIDValue: "bar-1",
		SymbolValue:  "BTCUSDT",
		MarketType:   "perp",
		Interval:     "15m",
		Ts:           time.Unix(1710000000, 0).UTC(),
		Open:         62000,
		High:         62100,
		Low:          61950,
		Close:        62050,
		Volume:       12.5,
	}}, mapper)
	if err != nil {
		t.Fatalf("adapt candles: %v", err)
	}
	if len(candles) != 1 {
		t.Fatalf("unexpected candle count: %d", len(candles))
	}
	if candles[0].Venue != Venue || candles[0].MarketType != common.MarketTypePerp || candles[0].CanonicalSymbol != "BTC" {
		t.Fatalf("unexpected adapted candle: %+v", candles[0])
	}
	if candles[0].CloseTime.Sub(candles[0].OpenTime) != 15*time.Minute {
		t.Fatalf("unexpected candle duration: %+v", candles[0])
	}
}

func TestAdaptTickerAndContractRules(t *testing.T) {
	mapper := testMapper(t)
	snapshot, err := AdaptTicker("BTCUSDT", "USDT-FUTURES", 62000.1, 61999.9, time.Unix(1710000000, 0), mapper)
	if err != nil {
		t.Fatalf("adapt ticker: %v", err)
	}
	if snapshot.CanonicalSymbol != "BTC" || snapshot.LastPrice != 62000.1 || snapshot.MarkPrice != 61999.9 {
		t.Fatalf("unexpected price snapshot: %+v", snapshot)
	}

	specs, err := AdaptContractRules(map[string]exbitget.ContractRule{
		"BTCUSDT": {Symbol: "BTCUSDT", MinTradeNum: 0.001, SizeMultiplier: 0.001, MaxLeverage: 125},
	}, "USDT-FUTURES", time.Unix(1710000000, 0), mapper)
	if err != nil {
		t.Fatalf("adapt contract rules: %v", err)
	}
	if len(specs) != 1 || specs[0].CanonicalSymbol != "BTC" || specs[0].MaxLeverage != 125 {
		t.Fatalf("unexpected contract specs: %+v", specs)
	}
}
