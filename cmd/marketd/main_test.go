package main

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"quantlab/internal/config"
	"quantlab/internal/core"
	"quantlab/internal/exchange/bitget"
	"quantlab/internal/market"
	sqlitepkg "quantlab/internal/store/sqlite"
	"quantlab/internal/watchlist"
)

type runtimeRecorder struct {
	cfgs []market.RuntimeConfig
}

type runtimeStub struct{}

func (runtimeStub) Run(context.Context) error { return nil }

type appendRecord struct {
	source string
	event  sqlitepkg.LogEvent
}

type liveStoreStub struct {
	records []appendRecord
}

func (store *liveStoreStub) AppendEvent(_ context.Context, source string, evt sqlitepkg.LogEvent, _ []byte) (int64, error) {
	store.records = append(store.records, appendRecord{source: source, event: evt})
	return int64(len(store.records)), nil
}

type warehouseStoreStub struct {
	specs []config.DatasetConfig
	bars  [][]core.Bar
	err   error
}

func (store *warehouseStoreStub) UpsertBars(_ context.Context, spec config.DatasetConfig, bars []core.Bar) (int, error) {
	if store.err != nil {
		return 0, store.err
	}
	store.specs = append(store.specs, spec)
	store.bars = append(store.bars, append([]core.Bar(nil), bars...))
	return len(bars), nil
}

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

	newSQLiteStore = func(path string) (liveEventStore, error) { return &liveStoreStub{}, nil }
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

