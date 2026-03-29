package sqlite

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"quantlab/internal/market"
	"quantlab/internal/trader"
)

func TestAppendEventAndCheckpointRoundTrip(t *testing.T) {
	store := newTestStore(t)
	evt := market.BarClosedEvent{EventIDValue: "evt-1", SymbolValue: "BTCUSDT", Interval: "1m", Ts: time.Unix(1710000000, 0)}
	raw := []byte(`{"close":"62000"}`)
	if _, err := store.AppendEvent(context.Background(), "market", evt, raw); err != nil {
		t.Fatalf("append event: %v", err)
	}
	if got := readEventPayload(t, store, evt.EventID()); got != string(raw) {
		t.Fatalf("unexpected logged payload: %s", got)
	}
	if err := store.SaveCheckpoint(context.Background(), "btc", trader.EngineState{ArmingState: trader.ArmingSafe}); err != nil {
		t.Fatalf("save checkpoint: %v", err)
	}
	got, err := store.LoadCheckpoint(context.Background(), "btc")
	if err != nil {
		t.Fatalf("load checkpoint: %v", err)
	}
	if got.ArmingState != trader.ArmingSafe {
		t.Fatalf("unexpected checkpoint: %+v", got)
	}
}

func TestAppendEventAllowsNilPayload(t *testing.T) {
	store := newTestStore(t)
	evt := market.BarClosedEvent{EventIDValue: "evt-2", SymbolValue: "BTCUSDT", Interval: "1m", Ts: time.Unix(1710000001, 0)}
	if _, err := store.AppendEvent(context.Background(), "market", evt, nil); err != nil {
		t.Fatalf("append event with nil payload: %v", err)
	}
	if got := readEventPayload(t, store, evt.EventID()); got != "null" {
		t.Fatalf("unexpected nil payload encoding: %s", got)
	}
}

func TestAppendEventReturnsMonotonicSeqAndListEventsAfter(t *testing.T) {
	store := newTestStore(t)
	first, err := store.AppendEvent(context.Background(), "market", market.BarClosedEvent{
		EventIDValue: "evt-1",
		SymbolValue:  "BTCUSDT",
		Interval:     "1m",
		Ts:           time.Unix(1710000000, 0),
	}, []byte(`{"close":"62000"}`))
	if err != nil {
		t.Fatalf("append first: %v", err)
	}
	second, err := store.AppendEvent(context.Background(), "trader", market.BarClosedEvent{
		EventIDValue: "evt-2",
		SymbolValue:  "BTCUSDT",
		Interval:     "1m",
		Ts:           time.Unix(1710000060, 0),
	}, []byte(`{"close":"62050"}`))
	if err != nil {
		t.Fatalf("append second: %v", err)
	}
	if second <= first {
		t.Fatalf("seq must increase: first=%d second=%d", first, second)
	}

	events, err := store.ListEventsAfter(context.Background(), first, 10, "trader")
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	if len(events) != 1 || events[0].Seq != second || events[0].Source != "trader" {
		t.Fatalf("unexpected events: %+v", events)
	}
}

func TestConsumerCursorRoundTrip(t *testing.T) {
	store := newTestStore(t)
	if err := store.SaveConsumerCursor(context.Background(), "traderd", 42); err != nil {
		t.Fatalf("save cursor: %v", err)
	}
	got, err := store.LoadConsumerCursor(context.Background(), "traderd")
	if err != nil {
		t.Fatalf("load cursor: %v", err)
	}
	if got != 42 {
		t.Fatalf("unexpected cursor: %d", got)
	}
}

func TestNewStoreCreatesParentDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "demo", "live-state.db")
	store, err := NewStore(path)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	if store == nil {
		t.Fatalf("expected store instance")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("stat db path: %v", err)
	}
}

func newTestStore(t *testing.T) *Store {
	t.Helper()
	path := filepath.Join(t.TempDir(), "live-state.db")
	store, err := NewStore(path)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	return store
}

func readEventPayload(t *testing.T, store *Store, eventID string) string {
	t.Helper()
	var body []byte
	err := store.db.QueryRow(`SELECT payload_json FROM event_log WHERE event_id = ?`, eventID).Scan(&body)
	if err != nil {
		t.Fatalf("read event payload: %v", err)
	}
	return string(body)
}
