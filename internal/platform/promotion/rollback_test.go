package promotion

import (
	"strings"
	"testing"
	"time"
)

func TestRollbackMovesLiveRequestToRolledBack(t *testing.T) {
	now := time.Date(2026, 3, 29, 16, 25, 0, 0, time.UTC)
	request, err := Rollback(Request{
		ID:         "promo-1",
		StrategyID: "mstr-wave-fib",
		Version:    "v0.1.0",
		ConfigPath: "configs/demo-mstr-bundle.yaml",
		State:      StateLiveActive,
		CreatedAt:  now.Add(-time.Hour),
		UpdatedAt:  now.Add(-time.Hour),
	}, now, "operator rollback")
	if err != nil {
		t.Fatalf("rollback: %v", err)
	}
	if request.State != StateRolledBack {
		t.Fatalf("unexpected rolled back state: %+v", request)
	}
	if !strings.Contains(strings.Join(request.Details, "|"), "reason=operator rollback") {
		t.Fatalf("expected rollback reason in details: %+v", request.Details)
	}
}
