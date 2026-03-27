package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"quantlab/internal/config"
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
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, cfg); err != nil && err != context.Canceled {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, cfg config.Config) error {
	if !cfg.Live.Enabled {
		return fmt.Errorf("live.enabled=false")
	}
	<-ctx.Done()
	return ctx.Err()
}
