package core

import (
"math"
"testing"
)

func TestBuildRobustScore(t *testing.T) {
reports := []Report{
{Symbol: "BTCUSDT", ObjectiveScore: 1.0, OutOfSample: Stats{TotalReturn: 0.10}},
{Symbol: "ETHUSDT", ObjectiveScore: 0.4, OutOfSample: Stats{TotalReturn: -0.02}},
{Symbol: "CRCL", ObjectiveScore: 0.6, OutOfSample: Stats{TotalReturn: 0.05}},
{Symbol: "XAUUSD", ObjectiveScore: -0.2, OutOfSample: Stats{TotalReturn: 0.01}},
}

got := BuildRobustScore(reports)
if math.Abs(got.MedianObjectiveScore-0.5) > 1e-9 {
t.Fatalf("unexpected median: %.6f", got.MedianObjectiveScore)
}
if math.Abs(got.BucketMeans["crypto"]-0.7) > 1e-9 {
t.Fatalf("unexpected crypto mean: %.6f", got.BucketMeans["crypto"])
}
if math.Abs(got.BucketMeans["us_equity"]-0.6) > 1e-9 {
t.Fatalf("unexpected us_equity mean: %.6f", got.BucketMeans["us_equity"])
}
if math.Abs(got.BucketMeans["commodity"]-(-0.2)) > 1e-9 {
t.Fatalf("unexpected commodity mean: %.6f", got.BucketMeans["commodity"])
}
if math.Abs(got.PositiveOOSReturnRatio-0.75) > 1e-9 {
t.Fatalf("unexpected positive ratio: %.6f", got.PositiveOOSReturnRatio)
}
if math.Abs(got.FinalScore-0.375) > 1e-9 {
t.Fatalf("unexpected final score: %.6f", got.FinalScore)
}
}

func TestBuildRobustScorePenalizesMissingBucketCoverage(t *testing.T) {
got := BuildRobustScore([]Report{{Symbol: "BTCUSDT", ObjectiveScore: 1.0, OutOfSample: Stats{TotalReturn: 0.1}}})
if got.MinBucketMean != 0 {
t.Fatalf("expected missing buckets to keep min bucket mean at 0, got %.6f", got.MinBucketMean)
}
if math.Abs(got.FinalScore-0.75) > 1e-9 {
t.Fatalf("unexpected final score with missing buckets: %.6f", got.FinalScore)
}
}
