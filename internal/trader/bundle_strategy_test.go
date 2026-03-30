package trader

import (
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
	"time"

	"quantlab/internal/core"
	"quantlab/internal/strategybundle"
)

func TestBundleStrategyMirrorsLegacyRuleProfile(t *testing.T) {
	bundle, err := strategybundle.Load(bundleFixturePath(t))
	if err != nil {
		t.Fatalf("load bundle: %v", err)
	}
	bars := sampleBundleBars(220)
	legacy := LegacyRuleProfile{StrategyCfg: bundle.Strategy}
	strategy := BundleStrategy{Bundle: bundle}
	legacySignal := legacy.OnBar("MSTRUSDT", "1m", bars)
	bundleSignal := strategy.OnBar("MSTRUSDT", "1m", bars)
	if !reflect.DeepEqual(bundleSignal, legacySignal) {
		t.Fatalf("bundle strategy drifted from legacy\nlegacy=%+v\nbundle=%+v", legacySignal, bundleSignal)
	}
}

func TestBundleStrategyNameIncludesBundleIdentity(t *testing.T) {
	bundle := strategybundle.Bundle{StrategyID: "mstr-wave-fib", Version: "v0.1.0"}
	if got := (BundleStrategy{Bundle: bundle}).Name(); got != "mstr-wave-fib@v0.1.0" {
		t.Fatalf("unexpected name: %s", got)
	}
}

func bundleFixturePath(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime caller unavailable")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "strategies", "mstr-wave-fib", "versions", "v0.1.0"))
}

func sampleBundleBars(count int) []core.Bar {
	bars := make([]core.Bar, 0, count)
	start := time.Unix(1710000000, 0).UTC()
	price := 100.0
	for i := 0; i < count; i++ {
		drift := 0.35
		if i%17 == 0 {
			drift = -1.4
		}
		if i%29 == 0 {
			drift = 1.8
		}
		closePrice := price + drift
		high := closePrice + 1.2
		low := closePrice - 1.2
		if i%17 == 0 {
			low = closePrice - 3.4
		}
		bars = append(bars, core.Bar{
			Time:   start.Add(time.Duration(i) * time.Minute),
			Open:   price,
			High:   high,
			Low:    low,
			Close:  closePrice,
			Volume: 100 + float64(i%13)*6,
		})
		price = closePrice
	}
	return bars
}
