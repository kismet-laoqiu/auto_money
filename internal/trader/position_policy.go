package trader

type PolicyConfig struct {
	MaxTranches int
}

type AddDecision struct {
	Allow  bool
	Reason string
}

type PositionPolicy struct {
	cfg PolicyConfig
}

func NewPositionPolicy(cfg PolicyConfig) *PositionPolicy {
	return &PositionPolicy{cfg: cfg}
}

func (policy *PositionPolicy) DecideAdd(state SymbolState, candidate Candidate) AddDecision {
	if policy.cfg.MaxTranches > 0 && state.Tranches >= policy.cfg.MaxTranches {
		return AddDecision{Allow: false, Reason: "max_tranches"}
	}
	return AddDecision{Allow: true}
}
