package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadLiveConfigDefaults(t *testing.T) {
	cfg, err := Load("testdata/live-minimal.yaml")
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Live.Exchange.ProductType != "USDT-FUTURES" {
		t.Fatalf("unexpected product type: %s", cfg.Live.Exchange.ProductType)
	}
	if cfg.Live.Exchange.PositionMode != "one_way_mode" {
		t.Fatalf("unexpected position mode: %s", cfg.Live.Exchange.PositionMode)
	}
	if cfg.Live.Risk.MaxLeverage != 3 {
		t.Fatalf("unexpected leverage cap: %d", cfg.Live.Risk.MaxLeverage)
	}
	if cfg.Live.Runtime.ArmingState != "safe" {
		t.Fatalf("unexpected arming state: %s", cfg.Live.Runtime.ArmingState)
	}
	if cfg.WatchlistPath != "platform/watchlist.yaml" {
		t.Fatalf("unexpected watchlist path: %s", cfg.WatchlistPath)
	}
}

func TestLoadLiveConfig(t *testing.T) {
	cfg, err := Load("../../configs/live.yaml")
	if err != nil {
		t.Fatalf("load live config: %v", err)
	}
	if cfg.WatchlistPath != "platform/watchlist.yaml" {
		t.Fatalf("unexpected watchlist path: %s", cfg.WatchlistPath)
	}
	if cfg.WarehouseConfigPath != "platform/warehouse.yaml" {
		t.Fatalf("unexpected warehouse config path: %s", cfg.WarehouseConfigPath)
	}
	if cfg.Live.Exchange.ProductType != "USDT-FUTURES" {
		t.Fatalf("unexpected product type: %s", cfg.Live.Exchange.ProductType)
	}
	if cfg.Live.Exchange.MarginMode != "isolated" {
		t.Fatalf("unexpected margin mode: %s", cfg.Live.Exchange.MarginMode)
	}
	if cfg.Live.Exchange.APIKeyEnv != "BITGET_API_KEY" || cfg.Live.Exchange.APISecretEnv != "BITGET_API_SECRET" || cfg.Live.Exchange.PassphraseEnv != "BITGET_PASSPHRASE" {
		t.Fatalf("unexpected credential envs: %+v", cfg.Live.Exchange)
	}
	if !cfg.Live.Runtime.ObserveOnly {
		t.Fatalf("live config must default to observe_only")
	}
	if cfg.Live.Risk.MaxLeverage != 3 {
		t.Fatalf("unexpected leverage cap: %d", cfg.Live.Risk.MaxLeverage)
	}
	if !cfg.Live.Agent.AdvisoryOnly {
		t.Fatalf("agent must remain advisory-only")
	}
	if !cfg.Insights.Enabled {
		t.Fatalf("insights must be enabled in live config")
	}
	if cfg.Insights.DailySignal.Interval != "1d" {
		t.Fatalf("unexpected daily signal interval: %s", cfg.Insights.DailySignal.Interval)
	}
	if cfg.Insights.Anomaly.Interval != "15m" {
		t.Fatalf("unexpected anomaly interval: %s", cfg.Insights.Anomaly.Interval)
	}
	if cfg.Insights.Dashboard.MarketdRestartUnit != "quantlab-marketd.service" {
		t.Fatalf("unexpected marketd restart unit: %s", cfg.Insights.Dashboard.MarketdRestartUnit)
	}
}

func TestLoadConfigIncludesWarehouseConfigPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "live.yaml")
	if err := os.WriteFile(path, []byte("warehouse_config_path: configs/platform/warehouse.yaml\nlive:\n  enabled: true\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.WarehouseConfigPath != "configs/platform/warehouse.yaml" {
		t.Fatalf("unexpected warehouse config path: %s", cfg.WarehouseConfigPath)
	}
}

func TestLoadLiveConfigInsightsDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "live.yaml")
	if err := os.WriteFile(path, []byte("live:\n  enabled: true\ninsights:\n  enabled: true\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Insights.ScanInterval <= 0 {
		t.Fatalf("expected scan interval default, got %s", cfg.Insights.ScanInterval)
	}
	if cfg.Insights.PriceTTL <= 0 {
		t.Fatalf("expected price ttl default, got %s", cfg.Insights.PriceTTL)
	}
	if cfg.Insights.DailySignal.LookbackBars != 1095 {
		t.Fatalf("unexpected daily lookback: %d", cfg.Insights.DailySignal.LookbackBars)
	}
	if cfg.Insights.Anomaly.LookbackBars != 105120 {
		t.Fatalf("unexpected anomaly lookback: %d", cfg.Insights.Anomaly.LookbackBars)
	}
	if cfg.Insights.Dashboard.PlatformctlPath != "./bin/platformctl" {
		t.Fatalf("unexpected platformctl path: %s", cfg.Insights.Dashboard.PlatformctlPath)
	}
}
