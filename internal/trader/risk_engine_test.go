package trader

import "testing"

func TestRiskEngineBlocksOverLeverage(t *testing.T) {
	risk := NewRiskEngine(RiskConfig{MaxLeverage: 3})
	risk.UpdateState(EngineState{ArmingState: ArmingArmed})
	verdict := risk.Check(EntryIntent{Symbol: "BTCUSDT", Notional: 900, Equity: 200})
	if verdict.Allow {
		t.Fatalf("expected leverage gate to block intent")
	}
	if verdict.Reason != "max_leverage" {
		t.Fatalf("unexpected verdict: %+v", verdict)
	}
}

func TestRiskEngineBlocksWhenNotArmed(t *testing.T) {
	risk := NewRiskEngine(RiskConfig{MaxLeverage: 3})
	risk.UpdateState(EngineState{ArmingState: ArmingSafe})
	verdict := risk.Check(EntryIntent{Symbol: "BTCUSDT", Notional: 100, Equity: 200})
	if verdict.Allow {
		t.Fatalf("expected arming gate to block intent")
	}
	if verdict.Reason != "not_armed" {
		t.Fatalf("unexpected verdict: %+v", verdict)
	}
}
