package market

import (
	"testing"
	"time"
)

var baseTs = time.Date(2024, 3, 9, 16, 0, 0, 0, time.UTC)

func TestMicroBarAggregatorClosesOneSecondBars(t *testing.T) {
	agg := NewMicroBarAggregator(time.Second)
	out := agg.Push(TradeTickEvent{SymbolValue: "BTCUSDT", Price: 100, Size: 1, Ts: baseTs})
	if len(out) != 0 {
		t.Fatalf("unexpected close on first tick")
	}
	out = agg.Push(TradeTickEvent{SymbolValue: "BTCUSDT", Price: 99, Size: 1, Ts: baseTs.Add(1100 * time.Millisecond)})
	if len(out) != 1 || out[0].Kind() != "micro_bar_closed" {
		t.Fatalf("expected one closed micro bar, got %+v", out)
	}
}
