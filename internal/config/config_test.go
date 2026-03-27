package config

import "testing"

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
}
