package strategybundle

import (
	"path/filepath"
	"testing"
)

func TestRegistryListReturnsNewestVersionFirst(t *testing.T) {
	root := t.TempDir()
	writeBundleVersion(t, root, "mstr-wave-fib", "v0.1.0", "first")
	writeBundleVersion(t, root, "mstr-wave-fib", "v0.2.0", "second")

	registry := NewRegistry(root)
	versions, err := registry.List("mstr-wave-fib")
	if err != nil {
		t.Fatalf("list versions: %v", err)
	}
	if len(versions) != 2 {
		t.Fatalf("unexpected versions: %+v", versions)
	}
	if versions[0].Version != "v0.2.0" || versions[1].Version != "v0.1.0" {
		t.Fatalf("unexpected ordering: %+v", versions)
	}
	if versions[0].RootPath != filepath.Join(root, "mstr-wave-fib", "versions", "v0.2.0") {
		t.Fatalf("unexpected root path: %+v", versions[0])
	}
}

func TestRegistryGetLoadsBundle(t *testing.T) {
	root := t.TempDir()
	writeBundleVersion(t, root, "mstr-wave-fib", "v0.1.0", "bundle from registry")

	bundle, err := NewRegistry(root).Get("mstr-wave-fib", "v0.1.0")
	if err != nil {
		t.Fatalf("get bundle: %v", err)
	}
	if bundle.StrategyID != "mstr-wave-fib" || bundle.Version != "v0.1.0" || bundle.Description != "bundle from registry" {
		t.Fatalf("unexpected bundle: %+v", bundle)
	}
}

func TestCompareVersionsOrdersNumerically(t *testing.T) {
	if CompareVersions("v0.2.0", "v0.10.0") >= 0 {
		t.Fatalf("expected v0.10.0 to sort after v0.2.0")
	}
	if CompareVersions("v1.0.0", "v0.10.0") <= 0 {
		t.Fatalf("expected v1.0.0 to sort after v0.10.0")
	}
	if CompareVersions("v0.1.0", "v0.1.0") != 0 {
		t.Fatalf("expected equal versions to compare as zero")
	}
}

func writeBundleVersion(t *testing.T, root, strategyID, version, description string) {
	t.Helper()
	bundleRoot := filepath.Join(root, strategyID, "versions", version)
	writeFile(t, filepath.Join(bundleRoot, "strategy.yaml"), "strategy_id: "+strategyID+"\nversion: "+version+"\ndescription: "+description+"\nobjective:\n  risk_free_rate: 0\n  overfit_penalty_weight: 0.35\n  drawdown_weight: 2\n  calmar_weight: 0.35\n  pnl_weight: 0.2\nstrategy:\n  fast_sma: 15\n  slow_sma: 60\n  pivot_window: 3\n  level_lookback: 90\n  level_tolerance: 0.012\n  fib_tolerance: 0.012\n  signal_threshold: 3.25\n  stop_atr: 1\n  reward_risk: 3\n  max_hold_bars: 20\n  cooldown_bars: 3\n  commission_bps: 2\n  slippage_bps: 3\n  atr_window: 14\n")
	writeFile(t, filepath.Join(bundleRoot, "universe.yaml"), "datasets:\n  - name: mstrusdt_demo_replay\n    provider: bitget\n    symbol: MSTRUSDT\n    interval: 1m\n    limit: 720\n    product_type: USDT-FUTURES\nsymbols:\n  - MSTRUSDT\n")
	writeFile(t, filepath.Join(bundleRoot, "risk.yaml"), "max_leverage: 3\nmargin_mode: isolated\nproduct_type: USDT-FUTURES\nsymbols:\n  - symbol: MSTRUSDT\n    max_notional: 30\n    max_tranches: 1\n")
	writeFile(t, filepath.Join(bundleRoot, "score.cel"), "final_score")
	writeFile(t, filepath.Join(bundleRoot, "gates.cel"), "shadow")
}
