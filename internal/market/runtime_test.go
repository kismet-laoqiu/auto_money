package market

import (
	"context"
	"fmt"
	"testing"
	"time"
)

type fakeBootstrapLoader struct {
	events []BarClosedEvent
}

func (loader fakeBootstrapLoader) FetchCandles(_ context.Context, _, _, _ string, _ int) ([]BarClosedEvent, error) {
	return append([]BarClosedEvent(nil), loader.events...), nil
}

type fakePrivateBootstrapLoader struct {
	accounts  []AccountEvent
	positions []PositionEvent
}

func (loader fakePrivateBootstrapLoader) FetchFuturesAccounts(_ context.Context, _ string) ([]AccountEvent, error) {
	return append([]AccountEvent(nil), loader.accounts...), nil
}

func (loader fakePrivateBootstrapLoader) FetchFuturesPositions(_ context.Context, _, _ string) ([]PositionEvent, error) {
	return append([]PositionEvent(nil), loader.positions...), nil
}

type fakeEventSource struct {
	messages [][]byte
}

func (source fakeEventSource) Events(ctx context.Context) <-chan []byte {
	out := make(chan []byte, len(source.messages))
	go func() {
		defer close(out)
		for _, msg := range source.messages {
			select {
			case <-ctx.Done():
				return
			case out <- msg:
			}
		}
	}()
	return out
}

type fakeDecoder struct {
	batches [][]MarketEvent
}

func (decoder *fakeDecoder) Decode([]byte) ([]MarketEvent, error) {
	if len(decoder.batches) == 0 {
		return nil, nil
	}
	next := decoder.batches[0]
	decoder.batches = decoder.batches[1:]
	return next, nil
}

type recordedEvent struct {
	source string
	kind   string
}

type recordingStore struct {
	events []recordedEvent
}

func (store *recordingStore) AppendEvent(_ context.Context, source string, evt MarketEvent, _ []byte) (int64, error) {
	store.events = append(store.events, recordedEvent{source: source, kind: evt.Kind()})
	return int64(len(store.events)), nil
}

type dedupeRecordingStore struct {
	events []recordedEvent
	seen   map[string]bool
}

func (store *dedupeRecordingStore) AppendEvent(_ context.Context, source string, evt MarketEvent, _ []byte) (int64, error) {
	if store.seen == nil {
		store.seen = map[string]bool{}
	}
	if store.seen[evt.EventID()] {
		return 0, fmt.Errorf("UNIQUE constraint failed: event_log.event_id")
	}
	store.seen[evt.EventID()] = true
	store.events = append(store.events, recordedEvent{source: source, kind: evt.Kind()})
	return int64(len(store.events)), nil
}

func TestRuntimePersistsBootstrapAndPublicEvents(t *testing.T) {
	store := &recordingStore{}
	runtime := NewRuntime(RuntimeConfig{
		Symbol:         "MSTRUSDT",
		ProductType:    "USDT-FUTURES",
		Interval:       "1m",
		BootstrapLimit: 2,
		AppendEvent:    store.AppendEvent,
		Loader: fakeBootstrapLoader{events: []BarClosedEvent{
			{EventIDValue: "bootstrap-1", SymbolValue: "MSTRUSDT", Interval: "1m", Ts: time.Unix(1710000000, 0), Close: 62000},
		}},
		Source: fakeEventSource{messages: [][]byte{[]byte(`{"kind":"public"}`)}},
		Decoder: &fakeDecoder{batches: [][]MarketEvent{{
			TradeTickEvent{EventIDValue: "trade-1", SymbolValue: "MSTRUSDT", Ts: time.Unix(1710000001, 0), Price: 62010, Size: 0.02, Side: "buy"},
			BarClosedEvent{EventIDValue: "bar-1", SymbolValue: "MSTRUSDT", Interval: "1m", Ts: time.Unix(1710000060, 0), Close: 62080},
		}}},
	})
	if err := runtime.Run(context.Background()); err != nil {
		t.Fatalf("run runtime: %v", err)
	}
	if len(store.events) != 3 {
		t.Fatalf("unexpected event count: %d", len(store.events))
	}
	if store.events[0].source != "market.bootstrap" || store.events[0].kind != "bar_closed" {
		t.Fatalf("unexpected bootstrap event: %+v", store.events[0])
	}
	if store.events[1].source != "market.public" || store.events[1].kind != "trade_tick" {
		t.Fatalf("unexpected public trade event: %+v", store.events[1])
	}
	if store.events[2].source != "market.public" || store.events[2].kind != "bar_closed" {
		t.Fatalf("unexpected public bar event: %+v", store.events[2])
	}
}

