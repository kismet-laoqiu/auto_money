package trader

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"quantlab/internal/core"
	"quantlab/internal/market"
)

func TestRuntimeRestoresCheckpointAndCursor(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)
	saved := EngineState{
		ArmingState: ArmingDegraded,
		Symbols: map[string]SymbolState{
			"MSTRUSDT": {
				Phase:    PhaseHoldingLong,
				Tranches: 1,
			},
		},
	}
	if err := store.SaveCheckpoint(ctx, "trader.runtime", saved); err != nil {
		t.Fatalf("save checkpoint: %v", err)
	}
	if err := store.SaveConsumerCursor(ctx, "traderd", 17); err != nil {
		t.Fatalf("save cursor: %v", err)
	}

	runtime := NewRuntime(RuntimeConfig{
		ConsumerKey:   "traderd",
		CheckpointKey: "trader.runtime",
		Store:         store,
		Engine:        NewEngine(Config{Strategy: &runtimeStrategy{signal: core.Signal{Side: core.Flat}}}),
		Sources:       []string{"market.bootstrap", "market.public"},
	})

	if err := runtime.ProcessAvailable(ctx); err != nil {
		t.Fatalf("process available: %v", err)
	}
	if runtime.cursor != 17 {
		t.Fatalf("unexpected cursor: %d", runtime.cursor)
	}
	got := runtime.engine.State()
	if got.ArmingState != ArmingDegraded {
		t.Fatalf("unexpected arming state: %+v", got)
	}
	if got.Symbols["MSTRUSDT"].Phase != PhaseHoldingLong {
		t.Fatalf("unexpected restored state: %+v", got.Symbols["MSTRUSDT"])
	}
}

func TestRuntimeWritesCandidateEnvelopeForBarClosed(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)
	input := market.BarClosedEvent{
		EventIDValue: "bar-1",
		SymbolValue:  "MSTRUSDT",
		Interval:     "1m",
		Ts:           time.Unix(1710000000, 0).UTC(),
		Open:         100,
		High:         101,
		Low:          99,
		Close:        100.5,
		Volume:       42,
	}
	raw, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("marshal market event: %v", err)
	}
	seq, err := store.AppendEvent(ctx, "market.public", input, raw)
	if err != nil {
		t.Fatalf("append event: %v", err)
	}

	runtime := NewRuntime(RuntimeConfig{
		ConsumerKey:   "traderd",
		CheckpointKey: "trader.runtime",
		Store:         store,
		Engine: NewEngine(Config{
			Strategy: &runtimeStrategy{signal: core.Signal{
				Side:    core.Long,
				Score:   4.2,
				Entry:   100.5,
				Stop:    99,
				Target:  104,
				Reasons: []string{"runtime"},
			}},
		}),
		Sources: []string{"market.public"},
	})

	if err := runtime.ProcessAvailable(ctx); err != nil {
		t.Fatalf("process available: %v", err)
	}
	events, err := store.ListEventsAfter(ctx, seq, 10, "trader")
	if err != nil {
		t.Fatalf("list trader events: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected one trader event, got %d", len(events))
	}
	if events[0].Kind != "candidate.created" {
		t.Fatalf("unexpected trader event kind: %+v", events[0])
	}

	var candidate CandidateEvent
	if err := json.Unmarshal(events[0].Payload, &candidate); err != nil {
		t.Fatalf("decode candidate event: %v", err)
	}
	if candidate.SymbolValue != "MSTRUSDT" || candidate.Interval != "1m" {
		t.Fatalf("unexpected candidate payload: %+v", candidate)
	}
	if candidate.Score != 4.2 || candidate.Side != core.Long {
		t.Fatalf("unexpected candidate signal: %+v", candidate)
	}

	cursor, err := store.LoadConsumerCursor(ctx, "traderd")
	if err != nil {
		t.Fatalf("load cursor: %v", err)
	}
	if cursor != seq {
		t.Fatalf("unexpected saved cursor: %d", cursor)
	}

	state, err := store.LoadCheckpoint(ctx, "trader.runtime")
	if err != nil {
		t.Fatalf("load checkpoint: %v", err)
	}
	if state.ArmingState != ArmingSafe {
		t.Fatalf("unexpected checkpoint state: %+v", state)
	}
}

func TestRuntimeIgnoresDuplicateCandidateEventAfterRestart(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)
	input := market.BarClosedEvent{
		EventIDValue: "bar-2",
		SymbolValue:  "MSTRUSDT",
		Interval:     "1m",
		Ts:           time.Unix(1710000060, 0).UTC(),
		Close:        101,
	}
	raw, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("marshal market event: %v", err)
	}
	seq, err := store.AppendEvent(ctx, "market.public", input, raw)
	if err != nil {
		t.Fatalf("append event: %v", err)
	}
	duplicate := BuildCandidateEvent(Candidate{
		Symbol:   "MSTRUSDT",
		Interval: "1m",
		Ts:       input.Ts,
		Side:     core.Long,
		Score:    4.2,
		Entry:    101,
		Stop:     100,
		Target:   103,
		Reasons:  []string{"runtime"},
	})
	duplicateRaw, err := json.Marshal(duplicate)
	if err != nil {
		t.Fatalf("marshal duplicate candidate: %v", err)
	}
	if _, err := store.AppendEvent(ctx, "trader", duplicate, duplicateRaw); err != nil {
		t.Fatalf("seed duplicate candidate: %v", err)
	}

	runtime := NewRuntime(RuntimeConfig{
		ConsumerKey:   "traderd",
		CheckpointKey: "trader.runtime",
		Store:         store,
		Engine: NewEngine(Config{
			Strategy: &runtimeStrategy{signal: core.Signal{
				Side:    core.Long,
				Score:   4.2,
				Entry:   101,
				Stop:    100,
				Target:  103,
				Reasons: []string{"runtime"},
			}},
		}),
		Sources: []string{"market.public"},
	})

	if err := runtime.ProcessAvailable(ctx); err != nil {
		t.Fatalf("process available after restart: %v", err)
	}
	cursor, err := store.LoadConsumerCursor(ctx, "traderd")
	if err != nil {
		t.Fatalf("load cursor: %v", err)
	}
	if cursor != seq {
		t.Fatalf("unexpected cursor after restart: %d", cursor)
	}
}

