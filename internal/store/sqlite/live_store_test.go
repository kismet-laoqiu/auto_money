package sqlite

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"quantlab/internal/market"
	"quantlab/internal/trader"
)

func TestAppendEventAndCheckpointRoundTrip(t *testing.T) {
	store := newTestStore(t)
	evt := market.BarClosedEvent{EventIDValue: "evt-1", SymbolValue: "BTCUSDT", Interval: "1m", Ts: time.Unix(1710000000, 0)}
	if err := store.AppendEvent(context.Background(), evt, []byte(`{"close":"62000"}`)); err != nil {
		t.Fatalf("append event: %v", err)
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

func newTestStore(t *testing.T) *Store {
	t.Helper()
	path := filepath.Join(t.TempDir(), "live-state.db")
	store, err := NewStore(path)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	return store
}
