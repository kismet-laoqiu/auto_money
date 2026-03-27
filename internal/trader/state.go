package trader

type ArmingState string

type Phase string

const (
	ArmingBooting  ArmingState = "booting"
	ArmingSafe     ArmingState = "safe"
	ArmingArmed    ArmingState = "armed"
	ArmingDegraded ArmingState = "degraded"
	ArmingHalted   ArmingState = "halted"
)

const (
	PhaseFlat         Phase = "flat"
	PhaseWatching     Phase = "watching_context"
	PhaseProbeLong    Phase = "probe_long"
	PhaseBuildingLong Phase = "building_long"
	PhaseHoldingLong  Phase = "holding_long"
	PhaseDeRisking    Phase = "de_risking"
	PhaseExitOnly     Phase = "exit_only"
)

type SymbolState struct {
	Phase         Phase   `json:"phase,omitempty"`
	Tranches      int     `json:"tranches,omitempty"`
	ContextOK     bool    `json:"context_ok,omitempty"`
	RSI14         float64 `json:"rsi14,omitempty"`
	NeedleDropPct float64 `json:"needle_drop_pct,omitempty"`
	ReclaimPct    float64 `json:"reclaim_pct,omitempty"`
}

type EngineState struct {
	ArmingState ArmingState            `json:"arming_state"`
	Symbols     map[string]SymbolState `json:"symbols,omitempty"`
}
