package backtest

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"quantlab/internal/config"
	"quantlab/internal/core"
	sqlitepkg "quantlab/internal/store/sqlite"
)

func TestServiceRunConfigBuildsReportsAndArtifacts(t *testing.T) {
	root := t.TempDir()
	cfgPath := filepath.Join(root, "backtest.yaml")
	if err := os.WriteFile(cfgPath, []byte(`
cache_dir: `+filepath.Join(root, "cache")+`
artifact_dir: `+filepath.Join(root, "artifacts")+`
objective:
  risk_free_rate: 0
  overfit_penalty_weight: 0.35
  drawdown_weight: 2
  calmar_weight: 0.35
  pnl_weight: 0.2
strategy:
  fast_sma: 2
  slow_sma: 3
  pivot_window: 1
  level_lookback: 5
  level_tolerance: 0.02
  fib_tolerance: 0.02
  signal_threshold: 1
  stop_atr: 1
  reward_risk: 2
  max_hold_bars: 5
  cooldown_bars: 1
  commission_bps: 0
  slippage_bps: 0
  atr_window: 2
datasets:
  - name: btc_test
    provider: bitget
    symbol: BTCUSDT
    interval: 1m
    limit: 20
`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	service := NewService(Config{
		Loader: fakeLoader{dataset: core.Dataset{
			Name:     "btc_test",
			Provider: "bitget",
			Symbol:   "BTCUSDT",
			Interval: "1m",
			Bars:     sampleBars(40),
		}},
	})

	result, err := service.RunConfig(context.Background(), cfgPath, false)
	if err != nil {
		t.Fatalf("run config: %v", err)
	}
	if len(result.Reports) != 1 || result.Reports[0].Symbol != "BTCUSDT" {
		t.Fatalf("unexpected reports: %+v", result.Reports)
	}
	if result.GeneratedAt.IsZero() {
		t.Fatalf("generated_at must be set: %+v", result)
	}
	if _, err := os.Stat(filepath.Join(root, "artifacts", "equity", "btc_test.csv")); err != nil {
		t.Fatalf("equity artifact missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "artifacts", "trades", "btc_test.csv")); err != nil {
		t.Fatalf("trades artifact missing: %v", err)
	}
}

func TestServiceRunConfigSupportsStrategyBundle(t *testing.T) {
	root := t.TempDir()
	rel, err := filepath.Rel(root, bundleFixturePath(t))
	if err != nil {
		t.Fatalf("relative bundle path: %v", err)
	}
	cfgPath := filepath.Join(root, "backtest-bundle.yaml")
	if err := os.WriteFile(cfgPath, []byte(`
cache_dir: `+filepath.Join(root, "cache")+`
artifact_dir: `+filepath.Join(root, "artifacts")+`
strategy_bundle_path: `+rel+`
`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	service := NewService(Config{
		Loader: fakeLoader{dataset: core.Dataset{
			Provider: "bitget",
			Symbol:   "MSTRUSDT",
			Interval: "1m",
			Bars:     sampleBars(80),
		}},
	})

	result, err := service.RunConfig(context.Background(), cfgPath, false)
	if err != nil {
		t.Fatalf("run config with bundle: %v", err)
	}
	if len(result.Reports) != 1 || result.Reports[0].Symbol != "MSTRUSDT" {
		t.Fatalf("unexpected bundle reports: %+v", result.Reports)
	}
	if _, err := os.Stat(filepath.Join(root, "artifacts", "equity", "mstrusdt_demo_replay.csv")); err != nil {
		t.Fatalf("bundle equity artifact missing: %v", err)
	}
}

type fakeLoader struct {
	dataset core.Dataset
}

func (loader fakeLoader) EnsureDataset(_ context.Context, _ string, spec config.DatasetConfig, _ bool) (core.Dataset, error) {
	dataset := loader.dataset
	dataset.Name = spec.Name
	dataset.Provider = spec.Provider
	dataset.Symbol = spec.Symbol
	dataset.Interval = spec.Interval
	return dataset, nil
}

func sampleBars(count int) []core.Bar {
	bars := make([]core.Bar, 0, count)
	start := time.Unix(1710000000, 0).UTC()
	price := 100.0
	for i := 0; i < count; i++ {
		closePrice := price + float64((i%5)-2)
		bars = append(bars, core.Bar{
			Time:   start.Add(time.Duration(i) * time.Minute),
			Open:   price,
			High:   closePrice + 1,
			Low:    closePrice - 1,
			Close:  closePrice,
			Volume: 10 + float64(i),
		})
		price = closePrice
	}
	return bars
}

func bundleFixturePath(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime caller unavailable")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "strategies", "mstr-wave-fib", "versions", "v0.1.0"))
}

func TestRunnerRunConfigWritesDatasetArtifacts(t *testing.T) {
	root := t.TempDir()
	cfgPath := writeRunnerConfig(t, root)
	writer := &fakeArtifactWriter{}
	runner := NewRunner(RunnerConfig{
		Loader: fakeLoader{dataset: core.Dataset{
			Provider: "bitget",
			Symbol:   "BTCUSDT",
			Interval: "1m",
			Bars:     sampleBars(40),
		}},
		Writer: writer,
	})

	run, err := runner.RunConfig(context.Background(), cfgPath, false)
	if err != nil {
		t.Fatalf("runner run config: %v", err)
	}
	if len(run.Reports) != 1 || run.Reports[0].Symbol != "BTCUSDT" {
		t.Fatalf("unexpected runner reports: %+v", run.Reports)
	}
	if len(writer.roots) != 1 || writer.roots[0] != filepath.Join(root, "artifacts") {
		t.Fatalf("unexpected artifact roots: %+v", writer.roots)
	}
	if len(writer.reports) != 1 || writer.reports[0].Symbol != "BTCUSDT" {
		t.Fatalf("unexpected artifact writes: %+v", writer.reports)
	}
}

func writeRunnerConfig(t *testing.T, root string) string {
	t.Helper()
	cfgPath := filepath.Join(root, "runner-backtest.yaml")
	if err := os.WriteFile(cfgPath, []byte(`
cache_dir: `+filepath.Join(root, "cache")+`
artifact_dir: `+filepath.Join(root, "artifacts")+`
objective:
  risk_free_rate: 0
  overfit_penalty_weight: 0.35
  drawdown_weight: 2
  calmar_weight: 0.35
  pnl_weight: 0.2
strategy:
  fast_sma: 2
  slow_sma: 3
  pivot_window: 1
  level_lookback: 5
  level_tolerance: 0.02
  fib_tolerance: 0.02
  signal_threshold: 1
  stop_atr: 1
  reward_risk: 2
  max_hold_bars: 5
  cooldown_bars: 1
  commission_bps: 0
  slippage_bps: 0
  atr_window: 2
datasets:
  - name: btc_test
    provider: bitget
    symbol: BTCUSDT
    interval: 1m
    limit: 20
`), 0o644); err != nil {
		t.Fatalf("write runner config: %v", err)
	}
	return cfgPath
}

type fakeArtifactWriter struct {
	roots   []string
	reports []core.Report
}

func (writer *fakeArtifactWriter) WriteReportArtifacts(root string, report core.Report) error {
	writer.roots = append(writer.roots, root)
	writer.reports = append(writer.reports, report)
	return nil
}

func TestServiceRunConfigArchivesCompletedRun(t *testing.T) {
	root := t.TempDir()
	store, err := sqlitepkg.NewStore(filepath.Join(root, "backtest.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	rel, err := filepath.Rel(root, bundleFixturePath(t))
	if err != nil {
		t.Fatalf("relative bundle path: %v", err)
	}
	cfgPath := filepath.Join(root, "archive-backtest.yaml")
	if err := os.WriteFile(cfgPath, []byte(`
cache_dir: `+filepath.Join(root, "cache")+`
artifact_dir: `+filepath.Join(root, "artifacts")+`
strategy_bundle_path: `+rel+`
`), 0o644); err != nil {
		t.Fatalf("write archive config: %v", err)
	}
	service := NewService(Config{
		Loader: fakeLoader{dataset: core.Dataset{
			Provider: "bitget",
			Symbol:   "MSTRUSDT",
			Interval: "1m",
			Bars:     sampleBars(80),
		}},
		Store: store,
		Now: func() time.Time {
			return time.Unix(1710003600, 0).UTC()
		},
	})

	result, err := service.RunConfig(context.Background(), cfgPath, false)
	if err != nil {
		t.Fatalf("run config with archive: %v", err)
	}
	events, err := store.ListEventsAfter(context.Background(), 0, 10, "backtest")
	if err != nil {
		t.Fatalf("list archive events: %v", err)
	}
	if len(events) != 1 || events[0].Kind != "backtest.run.completed" {
		t.Fatalf("unexpected archive events: %+v", events)
	}
	var archive RunArchiveEvent
	if err := json.Unmarshal(events[0].Payload, &archive); err != nil {
		t.Fatalf("decode archive payload: %v", err)
	}
	if archive.StrategyID != "mstr-wave-fib" || archive.BundleVersion != "v0.1.0" {
		t.Fatalf("unexpected archive identity: %+v", archive)
	}
	if archive.FeatureVersion != core.FeatureSetVersion {
		t.Fatalf("unexpected feature version: %+v", archive)
	}
	if archive.ConfigPath != cfgPath || archive.ArtifactDir != filepath.Join(root, "artifacts") {
		t.Fatalf("unexpected archive paths: %+v", archive)
	}
	if archive.FinalScore != result.FinalScore || archive.ObjectiveScore != result.ObjectiveScore || archive.MetricName != result.MetricName {
		t.Fatalf("unexpected archive scores: %+v result=%+v", archive, result)
	}
	if archive.DatasetHash == "" || len(archive.Datasets) != 1 || archive.Datasets[0].Hash == "" {
		t.Fatalf("unexpected dataset archive: %+v", archive)
	}
}
