package backtest

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"quantlab/internal/core"
	"quantlab/internal/scoring"
)

type ReportWriterConfig struct {
	Now func() time.Time
}

type ReportWriter struct {
	now func() time.Time
}

func NewReportWriter(cfg ReportWriterConfig) *ReportWriter {
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	return &ReportWriter{now: cfg.Now}
}

func (writer *ReportWriter) Build(reports []core.Report) Result {
	aggregate := core.AggregateReports(reports)
	robust := scoring.BuildFinalScore(reports)
	return Result{
		GeneratedAt:    writer.now().UTC(),
		MetricName:     scoring.MetricName,
		ObjectiveScore: aggregate.ObjectiveScore,
		FinalScore:     robust.FinalScore,
		Robust:         robust,
		Aggregate:      aggregate.Metrics,
		Reports:        append([]core.Report(nil), reports...),
	}
}

func (writer *ReportWriter) JSON(result Result) ([]byte, error) {
	body, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(body, '\n'), nil
}

func (writer *ReportWriter) Markdown(result Result) []byte {
	var builder strings.Builder
	fmt.Fprintf(&builder, "# Backtest Report\n\n")
	fmt.Fprintf(&builder, "- metric_name: %s\n", result.MetricName)
	fmt.Fprintf(&builder, "- generated_at: %s\n", result.GeneratedAt.UTC().Format(time.RFC3339))
	fmt.Fprintf(&builder, "- objective_score: %.6f\n", result.ObjectiveScore)
	fmt.Fprintf(&builder, "- final_score: %.6f\n\n", result.FinalScore)
	fmt.Fprintf(&builder, "## Aggregate Metrics\n\n")
	for _, key := range sortedMetricKeys(result.Aggregate) {
		fmt.Fprintf(&builder, "- %s: %.6f\n", key, result.Aggregate[key])
	}
	if len(result.Aggregate) == 0 {
		fmt.Fprintf(&builder, "- none\n")
	}
	fmt.Fprintf(&builder, "\n")
	for _, report := range result.Reports {
		fmt.Fprintf(&builder, "## %s\n\n", reportName(report))
		fmt.Fprintf(&builder, "- provider: %s\n", report.Provider)
		fmt.Fprintf(&builder, "- symbol: %s\n", report.Symbol)
		fmt.Fprintf(&builder, "- interval: %s\n", report.Interval)
		fmt.Fprintf(&builder, "- bars: %d\n", report.Bars)
		fmt.Fprintf(&builder, "- objective_score: %.6f\n", report.ObjectiveScore)
		fmt.Fprintf(&builder, "- out_of_sample_return: %.6f\n", report.OutOfSample.TotalReturn)
		fmt.Fprintf(&builder, "- out_of_sample_sharpe: %.6f\n", report.OutOfSample.Sharpe)
		fmt.Fprintf(&builder, "- out_of_sample_calmar: %.6f\n", report.OutOfSample.Calmar)
		fmt.Fprintf(&builder, "- signal_count: %d\n\n", report.SignalCount)
	}
	return []byte(builder.String())
}

func (writer *ReportWriter) WriteArtifacts(root string, result Result) error {
	if root == "" {
		return nil
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return err
	}
	body, err := writer.JSON(result)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(root, "backtest-result.json"), body, 0o644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(root, "backtest-report.md"), writer.Markdown(result), 0o644)
}

func sortedMetricKeys(metrics map[string]float64) []string {
	keys := make([]string, 0, len(metrics))
	for key := range metrics {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func reportName(report core.Report) string {
	if report.Name != "" {
		return report.Name
	}
	if report.Symbol != "" {
		return report.Symbol
	}
	return "dataset"
}
