package core_test

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"quantlab/internal/adapters"
	"quantlab/internal/config"
	"quantlab/internal/core"
)

type snapshotCase struct {
	name   string
	symbol string
	date   string
	want   string
	build  func(core.Dataset, int, config.Config) any
}

var (
	cachedConfig   config.Config
	cachedDatasets map[string]core.Dataset
)

func TestWaveStructureSnapshots(t *testing.T) {
	runSnapshotCases(t, []snapshotCase{
		waveSnapshot("btc_2025_07_15", "BTCUSDT", "2025-07-15", `{
  "wave_up_score": 0,
  "wave_down_score": 0,
  "impulse_extension_ratio": 1.9607256280543919,
  "corrective_depth_ratio": 1.7099650608215227,
  "swing_overlap_ratio": 1,
  "structure_age_bars": 32,
  "valid_structure": true
}`),
		waveSnapshot("eth_2025_05_18", "ETHUSDT", "2025-05-18", `{
  "wave_up_score": 1.7,
  "wave_down_score": 0,
  "impulse_extension_ratio": 6.973209867816497,
  "corrective_depth_ratio": 0.8565066798614551,
  "swing_overlap_ratio": 1,
  "structure_age_bars": 18,
  "valid_structure": true
}`),
		waveSnapshot("crcl_2025_12_12", "CRCL", "2025-12-12", `{
  "wave_up_score": 0,
  "wave_down_score": 2,
  "impulse_extension_ratio": 2.2320801067826284,
  "corrective_depth_ratio": 0.674601071623441,
  "swing_overlap_ratio": 1,
  "structure_age_bars": 44,
  "valid_structure": true
}`),
	})
}

func TestLevelClusterSnapshots(t *testing.T) {
	runSnapshotCases(t, []snapshotCase{
		levelSnapshot("eth_2024_11_14", "ETHUSDT", "2024-11-14", `{
  "nearest_support_distance_pct": 0.16939102878452056,
  "nearest_resistance_distance_pct": 1,
  "support_cluster_count": 6,
  "resistance_cluster_count": 5,
  "support_bounce_count": 0,
  "resistance_bounce_count": 0,
  "zone_width_pct": 0.014329054995063432,
  "recent_breakout_flag": false
}`),
		levelSnapshot("xau_2025_06_27", "XAUUSD", "2025-06-27", `{
  "nearest_support_distance_pct": 0.006951715536044761,
  "nearest_resistance_distance_pct": 0.028973124007416272,
  "support_cluster_count": 7,
  "resistance_cluster_count": 5,
  "support_bounce_count": 0,
  "resistance_bounce_count": 0,
  "zone_width_pct": 0,
  "recent_breakout_flag": false
}`),
		levelSnapshot("btc_2025_06_27", "BTCUSDT", "2025-06-27", `{
  "nearest_support_distance_pct": 0.006003591486739667,
  "nearest_resistance_distance_pct": 0.006003591486739667,
  "support_cluster_count": 6,
  "resistance_cluster_count": 6,
  "support_bounce_count": 0,
  "resistance_bounce_count": 1,
  "zone_width_pct": 0.009110153717612897,
  "recent_breakout_flag": false
}`),
	})
}

func TestFibConfluenceSnapshots(t *testing.T) {
	runSnapshotCases(t, []snapshotCase{
		fibSnapshot("eth_2025_04_16", "ETHUSDT", "2025-04-16", `{
  "fib_382_distance_pct": 0.01673592705783873,
  "fib_500_distance_pct": 0.05952864045043559,
  "fib_618_distance_pct": 0.1023213538430323,
  "fib_zone_hit_count": 0,
  "fib_cluster_overlap_score": 0,
  "active_swing_direction": "down",
  "valid_fib_context": true
}`),
		fibSnapshot("crcl_2025_09_26", "CRCL", "2025-09-26", `{
  "fib_382_distance_pct": 0.04372186047165506,
  "fib_500_distance_pct": 0.0068509248485701234,
  "fib_618_distance_pct": 0.03002001077451481,
  "fib_zone_hit_count": 1,
  "fib_cluster_overlap_score": 0,
  "active_swing_direction": "up",
  "valid_fib_context": true
}`),
		fibSnapshot("xau_2025_05_29", "XAUUSD", "2025-05-29", `{
  "fib_382_distance_pct": 0.013651857299591293,
  "fib_500_distance_pct": 0.02235571416514945,
  "fib_618_distance_pct": 0.03105957103070774,
  "fib_zone_hit_count": 0,
  "fib_cluster_overlap_score": 0.3458495382991336,
  "active_swing_direction": "up",
  "valid_fib_context": true
}`),
	})
}

