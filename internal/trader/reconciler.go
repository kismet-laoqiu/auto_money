package trader

type PositionSnapshot struct {
	Symbol string
	Qty    float64
}

type ReconcileVerdict struct {
	Severity        string
	Reason          string
	NextArmingState ArmingState
}

type Reconciler struct{}

func NewReconciler() *Reconciler {
	return &Reconciler{}
}

func (reconciler *Reconciler) Compare(local, remote PositionSnapshot) ReconcileVerdict {
	if local.Symbol != remote.Symbol || local.Qty != remote.Qty {
		return ReconcileVerdict{Severity: "warn", Reason: "position_mismatch", NextArmingState: ArmingDegraded}
	}
	return ReconcileVerdict{Severity: "ok", NextArmingState: ArmingArmed}
}

func (reconciler *Reconciler) CompareModes(localMode, remoteMode string) ReconcileVerdict {
	if localMode != remoteMode {
		return ReconcileVerdict{Severity: "critical", Reason: "position_mode_mismatch", NextArmingState: ArmingHalted}
	}
	return ReconcileVerdict{Severity: "ok", NextArmingState: ArmingArmed}
}
