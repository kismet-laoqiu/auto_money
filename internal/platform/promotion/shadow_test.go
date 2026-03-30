package promotion

import (
	"strings"
	"testing"
	"time"
)

func TestStartShadowAdvancesPromotionRequest(t *testing.T) {
	now := time.Date(2026, 3, 29, 16, 0, 0, 0, time.UTC)
	request, err := StartShadow(Request{
		ID:         "promo-1",
		StrategyID: "mstr-wave-fib",
		Version:    "v0.1.0",
		ConfigPath: "configs/demo-mstr-bundle.yaml",
		State:      StateBacktestPassed,
		CreatedAt:  now.Add(-time.Hour),
		UpdatedAt:  now.Add(-time.Hour),
	}, now)
	if err != nil {
		t.Fatalf("start shadow: %v", err)
	}
	if request.State != StateShadowRunning {
		t.Fatalf("unexpected shadow state: %+v", request)
	}
	if !strings.Contains(request.Title, "shadow started") || !strings.Contains(request.Summary, "shadow gate running") {
		t.Fatalf("unexpected shadow messaging: %+v", request)
	}
}

func TestPassShadowAdvancesPromotionRequest(t *testing.T) {
	now := time.Date(2026, 3, 29, 16, 5, 0, 0, time.UTC)
	request, err := PassShadow(Request{
		ID:         "promo-1",
		StrategyID: "mstr-wave-fib",
		Version:    "v0.1.0",
		ConfigPath: "configs/demo-mstr-bundle.yaml",
		State:      StateShadowRunning,
		CreatedAt:  now.Add(-time.Hour),
		UpdatedAt:  now.Add(-time.Hour),
	}, now)
	if err != nil {
		t.Fatalf("pass shadow: %v", err)
	}
	if request.State != StateShadowPassed {
		t.Fatalf("unexpected passed state: %+v", request)
	}
}
