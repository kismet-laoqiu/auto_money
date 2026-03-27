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
	if err := store.AppendEvent(context.Background(), evt, raw); err != nil {
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
	if err := store.AppendEvent(context.Background(), evt, nil); err != nil {
		t.Fatalf("append event with nil payload: %v", err)
	}
	if got := readEventPayload(t, store, evt.EventID()); got != "null" {
		t.Fatalf("unexpected nil payload encoding: %s", got)
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
	err := store.db.QueryRow(`SELECT payload_json FROM market_event_log WHERE event_id = ?`, eventID).Scan(&body)
	if err != nil {
		t.Fatalf("read event payload: %v", err)
	}
	return string(body)
}
