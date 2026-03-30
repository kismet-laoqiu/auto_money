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
