package trader

import "testing"

func TestPositionPolicyLimitsTrancheCount(t *testing.T) {
	policy := NewPositionPolicy(PolicyConfig{MaxTranches: 3})
	state := SymbolState{Phase: PhaseBuildingLong, Tranches: 3}
	decision := policy.DecideAdd(state, Candidate{Symbol: "BTCUSDT", Score: 4.2})
	if decision.Allow {
		t.Fatalf("expected add to be blocked at max tranches")
	}
	if decision.Reason != "max_tranches" {
		t.Fatalf("unexpected decision: %+v", decision)
	}
}
