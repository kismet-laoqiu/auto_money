package trader

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"quantlab/internal/core"
	"quantlab/internal/market"
)

func TestRuntimeEmitsEntryIntentOnlyWhenArmedAndRiskPasses(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)
	bar := market.BarClosedEvent{
		EventIDValue: "bar-intent-armed",
		SymbolValue:  "MSTRUSDT",
		Interval:     "1m",
		Ts:           time.Unix(1710000180, 0).UTC(),
		Close:        125,
	}
	raw, err := json.Marshal(bar)
	if err != nil {
		t.Fatalf("marshal bar: %v", err)
	}
	seq, err := store.AppendEvent(ctx, "market.public", bar, raw)
	if err != nil {
		t.Fatalf("append bar: %v", err)
	}
	runtime := NewRuntime(RuntimeConfig{
		ConsumerKey:   "traderd",
		CheckpointKey: "trader.runtime",
		Store:         store,
		Engine: NewEngine(Config{
			ArmingState: ArmingArmed,
			Strategy: &runtimeStrategy{signal: core.Signal{
				Side:    core.Long,
				Score:   4.2,
				Entry:   125,
				Stop:    120,
				Target:  135,
				Reasons: []string{"runtime"},
			}},
		}),
		Sources: []string{"market.public"},
		Risk:    NewRiskEngine(RiskConfig{MaxLeverage: 3}),
		RunID:   "run-17",
	})

	if err := runtime.ProcessAvailable(ctx); err != nil {
		t.Fatalf("process available: %v", err)
	}
	events, err := store.ListEventsAfter(ctx, seq, 10, "trader")
	if err != nil {
		t.Fatalf("list trader events: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected candidate + intent, got %d events: %+v", len(events), events)
	}
	if events[0].Kind != "candidate.created" || events[1].Kind != "entry.intent.created" {
		t.Fatalf("unexpected trader events: %+v", events)
	}
	var payload map[string]any
	if err := json.Unmarshal(events[1].Payload, &payload); err != nil {
		t.Fatalf("decode intent payload: %v", err)
	}
	if payload["size"] != "0.04" {
		t.Fatalf("expected intent size 0.04, got %+v", payload)
	}
	if payload["leverage"] != "3" {
		t.Fatalf("expected leverage 3, got %+v", payload)
	}
}

func TestRuntimeDoesNotEmitEntryIntentInSafeMode(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)
	bar := market.BarClosedEvent{
		EventIDValue: "bar-intent-safe",
		SymbolValue:  "MSTRUSDT",
		Interval:     "1m",
		Ts:           time.Unix(1710000120, 0).UTC(),
		Close:        125,
	}
	raw, err := json.Marshal(bar)
	if err != nil {
		t.Fatalf("marshal bar: %v", err)
	}
	seq, err := store.AppendEvent(ctx, "market.public", bar, raw)
	if err != nil {
		t.Fatalf("append bar: %v", err)
	}
	runtime := NewRuntime(RuntimeConfig{
		ConsumerKey:   "traderd",
		CheckpointKey: "trader.runtime",
		Store:         store,
		Engine: NewEngine(Config{
			ArmingState: ArmingSafe,
			Strategy: &runtimeStrategy{signal: core.Signal{
				Side:    core.Long,
				Score:   4.2,
				Entry:   125,
				Stop:    120,
				Target:  135,
				Reasons: []string{"runtime"},
			}},
		}),
		Sources: []string{"market.public"},
		Risk:    NewRiskEngine(RiskConfig{MaxLeverage: 3}),
		RunID:   "run-17",
	})

	if err := runtime.ProcessAvailable(ctx); err != nil {
		t.Fatalf("process available: %v", err)
	}
	events, err := store.ListEventsAfter(ctx, seq, 10, "trader")
	if err != nil {
		t.Fatalf("list trader events: %v", err)
	}
	if len(events) != 1 || events[0].Kind != "candidate.created" {
		t.Fatalf("safe mode must emit only candidate event: %+v", events)
	}
}
