package trader

import (
	"fmt"

	"quantlab/internal/core"
	"quantlab/internal/strategybundle"
)

type BundleStrategy struct {
	Bundle strategybundle.Bundle
}

func (strategy BundleStrategy) Name() string {
	if strategy.Bundle.StrategyID == "" {
		return "bundle"
	}
	if strategy.Bundle.Version == "" {
		return strategy.Bundle.StrategyID
	}
	return fmt.Sprintf("%s@%s", strategy.Bundle.StrategyID, strategy.Bundle.Version)
}

func (strategy BundleStrategy) OnBar(symbol, interval string, bars []core.Bar) core.Signal {
	if len(bars) == 0 {
		return core.Signal{Side: core.Flat}
	}
	return core.EvaluateSignal(bars, len(bars)-1, strategy.Bundle.Strategy)
}
