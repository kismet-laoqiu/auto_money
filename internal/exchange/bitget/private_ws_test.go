package bitget

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"quantlab/internal/market"
)

func TestPrivateMessageParsesNumericCodeAck(t *testing.T) {
	var msg privateMessage
	if err := json.Unmarshal([]byte(`{"event":"subscribe","code":0}`), &msg); err != nil {
		t.Fatalf("decode private ack: %v", err)
	}
	if msg.Event != "subscribe" || msg.Code != "0" {
		t.Fatalf("unexpected private ack: %+v", msg)
	}
}

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

func TestDecodePrivateFillEvent(t *testing.T) {
	raw := []byte(`{"arg":{"channel":"fill","instId":"default"},"data":[{"clientOid":"ql-1","orderId":"123","symbol":"MSTRUSDT","price":"126.21","baseVolume":"0.04"}]}`)
	events, err := DecodePrivateEvents(raw)
	if err != nil {
		t.Fatalf("decode private fill: %v", err)
	}
	if len(events) != 1 || events[0].Kind() != "order_fill" {
		t.Fatalf("unexpected fill events: %+v", events)
	}
	order, ok := events[0].(market.OrderEvent)
	if !ok {
		t.Fatalf("unexpected event type: %T", events[0])
	}
	if order.SymbolValue != "MSTRUSDT" || order.Size != 0.04 || order.Price != 126.21 {
		t.Fatalf("unexpected fill order event: %+v", order)
	}
}

func TestDecodePrivateOrderEventAllowsEmptyPriceAvg(t *testing.T) {
	raw := []byte(`{"arg":{"channel":"orders","instId":"MSTRUSDT"},"data":[{"clientOid":"ql-1","orderId":"123","status":"live","size":"0.04","priceAvg":""}]}`)
	events, err := DecodePrivateEvents(raw)
	if err != nil {
		t.Fatalf("decode private order with empty priceAvg: %v", err)
	}
	order, ok := events[0].(market.OrderEvent)
	if !ok {
		t.Fatalf("unexpected event type: %T", events[0])
	}
	if order.Price != 0 || order.Size != 0.04 {
		t.Fatalf("unexpected order event: %+v", order)
	}
}

func TestDecodePrivatePositionEvent(t *testing.T) {
	raw := []byte(`{"arg":{"channel":"positions","instId":"default"},"data":[{"instId":"MSTRUSDT","holdSide":"long","total":"0.01","uTime":"1710000000000"}]}`)
	events, err := DecodePrivateEvents(raw)
	if err != nil {
		t.Fatalf("decode private positions: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected one position event, got %+v", events)
	}
	position, ok := events[0].(market.PositionEvent)
	if !ok {
		t.Fatalf("unexpected event type: %T", events[0])
	}
	if position.SymbolValue != "MSTRUSDT" || position.Qty != 0.01 || position.Kind() != "position_snapshot" {
		t.Fatalf("unexpected position event: %+v", position)
	}
}

func TestDecodePrivateShortPositionEventUsesNegativeQty(t *testing.T) {
	raw := []byte(`{"arg":{"channel":"positions","instId":"default"},"data":[{"instId":"MSTRUSDT","holdSide":"short","total":"0.02","uTime":"1710000000000"}]}`)
	events, err := DecodePrivateEvents(raw)
	if err != nil {
		t.Fatalf("decode private positions: %v", err)
	}
	position, ok := events[0].(market.PositionEvent)
	if !ok {
		t.Fatalf("unexpected event type: %T", events[0])
	}
	if position.Qty != -0.02 {
		t.Fatalf("expected negative short qty, got %+v", position)
	}
}

func TestDecodePrivateAccountEvent(t *testing.T) {
	raw := []byte(`{"arg":{"channel":"account","coin":"default"},"data":[{"marginCoin":"USDT","available":"11.98545761","equity":"11.98545761","usdtEquity":"11.985457617660","unrealizedPL":"0.000000000000"}]}`)
	events, err := DecodePrivateEvents(raw)
	if err != nil {
		t.Fatalf("decode private account: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected one account event, got %+v", events)
	}
	account, ok := events[0].(market.AccountEvent)
	if !ok {
		t.Fatalf("unexpected event type: %T", events[0])
	}
	if account.MarginCoin != "USDT" || account.Available != 11.98545761 || account.Kind() != "account_snapshot" {
		t.Fatalf("unexpected account event: %+v", account)
	}
}

func TestPrivateWSSourceReconnectsAfterServerClose(t *testing.T) {
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

		var login loginFrame
		if err := conn.ReadJSON(&login); err != nil {
			t.Errorf("read login: %v", err)
			return
		}
		if login.Op != "login" || len(login.Args) != 1 || login.Args[0].APIKey != "key" {
			t.Errorf("unexpected login frame: %+v", login)
			return
		}
		if err := conn.WriteMessage(websocket.TextMessage, []byte(`{"event":"login","code":0}`)); err != nil {
			t.Errorf("write login ack: %v", err)
			return
		}

		var frame privateSubscribeFrame
		if err := conn.ReadJSON(&frame); err != nil {
			t.Errorf("read subscribe: %v", err)
			return
		}
		if len(frame.Args) != 1 || frame.Args[0].Channel != "orders" {
			t.Errorf("unexpected subscribe frame: %+v", frame)
			return
		}

		ordinal := connections.Add(1)
		payload := []byte(`{"arg":{"channel":"orders","instId":"default"},"data":[{"clientOid":"ql-1","orderId":"123","status":"filled","size":"0.01","priceAvg":"62010"}]}`)
		if ordinal == 2 {
			payload = []byte(`{"arg":{"channel":"orders","instId":"default"},"data":[{"clientOid":"ql-2","orderId":"124","status":"filled","size":"0.02","priceAvg":"62020"}]}`)
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

	source := NewPrivateWSSource(
		"ws"+strings.TrimPrefix(server.URL, "http"),
		PrivateCredentials{Key: "key", Secret: "secret", Passphrase: "passphrase"},
		PrivateSubscription{InstType: "USDT-FUTURES", Channel: "orders", InstID: "default"},
	)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	events := source.Events(ctx)
	first := readPrivateRawMessage(t, events)
	second := readPrivateRawMessage(t, events)
	close(hold)
	cancel()

	if string(first) == string(second) {
		t.Fatalf("expected reconnect to deliver a distinct second payload, got %q then %q", first, second)
	}
	if connections.Load() < 2 {
		t.Fatalf("expected reconnect, saw %d websocket connections", connections.Load())
	}
}

func readPrivateRawMessage(t *testing.T, events <-chan []byte) []byte {
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
