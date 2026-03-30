package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"quantlab/internal/adapters"
	"quantlab/internal/strategybundle"
)

func main() {
	fs := flag.NewFlagSet("stream", flag.ContinueOnError)
	configPath := fs.String("config", "configs/baseline.yaml", "config file")
	if err := fs.Parse(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	cfg, _, err := strategybundle.LoadConfig(*configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if err := adapters.RunBitgetStream(ctx, cfg); err != nil && err != context.Canceled {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