func TestRuntimeDowngradesOnPositionMismatch(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)
	bar := market.BarClosedEvent{
		EventIDValue: "bar-mismatch",
		SymbolValue:  "MSTRUSDT",
		Interval:     "1m",
		Ts:           time.Unix(1710000240, 0).UTC(),
		Close:        104,
	}
	barRaw, err := json.Marshal(bar)
	if err != nil {
		t.Fatalf("marshal bar: %v", err)
	}
	barSeq, err := store.AppendEvent(ctx, "market.public", bar, barRaw)
	if err != nil {
		t.Fatalf("append bar: %v", err)
	}
	runtime := NewRuntime(RuntimeConfig{
		ConsumerKey:   "traderd",
		CheckpointKey: "trader.runtime",
		Store:         store,
		Engine: NewEngine(Config{
			ArmingState: ArmingArmed,
			Strategy:    &runtimeStrategy{signal: core.Signal{Side: core.Long, Score: 4.2, Entry: 104, Stop: 102, Target: 107, Reasons: []string{"runtime"}}},
		}),
		Sources:    []string{"market.public", "market.private"},
		Risk:       NewRiskEngine(RiskConfig{MaxLeverage: 3}),
		Reconciler: NewReconciler(),
		RunID:      "run-17",
	})
	if err := runtime.ProcessAvailable(ctx); err != nil {
		t.Fatalf("process bar: %v", err)
	}

	position := market.PositionEvent{
		EventIDValue: "position-1",
		SymbolValue:  "MSTRUSDT",
		Ts:           time.Unix(1710000245, 0).UTC(),
		Qty:          0.02,
	}
	positionRaw, err := json.Marshal(position)
	if err != nil {
		t.Fatalf("marshal position: %v", err)
	}
	if _, err := store.AppendEvent(ctx, "market.private", position, positionRaw); err != nil {
		t.Fatalf("append position: %v", err)
	}

	if err := runtime.ProcessAvailable(ctx); err != nil {
		t.Fatalf("process position: %v", err)
	}
	if runtime.engine.State().ArmingState != ArmingDegraded {
		t.Fatalf("unexpected arming state: %+v", runtime.engine.State())
	}
	events, err := store.ListEventsAfter(ctx, barSeq, 10, "trader")
	if err != nil {
		t.Fatalf("list trader events: %v", err)
	}
	found := false
	for _, event := range events {
		if event.Kind == "risk.state_changed" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected risk.state_changed event, got %+v", events)
	}
}

func newTestStore(t *testing.T) *memoryRuntimeStore {
	t.Helper()
	return newMemoryRuntimeStore()
}

type runtimeStrategy struct {
	signal core.Signal
}

func (strategy *runtimeStrategy) OnBar(symbol, interval string, bars []core.Bar) core.Signal {
	return strategy.signal
}



type memoryRuntimeStore struct {
	nextSeq     int64
	events      []EventEnvelope
	cursors     map[string]int64
	checkpoints map[string]EngineState
	eventIDs    map[string]bool
}

func newMemoryRuntimeStore() *memoryRuntimeStore {
	return &memoryRuntimeStore{
		cursors:     map[string]int64{},
		checkpoints: map[string]EngineState{},
		eventIDs:    map[string]bool{},
	}
}

func (store *memoryRuntimeStore) AppendEvent(ctx context.Context, source string, evt EventLog, raw []byte) (int64, error) {
	if store.eventIDs[evt.EventID()] {
		return 0, fmt.Errorf("UNIQUE constraint failed: event_log.event_id")
	}
	store.nextSeq++
	store.events = append(store.events, EventEnvelope{
		Seq:        store.nextSeq,
		Source:     source,
		EventID:    evt.EventID(),
		Symbol:     evt.Symbol(),
		Kind:       evt.Kind(),
		ExchangeTS: evt.EventTime(),
		ReceivedTS: evt.EventTime(),
		Payload:    append([]byte(nil), raw...),
	})
	store.eventIDs[evt.EventID()] = true
	return store.nextSeq, nil
}

func (store *memoryRuntimeStore) ListEventsAfter(ctx context.Context, afterSeq int64, limit int, sources ...string) ([]EventEnvelope, error) {
	filter := map[string]bool{}
	for _, source := range sources {
		filter[source] = true
	}
	out := make([]EventEnvelope, 0, limit)
	for _, event := range store.events {
		if event.Seq <= afterSeq {
			continue
		}
		if len(filter) != 0 && !filter[event.Source] {
			continue
		}
		out = append(out, event)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (store *memoryRuntimeStore) SaveConsumerCursor(ctx context.Context, consumer string, seq int64) error {
	store.cursors[consumer] = seq
	return nil
}

func (store *memoryRuntimeStore) LoadConsumerCursor(ctx context.Context, consumer string) (int64, error) {
	return store.cursors[consumer], nil
}

func (store *memoryRuntimeStore) SaveCheckpoint(ctx context.Context, shard string, state EngineState) error {
	store.checkpoints[shard] = state
	return nil
}

func (store *memoryRuntimeStore) LoadCheckpoint(ctx context.Context, shard string) (EngineState, error) {
	return store.checkpoints[shard], nil
}
