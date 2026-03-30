package strategybundle

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestLoadFixtureBundle(t *testing.T) {
	bundle, err := Load(fixtureBundlePath(t))
	if err != nil {
		t.Fatalf("load bundle: %v", err)
	}
	if bundle.StrategyID != "mstr-wave-fib" {
		t.Fatalf("unexpected strategy id: %s", bundle.StrategyID)
	}
	if bundle.Version != "v0.1.0" {
		t.Fatalf("unexpected version: %s", bundle.Version)
	}
	if bundle.Strategy.FastSMA != 15 || bundle.Strategy.SlowSMA != 60 {
		t.Fatalf("unexpected strategy config: %+v", bundle.Strategy)
	}
	if len(bundle.Universe.Datasets) != 1 || bundle.Universe.Datasets[0].Symbol != "MSTRUSDT" {
		t.Fatalf("unexpected universe: %+v", bundle.Universe)
	}
	if !strings.Contains(bundle.ScoreCEL, "final_score") {
		t.Fatalf("unexpected score program: %q", bundle.ScoreCEL)
	}
	if !strings.Contains(bundle.GatesCEL, "shadow") {
		t.Fatalf("unexpected gates program: %q", bundle.GatesCEL)
	}
}

func TestValidateRejectsMissingRequiredFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "strategy.yaml"), "strategy_id: demo\nversion: v0.0.1\nstrategy:\n  fast_sma: 15\n  slow_sma: 60\n")
	writeFile(t, filepath.Join(dir, "universe.yaml"), "datasets: []\n")
	writeFile(t, filepath.Join(dir, "risk.yaml"), "max_leverage: 3\n")
	writeFile(t, filepath.Join(dir, "score.cel"), "final_score")
	if err := Validate(dir); err == nil || !strings.Contains(err.Error(), "gates.cel") {
		t.Fatalf("expected missing gates.cel error, got %v", err)
	}
}

func TestFixtureBundlePassesValidation(t *testing.T) {
	if err := Validate(fixtureBundlePath(t)); err != nil {
		t.Fatalf("validate fixture bundle: %v", err)
	}
}

func fixtureBundlePath(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime caller unavailable")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "strategies", "mstr-wave-fib", "versions", "v0.1.0"))
}

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
