package hyperliquid

import (
	"path/filepath"
	"testing"

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

func TestAdaptUserFillsByTime(t *testing.T) {
	mapper := testMapper(t)
	body := []byte(`[{"coin":"BTC","side":"B","px":"62000","sz":"0.01","fee":"-0.5","closedPnl":"12.3","time":1710000000000,"oid":"1","tid":"2"}]`)
	fills, err := AdaptUserFillsByTime(body, "0xleader", mapper)
	if err != nil {
		t.Fatalf("adapt fills: %v", err)
	}
	if len(fills) != 1 || fills[0].CanonicalSymbol != "BTC" || fills[0].VenueTradeID != "2" {
		t.Fatalf("unexpected fills: %+v", fills)
	}
}

func TestAdaptCandleSnapshotAndOrderStatus(t *testing.T) {
	mapper := testMapper(t)
	candles, err := AdaptCandleSnapshot([]byte(`[{"coin":"BTC","o":"62000","h":"62100","l":"61900","c":"62050","v":"12.5","t":1710000000000,"T":1710000900000}]`), mapper)
	if err != nil {
		t.Fatalf("adapt candles: %v", err)
	}
	if len(candles) != 1 || candles[0].CanonicalSymbol != "BTC" || candles[0].Volume != 12.5 {
		t.Fatalf("unexpected candles: %+v", candles)
	}

	orders, err := AdaptOrderStatus([]byte(`[{"coin":"BTC","oid":"11","cloid":"cid-1","side":"B","status":"filled","limitPx":"62000","sz":"0.02","filled":"0.02","timestamp":1710000000000,"reduceOnly":false}]`), "0xleader", mapper)
	if err != nil {
		t.Fatalf("adapt order status: %v", err)
	}
	if len(orders) != 1 || orders[0].CanonicalSymbol != "BTC" || orders[0].FilledSize != 0.02 {
		t.Fatalf("unexpected orders: %+v", orders)
	}
}

func TestAdaptClearinghouseState(t *testing.T) {
	mapper := testMapper(t)
	body := []byte(`{"assetPositions":[{"type":"oneWay","position":{"coin":"BTC","szi":"0.03","entryPx":"62000","leverage":{"value":"3"},"positionValue":"1860"}}]}`)
	states, err := AdaptClearinghouseState(body, "0xleader", mapper)
	if err != nil {
		t.Fatalf("adapt clearinghouse state: %v", err)
	}
	if len(states) != 1 || states[0].CanonicalSymbol != "BTC" || states[0].Leverage != 3 {
		t.Fatalf("unexpected states: %+v", states)
	}
}
