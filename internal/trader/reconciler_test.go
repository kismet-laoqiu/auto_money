package trader

import "testing"

func TestReconcilerDowngradesToDegradedOnPositionMismatch(t *testing.T) {
	rec := NewReconciler()
	local := PositionSnapshot{Symbol: "BTCUSDT", Qty: 0.01}
	remote := PositionSnapshot{Symbol: "BTCUSDT", Qty: 0.02}
	verdict := rec.Compare(local, remote)
	if verdict.NextArmingState != ArmingDegraded {
		t.Fatalf("unexpected verdict: %+v", verdict)
	}
}

func TestReconcilerHaltsOnModeMismatch(t *testing.T) {
	rec := NewReconciler()
	verdict := rec.CompareModes("one_way_mode", "hedge_mode")
	if verdict.NextArmingState != ArmingHalted {
		t.Fatalf("unexpected verdict: %+v", verdict)
	}
}
