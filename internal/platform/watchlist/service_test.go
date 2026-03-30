package watchlistsvc

import (
	"context"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"os"
)

func TestServiceSaveSymbolsPreservesExistingSettingsAndDefaultsNewSymbols(t *testing.T) {
	path := filepath.Join(t.TempDir(), "watchlist.yaml")
	if err := os.WriteFile(path, []byte(`
provider: bitget
product_type: USDT-FUTURES
stream_interval: 1m
historical_intervals: [15m, 1h, 4h, 1d, 1w]
horizon_days: 1095
symbols:
  - symbol: BTCUSDT
    max_notional: 700
    max_tranches: 5
  - symbol: ETHUSDT
    max_notional: 900
    max_tranches: 6
`), 0o644); err != nil {
		t.Fatalf("write watchlist: %v", err)
	}
	service := NewService(Config{WatchlistPath: path})

	file, err := service.SaveSymbols("ETHUSDT\nDOGEUSDT")
	if err != nil {
		t.Fatalf("save symbols: %v", err)
	}
	if got := file.SymbolNames(); !reflect.DeepEqual(got, []string{"ETHUSDT", "DOGEUSDT"}) {
		t.Fatalf("unexpected symbols: %+v", got)
	}
	if file.Symbols[0].MaxNotional != 900 || file.Symbols[0].MaxTranches != 6 {
		t.Fatalf("expected existing symbol config preserved: %+v", file.Symbols[0])
	}
	if file.Symbols[1].MaxNotional != 700 || file.Symbols[1].MaxTranches != 5 {
		t.Fatalf("expected new symbol defaulted from baseline: %+v", file.Symbols[1])
	}
}

func TestServiceApplyRunsPlatformctlAndRestartsMarketd(t *testing.T) {
	path := filepath.Join(t.TempDir(), "watchlist.yaml")
	if err := os.WriteFile(path, []byte(`
provider: bitget
product_type: USDT-FUTURES
symbols:
  - symbol: BTCUSDT
    max_notional: 500
    max_tranches: 4
`), 0o644); err != nil {
		t.Fatalf("write watchlist: %v", err)
	}
	runner := &runnerStub{output: []byte(`{"watchlist_path":"configs/platform/watchlist.yaml"}`)}
	service := NewService(Config{
		WatchlistPath:       path,
		LiveConfigPath:      "configs/live.yaml",
		WarehouseConfigPath: "configs/platform/warehouse.yaml",
		PlatformctlPath:     "./bin/platformctl",
		MarketdRestartUnit:  "quantlab-marketd.service",
		Runner:              runner,
		Now:                 func() time.Time { return time.Unix(1710000000, 0).UTC() },
	})

	result, err := service.Apply(context.Background())
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if len(runner.calls) != 2 {
		t.Fatalf("unexpected runner calls: %+v", runner.calls)
	}
	if want := []string{"./bin/platformctl", "watchlist", "apply", "-watchlist", path, "-live-config", "configs/live.yaml", "-warehouse-config", "configs/platform/warehouse.yaml"}; !reflect.DeepEqual(runner.calls[0], want) {
		t.Fatalf("unexpected platformctl call: %+v", runner.calls[0])
	}
	if want := []string{"systemctl", "restart", "quantlab-marketd.service"}; !reflect.DeepEqual(runner.calls[1], want) {
		t.Fatalf("unexpected restart call: %+v", runner.calls[1])
	}
	if !strings.Contains(string(result.PlatformctlJSON), `"watchlist_path"`) {
		t.Fatalf("unexpected apply result: %+v", result)
	}
}

type runnerStub struct {
	calls  [][]string
	output []byte
	err    error
}

func (runner *runnerStub) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	call := append([]string{name}, args...)
	runner.calls = append(runner.calls, call)
	return append([]byte(nil), runner.output...), runner.err
}
