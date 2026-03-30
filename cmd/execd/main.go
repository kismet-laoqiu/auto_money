package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"quantlab/internal/exchange/bitget"
	"quantlab/internal/execution"
	sqlitepkg "quantlab/internal/store/sqlite"
	"quantlab/internal/strategybundle"
)

func main() {
	fs := flag.NewFlagSet("execd", flag.ContinueOnError)
	configPath := fs.String("config", "configs/live.yaml", "config file")
	stateDBPath := fs.String("state-db", "", "override sqlite state db path")
	once := fs.Bool("once", false, "process currently available intents and exit")
	flattenSymbol := fs.String("flatten-symbol", "", "close current position for symbol and exit")
	if err := fs.Parse(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	cfg, _, err := strategybundle.LoadConfig(*configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if !cfg.Live.Enabled {
		fmt.Fprintln(os.Stderr, "live.enabled=false")
		os.Exit(1)
	}
	if *stateDBPath == "" {
		*stateDBPath = cfg.Live.Runtime.StateDBPath
	}
	apiKey := os.Getenv(cfg.Live.Exchange.APIKeyEnv)
	apiSecret := os.Getenv(cfg.Live.Exchange.APISecretEnv)
	passphrase := os.Getenv(cfg.Live.Exchange.PassphraseEnv)
	if apiKey == "" || apiSecret == "" || passphrase == "" {
		fmt.Fprintln(os.Stderr, "missing Bitget credentials")
		os.Exit(1)
	}

	store, err := sqlitepkg.NewStore(*stateDBPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	client := bitget.NewPrivateClient(cfg.Live.Exchange.RESTBaseURL, bitget.PrivateCredentials{
		Key:        apiKey,
		Secret:     apiSecret,
		Passphrase: passphrase,
	})
	runtime := execution.NewRuntime(execution.RuntimeConfig{
		Store:       store,
		Exchange:    client,
		ProductType: cfg.Live.Exchange.ProductType,
		MarginMode:  cfg.Live.Exchange.MarginMode,
		MarginCoin:  "USDT",
	})

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if *flattenSymbol != "" {
		if err := runtime.FlattenSymbol(ctx, *flattenSymbol); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if *once {
		if err := runtime.ProcessAvailable(ctx); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if err := runtime.Run(ctx); err != nil && err != context.Canceled {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
