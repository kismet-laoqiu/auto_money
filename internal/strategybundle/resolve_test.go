package strategybundle

import (
	"path/filepath"
	"runtime"
	"testing"

	"quantlab/internal/config"
)

func TestLoadConfigResolvesBundlePathRelativeToConfigFile(t *testing.T) {
	dir := t.TempDir()
	rel, err := filepath.Rel(dir, fixtureBundlePath(t))
	if err != nil {
		t.Fatalf("relative bundle path: %v", err)
	}
	cfgPath := filepath.Join(dir, "bundle-config.yaml")
	writeFile(t, cfgPath, "strategy_bundle_path: "+rel+"\n")

	resolved, bundle, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if bundle == nil || bundle.RootPath != filepath.Clean(fixtureBundlePath(t)) {
		t.Fatalf("unexpected bundle: %+v", bundle)
	}
	if resolved.Strategy.FastSMA != 15 || resolved.Strategy.SlowSMA != 60 {
		t.Fatalf("unexpected strategy overlay: %+v", resolved.Strategy)
	}
	if len(resolved.Datasets) != 1 || resolved.Datasets[0].Symbol != "MSTRUSDT" {
		t.Fatalf("unexpected dataset overlay: %+v", resolved.Datasets)
	}
	if resolved.Live.Risk.MaxLeverage != 3 {
		t.Fatalf("unexpected leverage overlay: %d", resolved.Live.Risk.MaxLeverage)
	}
	if resolved.Live.Exchange.ProductType != "USDT-FUTURES" || resolved.Live.Exchange.MarginMode != "isolated" {
		t.Fatalf("unexpected exchange overlay: %+v", resolved.Live.Exchange)
	}
}

func TestResolveConfigOverlaysBundleFields(t *testing.T) {
	cfg := config.Config{
		StrategyBundlePath: fixtureBundlePath(t),
		Datasets:           []config.DatasetConfig{{Name: "legacy", Symbol: "BTCUSDT"}},
		Live: config.LiveConfig{
			Risk: config.RiskConfig{MaxLeverage: 1},
			Exchange: config.ExchangeConfig{
				ProductType: "SPOT",
				MarginMode:  "crossed",
				Symbols:     []config.LiveSymbolConfig{{Symbol: "BTCUSDT", MaxNotional: 1, MaxTranches: 1}},
			},
		},
	}
	resolved, bundle, err := ResolveConfig(cfg)
	if err != nil {
		t.Fatalf("resolve config: %v", err)
	}
	if bundle == nil || bundle.StrategyID != "mstr-wave-fib" {
		t.Fatalf("unexpected bundle: %+v", bundle)
	}
	if resolved.Strategy.FastSMA != 15 || resolved.Strategy.SlowSMA != 60 {
		t.Fatalf("unexpected strategy overlay: %+v", resolved.Strategy)
	}
	if len(resolved.Datasets) != 1 || resolved.Datasets[0].Symbol != "MSTRUSDT" {
		t.Fatalf("unexpected dataset overlay: %+v", resolved.Datasets)
	}
	if resolved.Live.Risk.MaxLeverage != 3 {
		t.Fatalf("unexpected leverage overlay: %d", resolved.Live.Risk.MaxLeverage)
	}
	if resolved.Live.Exchange.ProductType != "USDT-FUTURES" || resolved.Live.Exchange.MarginMode != "isolated" {
		t.Fatalf("unexpected exchange overlay: %+v", resolved.Live.Exchange)
	}
	if len(resolved.Live.Exchange.Symbols) != 1 || resolved.Live.Exchange.Symbols[0].Symbol != "MSTRUSDT" {
		t.Fatalf("unexpected live symbols: %+v", resolved.Live.Exchange.Symbols)
	}
}

func TestLoadConfigResolvesDemoBundleConfig(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime caller unavailable")
	}
	cfgPath := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "configs", "demo-mstr-bundle.yaml"))
	resolved, bundle, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("load demo bundle config: %v", err)
	}
	if bundle == nil || bundle.StrategyID != "mstr-wave-fib" || bundle.Version != "v0.1.0" {
		t.Fatalf("unexpected bundle: %+v", bundle)
	}
	if len(resolved.Datasets) != 1 || resolved.Datasets[0].Symbol != "MSTRUSDT" {
		t.Fatalf("unexpected datasets: %+v", resolved.Datasets)
	}
	if resolved.Live.Exchange.ProductType != "USDT-FUTURES" || resolved.Live.Exchange.MarginMode != "isolated" {
		t.Fatalf("unexpected exchange overlay: %+v", resolved.Live.Exchange)
	}
	if resolved.Live.Risk.MaxLeverage != 3 {
		t.Fatalf("unexpected leverage: %d", resolved.Live.Risk.MaxLeverage)
	}
}
