package bitget

import "testing"

func TestBuildTradeTickEvent(t *testing.T) {
	raw := []byte(`{"arg":{"instId":"BTCUSDT","channel":"trade"},"data":[["1710000000000","62000.1","0.05","buy"]]}`)
	events, err := DecodePublicEvents(raw)
	if err != nil {
		t.Fatalf("decode public events: %v", err)
	}
	if len(events) != 1 || events[0].Kind() != "trade_tick" {
		t.Fatalf("unexpected events: %+v", events)
	}
}
