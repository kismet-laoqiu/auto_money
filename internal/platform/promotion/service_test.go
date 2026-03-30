package promotion

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"quantlab/internal/backtest"
	sqlitepkg "quantlab/internal/store/sqlite"
)

func TestServiceCreatesAndReplaysPromotionLifecycle(t *testing.T) {
	store, err := sqlitepkg.NewStore(filepath.Join(t.TempDir(), "promotion.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	now := time.Date(2026, 3, 29, 15, 0, 0, 0, time.UTC)
	service := NewService(Config{
		Store:     store,
		Backtests: stubBacktestRunner{result: backtest.Result{GeneratedAt: now, ObjectiveScore: 0.86, FinalScore: 0.91}},
		Now:       func() time.Time { return now },
	})

	request, err := service.CreateRequest(context.Background(), CreateInput{
		StrategyID: "mstr-wave-fib",
		Version:    "v0.1.0",
			ConfigPath: "configs/live.yaml",
	})
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	if request.State != StateBacktestPassed || request.FinalScore != 0.91 {
		t.Fatalf("unexpected request after create: %+v", request)
	}

	if _, err := service.StartShadow(context.Background(), request.ID); err != nil {
		t.Fatalf("start shadow: %v", err)
	}
	if _, err := service.PassShadow(context.Background(), request.ID); err != nil {
		t.Fatalf("pass shadow: %v", err)
	}
	if _, err := service.StartCanary(context.Background(), request.ID); err != nil {
		t.Fatalf("start canary: %v", err)
	}
	request, err = service.Approve(context.Background(), request.ID)
	if err != nil {
		t.Fatalf("approve: %v", err)
	}
	if request.State != StateLiveActive {
		t.Fatalf("unexpected approved state: %+v", request)
	}

	got, err := service.Get(context.Background(), request.ID)
	if err != nil {
		t.Fatalf("get request: %v", err)
	}
	if got.State != StateLiveActive || got.StrategyID != "mstr-wave-fib" {
		t.Fatalf("unexpected replayed request: %+v", got)
	}
}

func TestServiceCanaryDegradedEventStaysNotifierCompatible(t *testing.T) {
	store, err := sqlitepkg.NewStore(filepath.Join(t.TempDir(), "promotion.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	now := time.Date(2026, 3, 29, 15, 0, 0, 0, time.UTC)
	service := NewService(Config{
		Store:     store,
		Backtests: stubBacktestRunner{result: backtest.Result{GeneratedAt: now, ObjectiveScore: 0.86, FinalScore: 0.91}},
		Now:       func() time.Time { return now },
	})
	request, err := service.CreateRequest(context.Background(), CreateInput{
		StrategyID: "mstr-wave-fib",
		Version:    "v0.1.1",
			ConfigPath: "configs/live.yaml",
	})
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	if _, err := service.StartShadow(context.Background(), request.ID); err != nil {
		t.Fatalf("start shadow: %v", err)
	}
	if _, err := service.PassShadow(context.Background(), request.ID); err != nil {
		t.Fatalf("pass shadow: %v", err)
	}
	if _, err := service.StartCanary(context.Background(), request.ID); err != nil {
		t.Fatalf("start canary: %v", err)
	}
	if _, err := service.DegradeCanary(context.Background(), request.ID, "shadow/live mismatch"); err != nil {
		t.Fatalf("degrade canary: %v", err)
	}

	events, err := store.ListEventsAfter(context.Background(), 0, 32, "platform")
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	var found bool
	for _, event := range events {
		if event.Kind != "promotion.canary_degraded" {
			continue
		}
		var payload struct {
			PromotionID string   `json:"promotion_id"`
			State       State    `json:"state"`
			Title       string   `json:"title"`
			Summary     string   `json:"summary"`
			Details     []string `json:"details"`
		}
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			t.Fatalf("decode canary degraded payload: %v", err)
		}
		if payload.PromotionID != request.ID || payload.State != StateCanaryDegraded {
			t.Fatalf("unexpected canary degraded payload: %+v", payload)
		}
		if payload.Title == "" || payload.Summary == "" || len(payload.Details) == 0 {
			t.Fatalf("expected notifier-compatible fields, got %+v", payload)
		}
		found = true
	}
	if !found {
		t.Fatalf("expected promotion.canary_degraded event")
	}
}

type stubBacktestRunner struct {
	result backtest.Result
	err    error
}

func (runner stubBacktestRunner) RunConfig(_ context.Context, _ string, _ bool) (backtest.Result, error) {
	return runner.result, runner.err
}
