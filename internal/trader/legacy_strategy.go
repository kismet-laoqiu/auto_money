package trader

import (
	"quantlab/internal/config"
	"quantlab/internal/core"
)

type SignalFunc func(bars []core.Bar, idx int, cfg config.StrategyConfig) core.Signal

type LegacyRuleProfile struct {
	StrategyCfg config.StrategyConfig
	Evaluate    SignalFunc
}

func (profile LegacyRuleProfile) OnBar(symbol, interval string, bars []core.Bar) core.Signal {
	if len(bars) == 0 {
		return core.Signal{Side: core.Flat}
	}
	evaluate := profile.Evaluate
	if evaluate == nil {
		evaluate = core.EvaluateSignal
	}
	return evaluate(bars, len(bars)-1, profile.StrategyCfg)
}
