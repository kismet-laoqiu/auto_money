package agent

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"quantlab/internal/core"
	"quantlab/internal/trader"
)

func TestRuntimeProcessesCandidateAndRiskEvents(t *testing.T) {
	store := &memoryRuntimeStore{
		events: []trader.EventEnvelope{
			mustEnvelope(t, 1, "trader", trader.CandidateEvent{
				EventIDValue: "candidate-1",
				SymbolValue:  "MSTRUSDT",
				Interval:     "1m",
				Ts:           time.Unix(1710000000, 0).UTC(),
				Side:         core.Long,
				Score:        4.2,
				Entry:        100,
				Stop:         99,
				Target:       104,
				Reasons:      []string{"momentum"},
			}),
			mustEnvelope(t, 2, "trader", trader.RiskEvent{
				EventIDValue: "risk-1",
				SymbolValue:  "MSTRUSDT",
				Ts:           time.Unix(1710000001, 0).UTC(),
				From:         trader.ArmingArmed,
				To:           trader.ArmingDegraded,
				Reason:       "position_mismatch",
			}),
		},
		cursors: map[string]int64{},
	}
	client := &fakeResponsesClient{
		createResponse: CreateResponse{ID: "resp-1", Status: "completed", Output: []OutputItem{{Content: []ContentPart{{Text: "ok"}}}}},
	}
	runtime := NewRuntime(RuntimeConfig{
		Store:       store,
		Service:     NewService(client, Config{Model: "gpt-5.4", Store: true}),
		ArtifactDir: t.TempDir(),
		MaxLeverage: 3,
	})
	if err := runtime.ProcessAvailable(context.Background()); err != nil {
		t.Fatalf("process available: %v", err)
	}
	if store.cursors["agentd"] != 2 {
		t.Fatalf("unexpected cursor: %+v", store.cursors)
	}
	if client.createCalls != 2 {
		t.Fatalf("expected two foreground calls, got %+v", client)
	}
	files, err := os.ReadDir(runtime.cfg.ArtifactDir)
	if err != nil {
		t.Fatalf("read artifact dir: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected two artifacts, got %d", len(files))
	}
	body, err := os.ReadFile(filepath.Join(runtime.cfg.ArtifactDir, files[0].Name()))
	if err != nil {
		t.Fatalf("read artifact: %v", err)
	}
	if !json.Valid(body) {
		t.Fatalf("artifact must be json: %s", body)
	}
}

func mustEnvelope(t *testing.T, seq int64, source string, event any) trader.EventEnvelope {
	t.Helper()
	body, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal envelope: %v", err)
	}
	typed := event.(interface {
		EventID() string
		Symbol() string
		EventTime() time.Time
		Kind() string
	})
	return trader.EventEnvelope{
		Seq:        seq,
		Source:     source,
		EventID:    typed.EventID(),
		Symbol:     typed.Symbol(),
		Kind:       typed.Kind(),
		ExchangeTS: typed.EventTime(),
		ReceivedTS: typed.EventTime(),
		Payload:    body,
	}
}

type memoryRuntimeStore struct {
	events   []trader.EventEnvelope
	cursors  map[string]int64
}

func (store *memoryRuntimeStore) ListEventsAfter(_ context.Context, afterSeq int64, limit int, sources ...string) ([]trader.EventEnvelope, error) {
	allowed := map[string]bool{}
	for _, source := range sources {
		allowed[source] = true
	}
	out := make([]trader.EventEnvelope, 0, len(store.events))
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

func (store *memoryRuntimeStore) SaveConsumerCursor(_ context.Context, consumer string, seq int64) error {
	store.cursors[consumer] = seq
	return nil
}

func (store *memoryRuntimeStore) LoadConsumerCursor(_ context.Context, consumer string) (int64, error) {
	return store.cursors[consumer], nil
}
