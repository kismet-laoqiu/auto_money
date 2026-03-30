package bitget

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"

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

func TestPublicWSSourceReconnectsAfterServerClose(t *testing.T) {
	previousReconnectDelay := wsReconnectDelay
	previousPingInterval := wsPingInterval
	wsReconnectDelay = 10 * time.Millisecond
	wsPingInterval = time.Hour
	defer func() {
		wsReconnectDelay = previousReconnectDelay
		wsPingInterval = previousPingInterval
	}()

	var connections atomic.Int32
	hold := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Upgrade(w, r, nil, 1024, 1024)
		if err != nil {
			t.Errorf("upgrade websocket: %v", err)
			return
		}
		defer conn.Close()

		var frame publicSubscribeFrame
		if err := conn.ReadJSON(&frame); err != nil {
			t.Errorf("read subscribe: %v", err)
			return
		}
		if len(frame.Args) != 1 || frame.Args[0].InstID != "MSTRUSDT" || frame.Args[0].Channel != "trade" {
			t.Errorf("unexpected subscribe frame: %+v", frame)
			return
		}

		ordinal := connections.Add(1)
		payload := []byte(`{"arg":{"instId":"MSTRUSDT","channel":"trade"},"data":[{"ts":"1710000000000","price":"62000.1","size":"0.05","side":"buy"}]}`)
		if ordinal == 2 {
			payload = []byte(`{"arg":{"instId":"MSTRUSDT","channel":"trade"},"data":[{"ts":"1710000001000","price":"62010.1","size":"0.03","side":"sell"}]}`)
		}
		if err := conn.WriteMessage(websocket.TextMessage, payload); err != nil {
			t.Errorf("write payload: %v", err)
			return
		}
		if ordinal == 1 {
			return
		}
		<-hold
	}))
	defer server.Close()

	source := NewPublicWSSource(
		"ws"+strings.TrimPrefix(server.URL, "http"),
		PublicSubscription{InstType: "USDT-FUTURES", Channel: "trade", InstID: "MSTRUSDT"},
	)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	events := source.Events(ctx)
	first := readRawMessage(t, events)
	second := readRawMessage(t, events)
	close(hold)
	cancel()

	if string(first) == string(second) {
		t.Fatalf("expected reconnect to deliver a distinct second payload, got %q then %q", first, second)
	}
	if connections.Load() < 2 {
		t.Fatalf("expected reconnect, saw %d websocket connections", connections.Load())
	}
}

func readRawMessage(t *testing.T, events <-chan []byte) []byte {
	t.Helper()
	select {
	case payload, ok := <-events:
		if !ok {
			t.Fatal("events channel closed unexpectedly")
		}
		return payload
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for websocket payload")
		return nil
	}
}
