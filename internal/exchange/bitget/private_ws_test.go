package bitget

import "testing"

func TestDecodePrivateOrderEvent(t *testing.T) {
	raw := []byte(`{"arg":{"channel":"orders","instId":"BTCUSDT"},"data":[{"clientOid":"ql-1","orderId":"123","status":"filled","size":"0.01","priceAvg":"62010"}]}`)
	events, err := DecodePrivateEvents(raw)
	if err != nil {
		t.Fatalf("decode private events: %v", err)
	}
	if len(events) != 1 || events[0].Kind() != "order_fill" {
		t.Fatalf("unexpected events: %+v", events)
	}
}