func TestPriceActionTriggerSnapshots(t *testing.T) {
	runSnapshotCases(t, []snapshotCase{
		triggerSnapshot("btc_2024_02_24", "BTCUSDT", "2024-02-24", `{
  "bullish_engulfing_flag": true,
  "bearish_engulfing_flag": false,
  "pin_bar_bull_score": 0,
  "pin_bar_bear_score": 0,
  "close_location_value": 0.8833962264150954,
  "range_expansion_ratio": 0.7131137975300506,
  "break_retest_flag": false,
  "trigger_quality_score": 0.85
}`),
		triggerSnapshot("eth_2024_03_14", "ETHUSDT", "2024-03-14", `{
  "bullish_engulfing_flag": false,
  "bearish_engulfing_flag": true,
  "pin_bar_bull_score": 0.42702791860545775,
  "pin_bar_bear_score": 0,
  "close_location_value": 0.5510799361066734,
  "range_expansion_ratio": 1.2451189307045667,
  "break_retest_flag": false,
  "trigger_quality_score": 0.9629129275148285
}`),
		triggerSnapshot("crcl_2025_07_18", "CRCL", "2025-07-18", `{
  "bullish_engulfing_flag": false,
  "bearish_engulfing_flag": true,
  "pin_bar_bull_score": 0,
  "pin_bar_bear_score": 0.2904648013096626,
  "close_location_value": 0.03186742643985047,
  "range_expansion_ratio": 1.8768791818870236,
  "break_retest_flag": false,
  "trigger_quality_score": 1.4405073134751238
}`),
	})
}

func TestVolumeConfirmationSnapshots(t *testing.T) {
	runSnapshotCases(t, []snapshotCase{
		volumeSnapshot("btc_2024_02_24", "BTCUSDT", "2024-02-24", `{
  "volume_zscore": -1.3498574678254542,
  "relative_volume_ratio": 0.46012904439802177,
  "pullback_volume_dryup_flag": true,
  "breakout_volume_confirmed": false,
  "volume_available": true
}`),
		volumeSnapshot("eth_2025_04_16", "ETHUSDT", "2025-04-16", `{
  "volume_zscore": -0.025349960466262422,
  "relative_volume_ratio": 0.9814051865730005,
  "pullback_volume_dryup_flag": false,
  "breakout_volume_confirmed": false,
  "volume_available": true
}`),
		volumeSnapshot("xau_2025_05_29", "XAUUSD", "2025-05-29", `{
  "volume_zscore": 0,
  "relative_volume_ratio": 0,
  "pullback_volume_dryup_flag": false,
  "breakout_volume_confirmed": false,
  "volume_available": false
}`),
	})
}

func TestRegimeSnapshots(t *testing.T) {
	runSnapshotCases(t, []snapshotCase{
		regimeSnapshot("btc_2025_07_15", "BTCUSDT", "2025-07-15", `{
  "trend_up_flag": true,
  "trend_down_flag": false,
  "range_flag": false,
  "high_vol_flag": false,
  "compression_flag": false,
  "regime_score": 1.6
}`),
		regimeSnapshot("eth_2025_04_16", "ETHUSDT", "2025-04-16", `{
  "trend_up_flag": false,
  "trend_down_flag": true,
  "range_flag": false,
  "high_vol_flag": true,
  "compression_flag": false,
  "regime_score": 2.3000000000000003
}`),
		regimeSnapshot("crcl_2025_06_20", "CRCL", "2025-06-20", `{
  "trend_up_flag": false,
  "trend_down_flag": false,
  "range_flag": false,
  "high_vol_flag": true,
  "compression_flag": false,
  "regime_score": 0.6499999999999999
}`),
		regimeSnapshot("xau_2025_07_08", "XAUUSD", "2025-07-08", `{
  "trend_up_flag": false,
  "trend_down_flag": false,
  "range_flag": true,
  "high_vol_flag": false,
  "compression_flag": false,
  "regime_score": 1.4
}`),
	})
}

func runSnapshotCases(t *testing.T, cases []snapshotCase) {
	t.Helper()
	cfg := loadSnapshotConfig(t)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dataset := loadSnapshotDataset(t, tc.symbol)
			idx := indexByDate(t, dataset.Bars, tc.date)
			assertSnapshot(t, tc.build(dataset, idx, cfg), tc.want)
		})
	}
}

func waveSnapshot(name, symbol, date, want string) snapshotCase {
	return snapshotCase{
		name:   name,
		symbol: symbol,
		date:   date,
		want:   want,
		build: func(dataset core.Dataset, idx int, cfg config.Config) any {
			return core.ExtractWaveStructureFeatures(dataset.Bars, idx, cfg.Strategy.PivotWindow, maxInt(cfg.Strategy.LevelLookback, cfg.Strategy.SlowSMA*2), 0.02)
		},
	}
}

func levelSnapshot(name, symbol, date, want string) snapshotCase {
	return snapshotCase{
		name:   name,
		symbol: symbol,
		date:   date,
		want:   want,
		build: func(dataset core.Dataset, idx int, cfg config.Config) any {
			return core.ExtractLevelClusterFeatures(dataset.Bars, idx, cfg.Strategy.PivotWindow, cfg.Strategy.LevelLookback, cfg.Strategy.LevelTolerance)
		},
	}
}

