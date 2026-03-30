package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"quantlab/internal/config"
	"quantlab/internal/exchange/bitget"
	"quantlab/internal/market"
	sqlitepkg "quantlab/internal/store/sqlite"
	"quantlab/internal/strategybundle"
)

func main() {
	fs := flag.NewFlagSet("marketd", flag.ContinueOnError)
	configPath := fs.String("config", "configs/demo-bitget.yaml", "config file")
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
	if len(cfg.Live.Exchange.Symbols) == 0 {
		return fmt.Errorf("live.exchange.symbols is empty")
	}
	symbol := cfg.Live.Exchange.Symbols[0].Symbol
	interval := cfg.Stream.Interval
	if interval == "" {
		interval = "1m"
	}
	bootstrapLimit := cfg.Stream.SnapshotLimit
	if bootstrapLimit <= 0 {
		bootstrapLimit = 10
	}
	publicClient := bitget.NewClient(cfg.Live.Exchange.RESTBaseURL)
	source := bitget.NewPublicWSSource(
		cfg.Live.Exchange.PublicWSURL,
		bitget.PublicSubscription{InstType: cfg.Live.Exchange.ProductType, Channel: "trade", InstID: symbol},
		bitget.PublicSubscription{InstType: cfg.Live.Exchange.ProductType, Channel: "candle" + interval, InstID: symbol},
	)
	var privateClient *bitget.Client
	var privateSource *bitget.PrivateWSSource
	apiKey := os.Getenv(cfg.Live.Exchange.APIKeyEnv)
	apiSecret := os.Getenv(cfg.Live.Exchange.APISecretEnv)
	passphrase := os.Getenv(cfg.Live.Exchange.PassphraseEnv)
	if apiKey != "" && apiSecret != "" && passphrase != "" {
		creds := bitget.PrivateCredentials{Key: apiKey, Secret: apiSecret, Passphrase: passphrase}
		privateClient = bitget.NewPrivateClient(cfg.Live.Exchange.RESTBaseURL, creds)
		privateSource = bitget.NewPrivateWSSource(
			cfg.Live.Exchange.PrivateWSURL,
			creds,
			bitget.PrivateSubscription{InstType: cfg.Live.Exchange.ProductType, Channel: "orders", InstID: "default"},
			bitget.PrivateSubscription{InstType: cfg.Live.Exchange.ProductType, Channel: "positions", InstID: "default"},
			bitget.PrivateSubscription{InstType: cfg.Live.Exchange.ProductType, Channel: "account", Coin: "default"},
		)
	}
	runtime := market.NewRuntime(market.RuntimeConfig{
		Symbol:         symbol,
		ProductType:    cfg.Live.Exchange.ProductType,
		MarginCoin:     marginCoinForProductType(cfg.Live.Exchange.ProductType),
		Interval:       interval,
		BootstrapLimit: bootstrapLimit,
		AppendEvent: func(ctx context.Context, source string, evt market.MarketEvent, raw []byte) (int64, error) {
			return store.AppendEvent(ctx, source, evt, raw)
		},
		Loader:         publicClient,
		PrivateLoader:  privateClient,
		Source:         source,
		Decoder:        bitget.NewPublicWSDecoder(),
		PrivateSource:  privateSource,
		PrivateDecoder: market.PrivateDecoder(bitget.DecodePrivateEvents),
	})
	return runtime.Run(ctx)
}

func marginCoinForProductType(productType string) string {
	parts := strings.SplitN(productType, "-", 2)
	if len(parts) == 0 || parts[0] == "" {
		return "USDT"
	}
	return parts[0]
}
