package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"quantlab/internal/config"
	"quantlab/internal/exchange/bitget"
	"quantlab/internal/market"
	sqlitepkg "quantlab/internal/store/sqlite"
)

func main() {
	fs := flag.NewFlagSet("marketd", flag.ContinueOnError)
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

func run(ctx context.Context, cfg config.Config) error {
	store, err := sqlitepkg.NewStore(cfg.Live.Runtime.StateDBPath)
	if err != nil {
		return err
	}
	_ = store
	feed := market.NewPublicFeed(nil, market.PublicDecoder(bitget.DecodePublicEvents), market.NewMicroBarAggregator(time.Second))
	for event := range feed.Events(ctx) {
		if err := store.AppendEvent(ctx, event, nil); err != nil {
			return err
		}
	}
	return ctx.Err()
}
