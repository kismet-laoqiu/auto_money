package promotion

import (
	"strings"
	"testing"
	"time"
)

func TestStartCanaryAdvancesPromotionRequest(t *testing.T) {
	now := time.Date(2026, 3, 29, 16, 10, 0, 0, time.UTC)
	request, err := StartCanary(Request{
		ID:         "promo-1",
		StrategyID: "mstr-wave-fib",
		Version:    "v0.1.0",
		ConfigPath: "configs/demo-mstr-bundle.yaml",
		State:      StateShadowPassed,
		CreatedAt:  now.Add(-time.Hour),
		UpdatedAt:  now.Add(-time.Hour),
	}, now)
	if err != nil {
		t.Fatalf("start canary: %v", err)
	}
	if request.State != StateCanaryRunning {
		t.Fatalf("unexpected canary state: %+v", request)
	}
}

func TestDegradeCanaryCarriesReasonIntoDetails(t *testing.T) {
	now := time.Date(2026, 3, 29, 16, 15, 0, 0, time.UTC)
	request, err := DegradeCanary(Request{
		ID:         "promo-1",
		StrategyID: "mstr-wave-fib",
		Version:    "v0.1.0",
		ConfigPath: "configs/demo-mstr-bundle.yaml",
		State:      StateCanaryRunning,
		CreatedAt:  now.Add(-time.Hour),
		UpdatedAt:  now.Add(-time.Hour),
	}, now, "shadow/live mismatch")
	if err != nil {
		t.Fatalf("degrade canary: %v", err)
	}
	if request.State != StateCanaryDegraded {
		t.Fatalf("unexpected degraded state: %+v", request)
	}
	if len(request.Details) == 0 || !strings.Contains(strings.Join(request.Details, "|"), "reason=shadow/live mismatch") {
		t.Fatalf("expected reason detail, got %+v", request.Details)
	}
}

func TestApprovePromotesRequestToLive(t *testing.T) {
	now := time.Date(2026, 3, 29, 16, 20, 0, 0, time.UTC)
	request, err := Approve(Request{
		ID:         "promo-1",
		StrategyID: "mstr-wave-fib",
		Version:    "v0.1.0",
		ConfigPath: "configs/demo-mstr-bundle.yaml",
		State:      StateCanaryRunning,
		CreatedAt:  now.Add(-time.Hour),
		UpdatedAt:  now.Add(-time.Hour),
	}, now)
	if err != nil {
		t.Fatalf("approve: %v", err)
	}
	if request.State != StateLiveActive || !strings.Contains(request.Summary, "live_active") {
		t.Fatalf("unexpected live request: %+v", request)
	}
}
