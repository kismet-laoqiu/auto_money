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
	Phase    Phase `json:"phase,omitempty"`
	Tranches int   `json:"tranches,omitempty"`
}

type EngineState struct {
	ArmingState ArmingState            `json:"arming_state"`
	Symbols     map[string]SymbolState `json:"symbols,omitempty"`
}