func TestRunStartsWarehouseRuntimesFromWatchlistIntervals(t *testing.T) {
	originalStore := newSQLiteStore
	originalPublicClient := newPublicBitgetClient
	originalPrivateClient := newPrivateBitgetClient
	originalPublicSource := newPublicWSSource
	originalPrivateSource := newPrivateWSSource
	originalRuntime := newMarketRuntime
	originalLoadWatchlist := loadWatchlistFile
	originalLoadWarehouseConfig := loadWarehouseConfig
	originalOpenWarehouseDB := openWarehouseDB
	originalNewWarehouseStore := newWarehouseStore
	defer func() {
		newSQLiteStore = originalStore
		newPublicBitgetClient = originalPublicClient
		newPrivateBitgetClient = originalPrivateClient
		newPublicWSSource = originalPublicSource
		newPrivateWSSource = originalPrivateSource
		newMarketRuntime = originalRuntime
		loadWatchlistFile = originalLoadWatchlist
		loadWarehouseConfig = originalLoadWarehouseConfig
		openWarehouseDB = originalOpenWarehouseDB
		newWarehouseStore = originalNewWarehouseStore
	}()

	liveStore := &liveStoreStub{}
	warehouseStore := &warehouseStoreStub{}
	newSQLiteStore = func(path string) (liveEventStore, error) { return liveStore, nil }
	newPublicBitgetClient = func(string) market.BootstrapLoader { return nil }
	newPrivateBitgetClient = func(string, bitget.PrivateCredentials) *bitget.Client { return nil }
	newPublicWSSource = func(string, string, string, string) market.EventSource { return nil }
	newPrivateWSSource = func(string, bitget.PrivateCredentials, string) market.EventSource { return nil }
	loadWatchlistFile = func(path string) (watchlist.File, error) {
		return watchlist.File{
			HistoricalIntervals: []string{"15m", "1h"},
			Symbols:             []config.LiveSymbolConfig{{Symbol: "BTCUSDT"}},
		}, nil
	}
	loadWarehouseConfig = func(path string) (warehouseConfig, error) {
		return warehouseConfig{DSN: "postgres://warehouse"}, nil
	}
	openWarehouseDB = func(cfg warehouseConfig) (warehouseDB, error) {
		return nil, nil
	}
	newWarehouseStore = func(db warehouseDB) warehouseBarStore { return warehouseStore }

	var recorder runtimeRecorder
	newMarketRuntime = func(cfg market.RuntimeConfig) marketRuntimeRunner {
		recorder.cfgs = append(recorder.cfgs, cfg)
		return runtimeStub{}
	}

	cfg := config.Config{
		WatchlistPath:       "configs/platform/watchlist.yaml",
		WarehouseConfigPath: "configs/platform/warehouse.yaml",
		Stream:              config.StreamConfig{Interval: "1m", SnapshotLimit: 20},
		Live: config.LiveConfig{
			Exchange: config.ExchangeConfig{
				ProductType: "USDT-FUTURES",
				Symbols:     []config.LiveSymbolConfig{{Symbol: "BTCUSDT"}},
			},
			Runtime: config.RuntimeConfig{StateDBPath: t.TempDir() + "/state.db"},
		},
	}

	if err := run(context.Background(), cfg); err != nil {
		t.Fatalf("run marketd: %v", err)
	}
	if len(recorder.cfgs) != 3 {
		t.Fatalf("unexpected runtime count: %d", len(recorder.cfgs))
	}

	byInterval := map[string]market.RuntimeConfig{}
	for _, runtimeCfg := range recorder.cfgs {
		byInterval[runtimeCfg.Interval] = runtimeCfg
	}
	if _, ok := byInterval["1m"]; !ok {
		t.Fatalf("missing stream runtime: %+v", recorder.cfgs)
	}
	if _, ok := byInterval["15m"]; !ok {
		t.Fatalf("missing 15m warehouse runtime: %+v", recorder.cfgs)
	}
	if _, ok := byInterval["1h"]; !ok {
		t.Fatalf("missing 1h warehouse runtime: %+v", recorder.cfgs)
	}

	streamBar := market.BarClosedEvent{
		EventIDValue: "stream-bar",
		SymbolValue:  "BTCUSDT",
		Interval:     "1m",
		Ts:           time.Unix(1710000000, 0).UTC(),
		Open:         100,
		High:         101,
		Low:          99,
		Close:        100.5,
		Volume:       12,
	}
	if _, err := byInterval["1m"].AppendEvent(context.Background(), "market.public", streamBar, nil); err != nil {
		t.Fatalf("append stream event: %v", err)
	}
	if len(liveStore.records) != 1 {
		t.Fatalf("unexpected live store writes: %+v", liveStore.records)
	}
	if len(warehouseStore.specs) != 0 {
		t.Fatalf("unexpected warehouse writes for stream interval: %+v", warehouseStore.specs)
	}

	warehouseBar := market.BarClosedEvent{
		EventIDValue: "warehouse-bar",
		SymbolValue:  "BTCUSDT",
		Interval:     "15m",
		Ts:           time.Unix(1710000900, 0).UTC(),
		Open:         110,
		High:         112,
		Low:          109,
		Close:        111,
		Volume:       18,
	}
	if _, err := byInterval["15m"].AppendEvent(context.Background(), "market.public", warehouseBar, nil); err != nil {
		t.Fatalf("append warehouse event: %v", err)
	}
	if len(warehouseStore.specs) != 1 {
		t.Fatalf("unexpected warehouse writes: %+v", warehouseStore.specs)
	}
	if warehouseStore.specs[0].Symbol != "BTCUSDT" || warehouseStore.specs[0].Interval != "15m" {
		t.Fatalf("unexpected warehouse dataset: %+v", warehouseStore.specs[0])
	}
	if len(warehouseStore.bars[0]) != 1 || warehouseStore.bars[0][0].Close != 111 {
		t.Fatalf("unexpected warehouse bars: %+v", warehouseStore.bars)
	}
	if len(liveStore.records) != 1 {
		t.Fatalf("warehouse runtime must not write sqlite events: %+v", liveStore.records)
	}
}

func TestAppendWarehouseBarTreatsCanceledTxDoneAsContextCanceled(t *testing.T) {
	store := &warehouseStoreStub{err: sql.ErrTxDone}
	appendFn := appendWarehouseBar(store, "bitget", "USDT-FUTURES", "BTCUSDT", "15m")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := appendFn(ctx, "market.public", market.BarClosedEvent{
		EventIDValue: "warehouse-bar",
		SymbolValue:  "BTCUSDT",
		Interval:     "15m",
		Ts:           time.Unix(1710000900, 0).UTC(),
		Open:         110,
		High:         112,
		Low:          109,
		Close:        111,
		Volume:       18,
	}, nil)
	if err != context.Canceled {
		t.Fatalf("expected context canceled, got %v", err)
	}
}
