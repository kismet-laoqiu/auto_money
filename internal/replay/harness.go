package replay

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"quantlab/internal/market"
	"quantlab/internal/trader"
)

type Report struct {
	EventCount     int                `json:"event_count"`
	CommandCount   int                `json:"command_count"`
	CandidateCount int                `json:"candidate_count"`
	FinalState     trader.EngineState `json:"final_state"`
	Simulation     Summary            `json:"simulation"`
}

type Harness struct {
	engine    *trader.Engine
	simulator Simulator
}

func NewHarness(engine *trader.Engine, simulator Simulator) *Harness {
	return &Harness{engine: engine, simulator: simulator}
}

func (h *Harness) Run(events []market.MarketEvent) (Report, error) {
	store := newHarnessStore()
	ctx := context.Background()
	for _, event := range events {
		body, err := json.Marshal(event)
		if err != nil {
			return Report{}, err
		}
		if _, err := store.AppendEvent(ctx, sourceForReplayEvent(event), replayEventAdapter{event}, body); err != nil {
			return Report{}, err
		}
	}
	runtime := trader.NewRuntime(trader.RuntimeConfig{
		Store:      store,
		Engine:     h.engine,
		Sources:    []string{"market.public", "market.private"},
		Reconciler: trader.NewReconciler(),
	})
	if err := runtime.ProcessAvailable(ctx); err != nil {
		return Report{}, err
	}
	report := Report{EventCount: len(events), FinalState: h.engine.State()}
	outputs, err := store.ListEventsAfter(ctx, 0, 4096, "trader")
	if err != nil {
		return Report{}, err
	}
	report.CommandCount = len(outputs)
	for _, env := range outputs {
		if env.Kind != "candidate.created" {
			continue
		}
		report.CandidateCount++
		var candidate trader.CandidateEvent
		if err := json.Unmarshal(env.Payload, &candidate); err != nil {
			return Report{}, err
		}
		if h.simulator != nil {
			if err := h.simulator.Apply(candidateToCommand(candidate)); err != nil {
				return Report{}, err
			}
		}
	}
	if h.simulator != nil {
		report.Simulation = h.simulator.Snapshot()
	}
	return report, nil
}

func (h *Harness) RunAndWrite(events []market.MarketEvent, artifactDir string) (Report, error) {
	report, err := h.Run(events)
	if err != nil {
		return Report{}, err
	}
	if err := os.MkdirAll(artifactDir, 0o755); err != nil {
		return Report{}, err
	}
	body, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return Report{}, err
	}
	if err := os.WriteFile(filepath.Join(artifactDir, "replay-report.json"), append(body, '\n'), 0o644); err != nil {
		return Report{}, err
	}
	return report, nil
}

func sourceForReplayEvent(event market.MarketEvent) string {
	switch event.Kind() {
	case "order_update", "order_fill", "position_snapshot", "position_update", "account_snapshot", "account_update":
		return "market.private"
	default:
		return "market.public"
	}
}

func candidateToCommand(event trader.CandidateEvent) trader.Candidate {
	return trader.Candidate{
		Symbol:   event.SymbolValue,
		Interval: event.Interval,
		Ts:       event.Ts,
		Side:     event.Side,
		Score:    event.Score,
		Entry:    event.Entry,
		Stop:     event.Stop,
		Target:   event.Target,
		Reasons:  append([]string(nil), event.Reasons...),
	}
}

type replayEventAdapter struct {
	event market.MarketEvent
}

func (adapter replayEventAdapter) Symbol() string          { return adapter.event.Symbol() }
func (adapter replayEventAdapter) EventTime() time.Time    { return adapter.event.EventTime() }
func (adapter replayEventAdapter) Kind() string            { return adapter.event.Kind() }

func (adapter replayEventAdapter) EventID() string {
	if id := adapter.event.EventID(); id != "" {
		return id
	}
	return fmt.Sprintf("replay:%s:%s:%d", adapter.event.Kind(), adapter.event.Symbol(), adapter.event.EventTime().UnixNano())
}

type harnessStore struct {
	nextSeq     int64
	events      []trader.EventEnvelope
	cursors     map[string]int64
	checkpoints map[string]trader.EngineState
	eventIDs    map[string]bool
}

func newHarnessStore() *harnessStore {
	return &harnessStore{
		cursors:     map[string]int64{},
		checkpoints: map[string]trader.EngineState{},
		eventIDs:    map[string]bool{},
	}
}

func (store *harnessStore) AppendEvent(_ context.Context, source string, evt trader.EventLog, raw []byte) (int64, error) {
	if store.eventIDs[evt.EventID()] {
		return 0, fmt.Errorf("UNIQUE constraint failed: event_log.event_id")
	}
	store.nextSeq++
	store.events = append(store.events, trader.EventEnvelope{
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

func (store *harnessStore) ListEventsAfter(_ context.Context, afterSeq int64, limit int, sources ...string) ([]trader.EventEnvelope, error) {
	allowed := map[string]bool{}
	for _, source := range sources {
		allowed[source] = true
	}
	out := make([]trader.EventEnvelope, 0, limit)
	for _, event := range store.events {
		if event.Seq <= afterSeq {
			continue
		}
		if len(allowed) != 0 && !allowed[event.Source] {
			continue
		}
		out = append(out, event)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (store *harnessStore) SaveConsumerCursor(_ context.Context, consumer string, seq int64) error {
	store.cursors[consumer] = seq
	return nil
}

func (store *harnessStore) LoadConsumerCursor(_ context.Context, consumer string) (int64, error) {
	return store.cursors[consumer], nil
}

func (store *harnessStore) SaveCheckpoint(_ context.Context, shard string, state trader.EngineState) error {
	store.checkpoints[shard] = state
	return nil
}

func (store *harnessStore) LoadCheckpoint(_ context.Context, shard string) (trader.EngineState, error) {
	return store.checkpoints[shard], nil
}
