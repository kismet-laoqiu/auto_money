package scoring

import "quantlab/internal/core"

const MetricName = "final_score"

func BuildFinalScore(reports []core.Report) core.RobustScore {
	return core.BuildRobustScore(reports)
}
