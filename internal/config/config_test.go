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

func TestLoadMSTRE2EConfig(t *testing.T) {
	cfg, err := Load("../../configs/demo-mstr-e2e.yaml")
	if err != nil {
		t.Fatalf("load mstr e2e config: %v", err)
	}
	if len(cfg.Live.Exchange.Symbols) != 1 || cfg.Live.Exchange.Symbols[0].Symbol != "MSTRUSDT" {
		t.Fatalf("unexpected live symbols: %+v", cfg.Live.Exchange.Symbols)
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
		t.Fatalf("mstr e2e config must default to observe_only")
	}
	if cfg.Live.Risk.MaxLeverage != 3 {
		t.Fatalf("unexpected leverage cap: %d", cfg.Live.Risk.MaxLeverage)
	}
	if !cfg.Live.Agent.AdvisoryOnly {
		t.Fatalf("agent must remain advisory-only")
	}
}
