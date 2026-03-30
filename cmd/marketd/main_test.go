package main

import (
	"context"
	"testing"

	"quantlab/internal/config"
	"quantlab/internal/exchange/bitget"
	"quantlab/internal/market"
	sqlitepkg "quantlab/internal/store/sqlite"
)

type runtimeRecorder struct {
	cfgs []market.RuntimeConfig
}

type runtimeStub struct{}

func (runtimeStub) Run(context.Context) error { return nil }

func TestRunStartsOnePublicRuntimePerSymbol(t *testing.T) {
	originalStore := newSQLiteStore
	originalPublicClient := newPublicBitgetClient
	originalPrivateClient := newPrivateBitgetClient
	originalPublicSource := newPublicWSSource
	originalPrivateSource := newPrivateWSSource
	originalRuntime := newMarketRuntime
	defer func() {
		newSQLiteStore = originalStore
		newPublicBitgetClient = originalPublicClient
		newPrivateBitgetClient = originalPrivateClient
		newPublicWSSource = originalPublicSource
		newPrivateWSSource = originalPrivateSource
		newMarketRuntime = originalRuntime
	}()

	newSQLiteStore = func(path string) (*sqlitepkg.Store, error) { return &sqlitepkg.Store{}, nil }
	newPublicBitgetClient = func(string) market.BootstrapLoader { return nil }
	newPrivateBitgetClient = func(string, bitget.PrivateCredentials) *bitget.Client { return nil }
	newPublicWSSource = func(string, string, string, string) market.EventSource { return nil }
	newPrivateWSSource = func(string, bitget.PrivateCredentials, string) market.EventSource { return nil }

	var recorder runtimeRecorder
	newMarketRuntime = func(cfg market.RuntimeConfig) marketRuntimeRunner {
		recorder.cfgs = append(recorder.cfgs, cfg)
		return runtimeStub{}
	}

	cfg := config.Config{
		Stream: config.StreamConfig{Interval: "1m", SnapshotLimit: 20},
		Live: config.LiveConfig{
			Exchange: config.ExchangeConfig{
				ProductType: "USDT-FUTURES",
				Symbols: []config.LiveSymbolConfig{
					{Symbol: "BTCUSDT"},
					{Symbol: "ETHUSDT"},
				},
			},
			Runtime: config.RuntimeConfig{StateDBPath: t.TempDir() + "/state.db"},
		},
	}

	if err := run(context.Background(), cfg); err != nil {
		t.Fatalf("run marketd: %v", err)
	}
	if len(recorder.cfgs) != 2 {
		t.Fatalf("unexpected runtime count: %d", len(recorder.cfgs))
	}
	if recorder.cfgs[0].Symbol != "BTCUSDT" || recorder.cfgs[1].Symbol != "ETHUSDT" {
		t.Fatalf("unexpected runtime symbols: %+v", recorder.cfgs)
	}
}
