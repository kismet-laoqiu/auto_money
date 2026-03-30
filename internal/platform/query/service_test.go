package query

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"quantlab/internal/config"
	"quantlab/internal/core"
)

func TestServiceBarsFromConfigUsesNamedDataset(t *testing.T) {
	root := t.TempDir()
	cfgPath := filepath.Join(root, "query.yaml")
	if err := os.WriteFile(cfgPath, []byte(`
cache_dir: `+filepath.Join(root, "cache")+`
artifact_dir: `+filepath.Join(root, "artifacts")+`
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
  - name: mstr_demo
    provider: bitget
    symbol: MSTRUSDT
    interval: 1m
    limit: 20
`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	service := NewService(Config{
		Loader: fakeLoader{dataset: core.Dataset{
			Name:     "mstr_demo",
			Provider: "bitget",
			Symbol:   "MSTRUSDT",
			Interval: "1m",
			Bars:     sampleBars(40),
		}},
	})

	result, err := service.BarsFromConfig(context.Background(), cfgPath, "mstr_demo", false)
	if err != nil {
		t.Fatalf("bars from config: %v", err)
	}
	if result.Name != "mstr_demo" || result.Symbol != "MSTRUSDT" || len(result.Bars) != 40 {
		t.Fatalf("unexpected bars result: %+v", result)
	}
}

func TestServiceFeaturesFromConfigComputesLatestFeatureSet(t *testing.T) {
	root := t.TempDir()
	cfgPath := filepath.Join(root, "query.yaml")
	if err := os.WriteFile(cfgPath, []byte(`
cache_dir: `+filepath.Join(root, "cache")+`
artifact_dir: `+filepath.Join(root, "artifacts")+`
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
  - name: mstr_demo
    provider: bitget
    symbol: MSTRUSDT
    interval: 1m
    limit: 20
`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	service := NewService(Config{
		Loader: fakeLoader{dataset: core.Dataset{
			Name:     "mstr_demo",
			Provider: "bitget",
			Symbol:   "MSTRUSDT",
			Interval: "1m",
			Bars:     sampleBars(40),
		}},
	})

	result, err := service.FeaturesFromConfig(context.Background(), cfgPath, "mstr_demo", false, 0)
	if err != nil {
		t.Fatalf("features from config: %v", err)
	}
	if result.Symbol != "MSTRUSDT" || result.Index != 39 {
		t.Fatalf("unexpected features result: %+v", result)
	}
	if result.Features.Trigger.CloseLocationValue == 0 && result.Features.Regime.RegimeScore == 0 {
		t.Fatalf("expected computed feature values, got %+v", result.Features)
	}
}

func TestServiceFeaturesFromConfigSupportsStrategyBundle(t *testing.T) {
	root := t.TempDir()
	rel, err := filepath.Rel(root, bundleFixturePath(t))
	if err != nil {
		t.Fatalf("relative bundle path: %v", err)
	}
	cfgPath := filepath.Join(root, "query-bundle.yaml")
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

	result, err := service.FeaturesFromConfig(context.Background(), cfgPath, "mstrusdt_demo_replay", false, 0)
	if err != nil {
		t.Fatalf("features from bundle config: %v", err)
	}
	if result.Name != "mstrusdt_demo_replay" || result.Symbol != "MSTRUSDT" {
		t.Fatalf("unexpected bundle features result: %+v", result)
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
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", "strategies", "mstr-wave-fib", "versions", "v0.1.0"))
}
