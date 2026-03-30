package scoring

import (
	"math"
	"testing"

	"quantlab/internal/core"
)

func TestBuildFinalScoreMatchesRobustFormula(t *testing.T) {
	reports := []core.Report{
		{Symbol: "BTCUSDT", ObjectiveScore: 1.0, OutOfSample: core.Stats{TotalReturn: 0.10}},
		{Symbol: "ETHUSDT", ObjectiveScore: 0.4, OutOfSample: core.Stats{TotalReturn: -0.02}},
		{Symbol: "CRCL", ObjectiveScore: 0.6, OutOfSample: core.Stats{TotalReturn: 0.05}},
		{Symbol: "XAUUSD", ObjectiveScore: -0.2, OutOfSample: core.Stats{TotalReturn: 0.01}},
	}

	got := BuildFinalScore(reports)
	if math.Abs(got.MedianObjectiveScore-0.5) > 1e-9 {
		t.Fatalf("unexpected median objective score: %.6f", got.MedianObjectiveScore)
	}
	if math.Abs(got.FinalScore-0.375) > 1e-9 {
		t.Fatalf("unexpected final score: %.6f", got.FinalScore)
	}
}
