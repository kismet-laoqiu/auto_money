package agent

import (
	"context"
	"os"
	"testing"
	"time"

	"quantlab/internal/core"
	"quantlab/internal/trader"
)

func TestRealRuntimeWritesArtifacts(t *testing.T) {
	if os.Getenv("RUN_OPENAI_REAL") != "1" {
		t.Skip("set RUN_OPENAI_REAL=1 to run real agent runtime tests")
	}
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("set OPENAI_API_KEY to run real agent runtime tests")
	}
	store := &memoryRuntimeStore{
		events: []trader.EventEnvelope{
			mustEnvelope(t, 1, "trader", trader.CandidateEvent{
				EventIDValue: "candidate-real-1",
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
				EventIDValue: "risk-real-1",
				SymbolValue:  "MSTRUSDT",
				Ts:           time.Unix(1710000001, 0).UTC(),
				From:         trader.ArmingArmed,
				To:           trader.ArmingDegraded,
				Reason:       "position_mismatch",
			}),
		},
		cursors: map[string]int64{},
	}
	runtime := NewRuntime(RuntimeConfig{
		Store:       store,
		Service:     NewService(NewHTTPResponsesClient(HTTPClientConfig{APIKey: apiKey, BaseURL: os.Getenv("OPENAI_BASE_URL")}), Config{Model: "gpt-5.4", Store: true}),
		MaxLeverage: 3,
		ArtifactDir: t.TempDir(),
	})
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	if err := runtime.ProcessAvailable(ctx); err != nil {
		t.Fatalf("process available: %v", err)
	}
	files, err := os.ReadDir(runtime.cfg.ArtifactDir)
	if err != nil {
		t.Fatalf("read artifact dir: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected two artifacts, got %d", len(files))
	}
}
