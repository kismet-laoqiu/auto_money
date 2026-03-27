package trader

type Profile interface {
	Name() string
}

type LeftAccumulationProfile struct{}

func (profile LeftAccumulationProfile) Name() string {
	return "left_accumulation"
}

type NeedleCaptureProfile struct{}

func (profile NeedleCaptureProfile) Name() string {
	return "needle_capture"
}

type StrategyRouter struct{}

func NewStrategyRouter() *StrategyRouter {
	return &StrategyRouter{}
}

func (router *StrategyRouter) Select(state SymbolState) Profile {
	if state.ContextOK && state.RSI14 > 0 && state.RSI14 < 28 && state.NeedleDropPct >= 1.5 && state.ReclaimPct >= 0.7 {
		return NeedleCaptureProfile{}
	}
	if state.ContextOK {
		return LeftAccumulationProfile{}
	}
	return LegacyRuleProfile{}
}
