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
	"quantlab/internal/trader"
)

func main() {
	fs := flag.NewFlagSet("traderd", flag.ContinueOnError)
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
	_, err := sqlitepkg.NewStore(cfg.Live.Runtime.StateDBPath)
	if err != nil {
		return err
	}
	engine := trader.NewEngine(trader.Config{ArmingState: trader.ArmingState(cfg.Live.Runtime.ArmingState)})
	_ = engine
	<-ctx.Done()
	return ctx.Err()
}
