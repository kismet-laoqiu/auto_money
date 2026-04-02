package common

import (
	"path/filepath"
	"testing"
	"time"
)

func TestLoadMappingsResolveBitgetAndHyperliquidSymbols(t *testing.T) {
	entries, err := LoadMappings(filepath.Join("testdata", "instrument_map.json"))
	if err != nil {
		t.Fatalf("load mappings: %v", err)
	}
	mapper := NewMapper(entries)

	bitgetBTC, ok := mapper.Resolve("bitget", MarketTypePerp, "BTCUSDT")
	if !ok {
		t.Fatalf("expected bitget BTCUSDT mapping")
	}
	if bitgetBTC.CanonicalSymbol != "BTC" || bitgetBTC.VenueSymbol != "BTCUSDT" {
		t.Fatalf("unexpected bitget mapping: %+v", bitgetBTC)
	}

	hyperliquidBTC, ok := mapper.Resolve("hyperliquid", MarketTypePerp, "BTC")
	if !ok {
		t.Fatalf("expected hyperliquid BTC mapping")
	}
	if hyperliquidBTC.CanonicalSymbol != "BTC" || hyperliquidBTC.VenueSymbol != "BTC" {
		t.Fatalf("unexpected hyperliquid perp mapping: %+v", hyperliquidBTC)
	}

	spotIndex, ok := mapper.Resolve("hyperliquid", MarketTypeSpot, "@107")
	if !ok {
		t.Fatalf("expected hyperliquid spot index mapping")
	}
	if spotIndex.CanonicalSymbol != "HYPE" || spotIndex.AssetID != "107" {
		t.Fatalf("unexpected hyperliquid spot mapping: %+v", spotIndex)
	}
}

func TestCommonModelsRetainCoreVenueSemantics(t *testing.T) {
	openTime := time.Unix(1710000000, 0).UTC()
	closeTime := openTime.Add(15 * time.Minute)
	candle := Candle{
		Venue:           "bitget",
		MarketType:      MarketTypePerp,
		CanonicalSymbol: "BTC",
		VenueSymbol:     "BTCUSDT",
		Interval:        "15m",
		OpenTime:        openTime,
		CloseTime:       closeTime,
		Open:            62000,
		High:            62100,
		Low:             61900,
		Close:           62050,
		Volume:          12.5,
	}
	if candle.Venue != "bitget" || candle.MarketType != MarketTypePerp || candle.CanonicalSymbol != "BTC" {
		t.Fatalf("unexpected candle identity: %+v", candle)
	}
	if !candle.OpenTime.Equal(openTime) || !candle.CloseTime.Equal(closeTime) {
		t.Fatalf("unexpected candle times: %+v", candle)
	}
}