func fibSnapshot(name, symbol, date, want string) snapshotCase {
	return snapshotCase{
		name:   name,
		symbol: symbol,
		date:   date,
		want:   want,
		build: func(dataset core.Dataset, idx int, cfg config.Config) any {
			levels := core.ExtractLevelClusterFeatures(dataset.Bars, idx, cfg.Strategy.PivotWindow, cfg.Strategy.LevelLookback, cfg.Strategy.LevelTolerance)
			return core.ExtractFibConfluenceFeatures(dataset.Bars, idx, cfg.Strategy.PivotWindow, maxInt(cfg.Strategy.LevelLookback, cfg.Strategy.SlowSMA*2), cfg.Strategy.FibTolerance, levels)
		},
	}
}

func triggerSnapshot(name, symbol, date, want string) snapshotCase {
	return snapshotCase{
		name:   name,
		symbol: symbol,
		date:   date,
		want:   want,
		build: func(dataset core.Dataset, idx int, cfg config.Config) any {
			levels := core.ExtractLevelClusterFeatures(dataset.Bars, idx, cfg.Strategy.PivotWindow, cfg.Strategy.LevelLookback, cfg.Strategy.LevelTolerance)
			fib := core.ExtractFibConfluenceFeatures(dataset.Bars, idx, cfg.Strategy.PivotWindow, maxInt(cfg.Strategy.LevelLookback, cfg.Strategy.SlowSMA*2), cfg.Strategy.FibTolerance, levels)
			return core.ExtractPriceActionTriggerFeatures(dataset.Bars, idx, cfg.Strategy.ATRWindow, levels, fib)
		},
	}
}

func volumeSnapshot(name, symbol, date, want string) snapshotCase {
	return snapshotCase{
		name:   name,
		symbol: symbol,
		date:   date,
		want:   want,
		build: func(dataset core.Dataset, idx int, cfg config.Config) any {
			levels := core.ExtractLevelClusterFeatures(dataset.Bars, idx, cfg.Strategy.PivotWindow, cfg.Strategy.LevelLookback, cfg.Strategy.LevelTolerance)
			fib := core.ExtractFibConfluenceFeatures(dataset.Bars, idx, cfg.Strategy.PivotWindow, maxInt(cfg.Strategy.LevelLookback, cfg.Strategy.SlowSMA*2), cfg.Strategy.FibTolerance, levels)
			trigger := core.ExtractPriceActionTriggerFeatures(dataset.Bars, idx, cfg.Strategy.ATRWindow, levels, fib)
			return core.ExtractVolumeConfirmationFeatures(dataset.Bars, idx, maxInt(cfg.Strategy.ATRWindow*2, 20), trigger)
		},
	}
}

func regimeSnapshot(name, symbol, date, want string) snapshotCase {
	return snapshotCase{
		name:   name,
		symbol: symbol,
		date:   date,
		want:   want,
		build: func(dataset core.Dataset, idx int, cfg config.Config) any {
			levels := core.ExtractLevelClusterFeatures(dataset.Bars, idx, cfg.Strategy.PivotWindow, cfg.Strategy.LevelLookback, cfg.Strategy.LevelTolerance)
			wave := core.ExtractWaveStructureFeatures(dataset.Bars, idx, cfg.Strategy.PivotWindow, maxInt(cfg.Strategy.LevelLookback, cfg.Strategy.SlowSMA*2), 0.02)
			return core.ExtractRegimeTags(dataset.Bars, idx, cfg.Strategy.FastSMA, cfg.Strategy.SlowSMA, cfg.Strategy.ATRWindow, maxInt(cfg.Strategy.LevelLookback, cfg.Strategy.SlowSMA*2), levels, wave)
		},
	}
}

func loadSnapshotConfig(t *testing.T) config.Config {
	t.Helper()
	if cachedDatasets != nil {
		return cachedConfig
	}
	cfg, err := config.Load(filepath.Join("..", "..", "configs", "baseline.yaml"))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	cfg.CacheDir = filepath.Join("..", "..", cfg.CacheDir)
	cachedConfig = cfg
	cachedDatasets = make(map[string]core.Dataset, len(cfg.Datasets))
	client := adapters.NewClient()
	for _, spec := range cfg.Datasets {
		dataset, err := client.EnsureDataset(context.Background(), cfg.CacheDir, spec, false)
		if err != nil {
			t.Fatalf("load dataset %s: %v", spec.Symbol, err)
		}
		cachedDatasets[strings.ToUpper(spec.Symbol)] = dataset
	}
	return cachedConfig
}

func loadSnapshotDataset(t *testing.T, symbol string) core.Dataset {
	t.Helper()
	loadSnapshotConfig(t)
	dataset, ok := cachedDatasets[strings.ToUpper(symbol)]
	if !ok {
		t.Fatalf("dataset %s not loaded", symbol)
	}
	return dataset
}

func indexByDate(t *testing.T, bars []core.Bar, want string) int {
	t.Helper()
	for i := range bars {
		if bars[i].Time.Format("2006-01-02") == want {
			return i
		}
	}
	t.Fatalf("date %s not found", want)
	return -1
}

func assertSnapshot(t *testing.T, value any, want string) {
	t.Helper()
	body, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatalf("marshal snapshot: %v", err)
	}
	got := string(body)
	want = strings.TrimSpace(want)
	if got != want {
		t.Fatalf("snapshot mismatch\nwant:\n%s\n\ngot:\n%s", want, got)
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
