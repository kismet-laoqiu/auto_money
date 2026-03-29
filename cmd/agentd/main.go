package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"quantlab/internal/agent"
	"quantlab/internal/config"
	sqlitepkg "quantlab/internal/store/sqlite"
	"quantlab/internal/trader"
)

func main() {
	fs := flag.NewFlagSet("agentd", flag.ContinueOnError)
	configPath := fs.String("config", "configs/demo-bitget.yaml", "config file")
	if err := fs.Parse(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	cfg, err := config.Load(*configPath)
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
	if err := run(ctx, cfg); err != nil && err != context.Canceled {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newService(cfg config.Config) (*agent.Service, error) {
	if !cfg.Live.Agent.AdvisoryOnly {
		return nil, fmt.Errorf("agentd requires live.agent.advisory_only=true")
	}
	apiKey := os.Getenv("OPENAI_API_KEY")
	baseURL := os.Getenv("OPENAI_BASE_URL")
	client := agent.NewHTTPResponsesClient(agent.HTTPClientConfig{APIKey: apiKey, BaseURL: baseURL})
	return agent.NewService(client, agent.Config{Store: true}), nil
}

func run(ctx context.Context, cfg config.Config) error {
	if !cfg.Live.Agent.Enabled {
		<-ctx.Done()
		return ctx.Err()
	}
	service, err := newService(cfg)
	if err != nil {
		return err
	}
	store, err := sqlitepkg.NewStore(cfg.Live.Runtime.StateDBPath)
	if err != nil {
		return err
	}
	runtime := agent.NewRuntime(agent.RuntimeConfig{
		Store:       runtimeStoreAdapter{store: store},
		Service:     service,
		MaxLeverage: cfg.Live.Risk.MaxLeverage,
		ArtifactDir: filepath.Join(cfg.ArtifactDir, "agentd"),
	})
	return runtime.Run(ctx)
}

type runtimeStoreAdapter struct {
	store *sqlitepkg.Store
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
