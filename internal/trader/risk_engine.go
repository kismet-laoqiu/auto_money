package trader

type RiskConfig struct {
	MaxLeverage int
}

type EntryIntent struct {
	Symbol   string
	Notional float64
	Equity   float64
}

type Verdict struct {
	Allow  bool
	Reason string
}

type RiskEngine struct {
	cfg   RiskConfig
	state EngineState
}

func NewRiskEngine(cfg RiskConfig) *RiskEngine {
	return &RiskEngine{cfg: cfg, state: EngineState{ArmingState: ArmingSafe}}
}

func (engine *RiskEngine) UpdateState(state EngineState) {
	engine.state = state
}

func (engine *RiskEngine) Check(intent EntryIntent) Verdict {
	leverage := 0.0
	if intent.Equity > 0 {
		leverage = intent.Notional / intent.Equity
	}
	if engine.cfg.MaxLeverage > 0 && leverage > float64(engine.cfg.MaxLeverage) {
		return Verdict{Allow: false, Reason: "max_leverage"}
	}
	if engine.state.ArmingState != ArmingArmed {
		return Verdict{Allow: false, Reason: "not_armed"}
	}
	return Verdict{Allow: true}
}
