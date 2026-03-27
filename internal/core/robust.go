package core

import (
"sort"
"strings"
)

type RobustScore struct {
FinalScore             float64            `json:"final_score"`
MedianObjectiveScore   float64            `json:"median_objective_score"`
MinBucketMean          float64            `json:"min_bucket_mean"`
PositiveOOSReturnRatio float64            `json:"positive_oos_return_ratio"`
BucketMeans            map[string]float64 `json:"bucket_means"`
}

func BuildRobustScore(reports []Report) RobustScore {
result := RobustScore{BucketMeans: map[string]float64{"crypto": 0, "us_equity": 0, "commodity": 0}}
if len(reports) == 0 {
return result
}

scores := make([]float64, 0, len(reports))
bucketSums := map[string]float64{"crypto": 0, "us_equity": 0, "commodity": 0}
bucketCounts := map[string]int{"crypto": 0, "us_equity": 0, "commodity": 0}
positiveCount := 0

for _, report := range reports {
scores = append(scores, report.ObjectiveScore)
bucket := bucketForSymbol(report.Symbol)
bucketSums[bucket] += report.ObjectiveScore
bucketCounts[bucket]++
if report.OutOfSample.TotalReturn > 0 {
positiveCount++
}
}

result.MedianObjectiveScore = median(scores)
result.PositiveOOSReturnRatio = float64(positiveCount) / float64(len(reports))
result.MinBucketMean = 0
firstBucket := true
for _, bucket := range []string{"crypto", "us_equity", "commodity"} {
if bucketCounts[bucket] > 0 {
result.BucketMeans[bucket] = bucketSums[bucket] / float64(bucketCounts[bucket])
}
if firstBucket || result.BucketMeans[bucket] < result.MinBucketMean {
result.MinBucketMean = result.BucketMeans[bucket]
firstBucket = false
}
}

result.FinalScore = 0.55*result.MedianObjectiveScore + 0.25*result.MinBucketMean + 0.20*result.PositiveOOSReturnRatio
return result
}

func bucketForSymbol(symbol string) string {
symbol = strings.ToUpper(strings.TrimSpace(symbol))
switch {
case strings.HasSuffix(symbol, "USDT"):
return "crypto"
case symbol == "XAUUSD":
return "commodity"
default:
return "us_equity"
}
}

func median(values []float64) float64 {
if len(values) == 0 {
return 0
}
ordered := append([]float64(nil), values...)
sort.Float64s(ordered)
mid := len(ordered) / 2
if len(ordered)%2 == 1 {
return ordered[mid]
}
return (ordered[mid-1] + ordered[mid]) / 2
}
