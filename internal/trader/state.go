package trader

type ArmingState string

const (
	ArmingBooting  ArmingState = "booting"
	ArmingSafe     ArmingState = "safe"
	ArmingArmed    ArmingState = "armed"
	ArmingDegraded ArmingState = "degraded"
	ArmingHalted   ArmingState = "halted"
)

type EngineState struct {
	ArmingState ArmingState `json:"arming_state"`
}
