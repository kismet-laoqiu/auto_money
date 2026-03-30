package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"quantlab/internal/config"
	sqlitepkg "quantlab/internal/store/sqlite"
	"quantlab/internal/strategybundle"
	"quantlab/internal/trader"
)

func main() {
	fs := flag.NewFlagSet("traderd", flag.ContinueOnError)
	configPath := fs.String("config", "configs/live.yaml", "config file")
	if err := fs.Parse(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	cfg, bundle, err := strategybundle.LoadConfig(*configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if !cfg.Live.Enabled {
		fmt.Fprintln(os.Stderr, "live.enabled=false")
		os.Exit(1)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, cfg, bundle); err != nil && err != context.Canceled {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, cfg config.Config, bundle *strategybundle.Bundle) error {
	store, err := sqlitepkg.NewStore(cfg.Live.Runtime.StateDBPath)
	if err != nil {
		return err
	}
	strategy := trader.Strategy(trader.LegacyRuleProfile{StrategyCfg: cfg.Strategy})
	if bundle != nil {
		strategy = trader.BundleStrategy{Bundle: *bundle}
	}
	engine := trader.NewEngine(trader.Config{
		ArmingState: trader.ArmingState(cfg.Live.Runtime.ArmingState),
		Strategy:    strategy,
	})
	runtime := trader.NewRuntime(trader.RuntimeConfig{
		ConsumerKey:   "traderd",
		CheckpointKey: "trader.runtime",
		ObserveOnly:   cfg.Live.Runtime.ObserveOnly,
		RunID:         "traderd",
		MaxLeverage:   cfg.Live.Risk.MaxLeverage,
		Store:         runtimeStoreAdapter{store: store},
		Engine:        engine,
		Risk:          trader.NewRiskEngine(trader.RiskConfig{MaxLeverage: cfg.Live.Risk.MaxLeverage}),
		Reconciler:    trader.NewReconciler(),
	})
	return runtime.Run(ctx)
}

type runtimeStoreAdapter struct {
	store *sqlitepkg.Store
}

func (adapter runtimeStoreAdapter) AppendEvent(ctx context.Context, source string, evt trader.EventLog, raw []byte) (int64, error) {
	return adapter.store.AppendEvent(ctx, source, evt, raw)
}

func (adapter runtimeStoreAdapter) ListEventsAfter(ctx context.Context, afterSeq int64, limit int, sources ...string) ([]trader.EventEnvelope, error) {
	envelopes, err := adapter.store.ListEventsAfter(ctx, afterSeq, limit, sources...)
	if err != nil {
		return nil, err
	}
	out := make([]trader.EventEnvelope, 0, len(envelopes))
	for _, env := range envelopes {
		out = append(out, trader.EventEnvelope{
			Seq:        env.Seq,
			Source:     env.Source,
			EventID:    env.EventID,
			Symbol:     env.Symbol,
			Kind:       env.Kind,
			ExchangeTS: env.ExchangeTS,
			ReceivedTS: env.ReceivedTS,
			Payload:    append([]byte(nil), env.Payload...),
		})
	}
	return out, nil
}

func (adapter runtimeStoreAdapter) SaveConsumerCursor(ctx context.Context, consumer string, seq int64) error {
	return adapter.store.SaveConsumerCursor(ctx, consumer, seq)
}

func (adapter runtimeStoreAdapter) LoadConsumerCursor(ctx context.Context, consumer string) (int64, error) {
	return adapter.store.LoadConsumerCursor(ctx, consumer)
}

func (adapter runtimeStoreAdapter) SaveCheckpoint(ctx context.Context, shard string, state trader.EngineState) error {
	return adapter.store.SaveCheckpoint(ctx, shard, state)
}

func (adapter runtimeStoreAdapter) LoadCheckpoint(ctx context.Context, shard string) (trader.EngineState, error) {
	return adapter.store.LoadCheckpoint(ctx, shard)
}
