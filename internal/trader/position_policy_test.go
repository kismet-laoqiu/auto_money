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

func TestPositionPolicyUsesProbePhaseForNeedleCapture(t *testing.T) {
	policy := NewPositionPolicy(PolicyConfig{MaxTranches: 3})
	if got := policy.DesiredPhase(NeedleCaptureProfile{}); got != PhaseProbeLong {
		t.Fatalf("unexpected needle phase: %s", got)
	}
	if got := policy.DesiredPhase(LeftAccumulationProfile{}); got != PhaseWatching {
		t.Fatalf("unexpected accumulation phase: %s", got)
	}
}
