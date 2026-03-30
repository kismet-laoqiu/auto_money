package backtest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"quantlab/internal/core"
)

func TestReportWriterBuildsSummaryArtifacts(t *testing.T) {
	now := time.Unix(1710000000, 0).UTC()
	writer := NewReportWriter(ReportWriterConfig{Now: func() time.Time { return now }})
	reports := []core.Report{
		{
			Name:           "btc_test",
			Provider:       "bitget",
			Symbol:         "BTCUSDT",
			Interval:       "1m",
			Bars:           40,
			ObjectiveScore: 0.8,
			OutOfSample:    core.Stats{TotalReturn: 0.12, Sharpe: 1.1, Calmar: 0.9, MaxDrawdown: 0.08, Trades: 4},
		},
		{
			Name:           "mstr_test",
			Provider:       "polygon",
			Symbol:         "MSTR",
			Interval:       "1d",
			Bars:           60,
			ObjectiveScore: 0.6,
			OutOfSample:    core.Stats{TotalReturn: -0.02, Sharpe: 0.4, Calmar: 0.2, MaxDrawdown: 0.10, Trades: 2},
		},
	}

	result := writer.Build(reports)
	if result.GeneratedAt != now {
		t.Fatalf("unexpected generated_at: %s", result.GeneratedAt)
	}
	if result.MetricName != "final_score" {
		t.Fatalf("unexpected metric name: %q", result.MetricName)
	}
	if len(result.Reports) != 2 || result.FinalScore != result.Robust.FinalScore {
		t.Fatalf("unexpected result payload: %+v", result)
	}

	artifactDir := t.TempDir()
	if err := writer.WriteArtifacts(artifactDir, result); err != nil {
		t.Fatalf("write summary artifacts: %v", err)
	}
	jsonBody, err := os.ReadFile(filepath.Join(artifactDir, "backtest-result.json"))
	if err != nil {
		t.Fatalf("read json artifact: %v", err)
	}
	if !strings.Contains(string(jsonBody), `"metric_name": "final_score"`) {
		t.Fatalf("json artifact missing metric_name: %s", jsonBody)
	}
	markdownBody, err := os.ReadFile(filepath.Join(artifactDir, "backtest-report.md"))
	if err != nil {
		t.Fatalf("read markdown artifact: %v", err)
	}
	text := string(markdownBody)
	if !strings.Contains(text, "# Backtest Report") || !strings.Contains(text, "btc_test") || !strings.Contains(text, "mstr_test") {
		t.Fatalf("markdown artifact missing summary: %s", text)
	}
}
