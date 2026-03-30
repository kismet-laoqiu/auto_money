package strategybundle

import (
	"os"
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

func TestLoadConfigResolvesWarehouseConfigPathRelativeToConfigFile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte("warehouse_config_path: warehouse.yaml\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	resolved, bundle, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if bundle != nil {
		t.Fatalf("unexpected bundle: %+v", bundle)
	}
	if resolved.WarehouseConfigPath != filepath.Join(dir, "warehouse.yaml") {
		t.Fatalf("unexpected warehouse config path: %s", resolved.WarehouseConfigPath)
	}
}

func TestLoadConfigOverlaysWatchlistSymbolsIntoLiveAndStream(t *testing.T) {
	dir := t.TempDir()
	watchlistPath := filepath.Join(dir, "watchlist.yaml")
	if err := os.WriteFile(watchlistPath, []byte("symbols:\n  - symbol: BTCUSDT\n    max_notional: 500\n    max_tranches: 4\n  - symbol: ETHUSDT\n    max_notional: 500\n    max_tranches: 4\n"), 0o644); err != nil {
		t.Fatalf("write watchlist: %v", err)
	}
	cfgPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte("watchlist_path: watchlist.yaml\nlive:\n  enabled: true\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	resolved, bundle, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if bundle != nil {
		t.Fatalf("unexpected bundle: %+v", bundle)
	}
	if len(resolved.Live.Exchange.Symbols) != 2 || resolved.Live.Exchange.Symbols[0].Symbol != "BTCUSDT" || resolved.Live.Exchange.Symbols[1].Symbol != "ETHUSDT" {
		t.Fatalf("unexpected live symbols: %+v", resolved.Live.Exchange.Symbols)
	}
	if len(resolved.Stream.Symbols) != 2 || resolved.Stream.Symbols[0] != "BTCUSDT" || resolved.Stream.Symbols[1] != "ETHUSDT" {
		t.Fatalf("unexpected stream symbols: %+v", resolved.Stream.Symbols)
	}
	if resolved.Stream.Interval != "1m" {
		t.Fatalf("unexpected stream interval: %s", resolved.Stream.Interval)
	}
}
