package bitget

import (
	"testing"
	"time"

	"quantlab/internal/market"
)

func TestBuildTradeTickEvent(t *testing.T) {
	raw := []byte(`{"arg":{"instId":"MSTRUSDT","channel":"trade"},"data":[{"ts":"1710000000000","price":"62000.1","size":"0.05","side":"buy"}]}`)
	events, err := NewPublicWSDecoder().Decode(raw)
	if err != nil {
		t.Fatalf("decode public events: %v", err)
	}
	if len(events) != 1 || events[0].Kind() != "trade_tick" {
		t.Fatalf("unexpected events: %+v", events)
	}
}

func TestDecodeCandleSnapshotsClosePreviousBar(t *testing.T) {
	decoder := NewPublicWSDecoder()
	first := []byte(`{"arg":{"instId":"MSTRUSDT","channel":"candle1m"},"data":[["1710000000000","62000","62100","61950","62050","12.5"]]}`)
	events, err := decoder.Decode(first)
	if err != nil {
		t.Fatalf("decode first candle: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("first candle snapshot must not close a bar: %+v", events)
	}

	second := []byte(`{"arg":{"instId":"MSTRUSDT","channel":"candle1m"},"data":[["1710000060000","62050","62200","62000","62180","8.5"]]}`)
	events, err = decoder.Decode(second)
	if err != nil {
		t.Fatalf("decode second candle: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected one closed bar, got %+v", events)
	}
	bar, ok := events[0].(market.BarClosedEvent)
	if !ok {
		t.Fatalf("unexpected event type: %T", events[0])
	}
	if !bar.Ts.Equal(time.UnixMilli(1710000000000).UTC()) || bar.Close != 62050 {
		t.Fatalf("unexpected closed bar: %+v", bar)
	}
}