func TestRuntimePersistsPrivateBootstrapAndStreamingEvents(t *testing.T) {
	store := &recordingStore{}
	publicDecoder := &fakeDecoder{batches: [][]MarketEvent{{
		TradeTickEvent{EventIDValue: "trade-1", SymbolValue: "MSTRUSDT", Ts: time.Unix(1710000001, 0), Price: 62010, Size: 0.02, Side: "buy"},
	}}}
	privateDecoder := &fakeDecoder{batches: [][]MarketEvent{{
		OrderEvent{EventIDValue: "order-1", SymbolValue: "MSTRUSDT", Ts: time.Unix(1710000002, 0), OrderID: "oid-1", Status: "filled", KindValue: "order_fill"},
	}}}
	runtime := NewRuntime(RuntimeConfig{
		Symbol:         "MSTRUSDT",
		ProductType:    "USDT-FUTURES",
		MarginCoin:     "USDT",
		Interval:       "1m",
		BootstrapLimit: 2,
		AppendEvent:    store.AppendEvent,
		Loader: fakeBootstrapLoader{events: []BarClosedEvent{
			{EventIDValue: "bootstrap-1", SymbolValue: "MSTRUSDT", Interval: "1m", Ts: time.Unix(1710000000, 0), Close: 62000},
		}},
		PrivateLoader: fakePrivateBootstrapLoader{
			accounts:  []AccountEvent{{EventIDValue: "account-1", SymbolValue: "USDT", MarginCoin: "USDT", Available: 10}},
			positions: []PositionEvent{{EventIDValue: "position-1", SymbolValue: "MSTRUSDT", Qty: 0.01}},
		},
		Source:         fakeEventSource{messages: [][]byte{[]byte(`{"kind":"public"}`)}},
		Decoder:        publicDecoder,
		PrivateSource:  fakeEventSource{messages: [][]byte{[]byte(`{"kind":"private"}`)}},
		PrivateDecoder: privateDecoder,
	})
	if err := runtime.Run(context.Background()); err != nil {
		t.Fatalf("run runtime: %v", err)
	}
	if len(store.events) != 5 {
		t.Fatalf("unexpected event count: %d", len(store.events))
	}
	counts := map[string]int{}
	for _, event := range store.events {
		counts[event.source+":"+event.kind]++
	}
	if counts["market.bootstrap:bar_closed"] != 1 {
		t.Fatalf("missing bootstrap bar: %+v", counts)
	}
	if counts["market.private:account_snapshot"] != 1 || counts["market.private:position_snapshot"] != 1 || counts["market.private:order_fill"] != 1 {
		t.Fatalf("missing private events: %+v", counts)
	}
	if counts["market.public:trade_tick"] != 1 {
		t.Fatalf("missing public trade event: %+v", counts)
	}
}

func TestRuntimeIgnoresDuplicateBootstrapEventsOnRestart(t *testing.T) {
	store := &dedupeRecordingStore{}
	runtime := NewRuntime(RuntimeConfig{
		Symbol:         "MSTRUSDT",
		ProductType:    "USDT-FUTURES",
		Interval:       "1m",
		BootstrapLimit: 1,
		AppendEvent:    store.AppendEvent,
		Loader: fakeBootstrapLoader{events: []BarClosedEvent{{
			EventIDValue: "bootstrap-1",
			SymbolValue:  "MSTRUSDT",
			Interval:     "1m",
			Ts:           time.Unix(1710000000, 0),
			Close:        62000,
		}}},
	})
	if err := runtime.Run(context.Background()); err != nil {
		t.Fatalf("first run: %v", err)
	}
	if err := runtime.Run(context.Background()); err != nil {
		t.Fatalf("second run: %v", err)
	}
	if len(store.events) != 1 {
		t.Fatalf("unexpected event count after restart: %d", len(store.events))
	}
}
