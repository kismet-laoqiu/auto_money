package trader

import "testing"

func TestNeedleCaptureRequiresContextAndReclaim(t *testing.T) {
	router := NewStrategyRouter()
	state := SymbolState{ContextOK: true, RSI14: 23, NeedleDropPct: 1.8, ReclaimPct: 0.9}
	profile := router.Select(state)
	if profile.Name() != "needle_capture" {
		t.Fatalf("unexpected profile: %s", profile.Name())
	}
}

func TestLeftAccumulationProfileIsFallbackWhenContextIsGood(t *testing.T) {
	router := NewStrategyRouter()
	state := SymbolState{ContextOK: true, RSI14: 34, NeedleDropPct: 0.6, ReclaimPct: 0.4}
	profile := router.Select(state)
	if profile.Name() != "left_accumulation" {
		t.Fatalf("unexpected profile: %s", profile.Name())
	}
}
