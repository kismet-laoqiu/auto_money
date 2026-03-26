package core

import (
	"testing"

	"quantlab/internal/config"
)

func TestTradeReturn(t *testing.T) {
	if got := tradeReturn(Long, 100, 110); got <= 0.09 || got >= 0.11 {
		t.Fatalf("expected long gain around 0.10, got %.4f", got)
	}
	if got := tradeReturn(Short, 100, 90); got <= 0.09 || got >= 0.11 {
		t.Fatalf("expected short gain around 0.10, got %.4f", got)
	}
}

func TestObjectiveScorePenalizesOverfit(t *testing.T) {
	objective := config.ObjectiveConfig{OverfitPenaltyWeight: 0.5, DrawdownWeight: 2, CalmarWeight: 0.35, PnLWeight: 0.2}
	good := ObjectiveScore(Stats{Sharpe: 1.2, TotalReturn: 0.3}, Stats{Sharpe: 1.0, Calmar: 0.8, CAGR: 0.2, MaxDrawdown: 0.1, TotalReturn: 0.18}, objective)
	bad := ObjectiveScore(Stats{Sharpe: 2.0, TotalReturn: 0.8}, Stats{Sharpe: 0.3, Calmar: 0.2, CAGR: 0.05, MaxDrawdown: 0.2, TotalReturn: 0.02}, objective)
	if good <= bad {
		t.Fatalf("expected better out-of-sample profile to score higher: good=%.4f bad=%.4f", good, bad)
	}
}

func TestObjectiveScoreRewardsHigherPnL(t *testing.T) {
	objective := config.ObjectiveConfig{DrawdownWeight: 2, CalmarWeight: 0.35, PnLWeight: 0.2}
	lowPnL := ObjectiveScore(Stats{}, Stats{Sharpe: 0.8, Calmar: 0.6, MaxDrawdown: 0.1, TotalReturn: 0.05}, objective)
	highPnL := ObjectiveScore(Stats{}, Stats{Sharpe: 0.8, Calmar: 0.6, MaxDrawdown: 0.1, TotalReturn: 0.25}, objective)
	if highPnL <= lowPnL {
		t.Fatalf("expected higher pnl to improve objective: high=%.4f low=%.4f", highPnL, lowPnL)
	}
}
