package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"quantlab/internal/config"
	"quantlab/internal/exchange/bitget"
	"quantlab/internal/market"
	sqlitepkg "quantlab/internal/store/sqlite"
	"quantlab/internal/strategybundle"
)

type marketRuntimeRunner interface {
	Run(context.Context) error
}

var newSQLiteStore = sqlitepkg.NewStore
var newPublicBitgetClient = func(baseURL string) market.BootstrapLoader {
	return bitget.NewClient(baseURL)
}
var newPrivateBitgetClient = func(baseURL string, creds bitget.PrivateCredentials) *bitget.Client {
	return bitget.NewPrivateClient(baseURL, creds)
}
var newPublicWSSource = func(url string, productType string, interval string, symbol string) market.EventSource {
	return bitget.NewPublicWSSource(
		url,
		bitget.PublicSubscription{InstType: productType, Channel: "trade", InstID: symbol},
		bitget.PublicSubscription{InstType: productType, Channel: "candle" + interval, InstID: symbol},
	)
}
var newPrivateWSSource = func(url string, creds bitget.PrivateCredentials, productType string) market.EventSource {
	return bitget.NewPrivateWSSource(
		url,
		creds,
		bitget.PrivateSubscription{InstType: productType, Channel: "orders", InstID: "default"},
		bitget.PrivateSubscription{InstType: productType, Channel: "positions", InstID: "default"},
		bitget.PrivateSubscription{InstType: productType, Channel: "account", Coin: "default"},
	)
}
var newMarketRuntime = func(cfg market.RuntimeConfig) marketRuntimeRunner {
	return market.NewRuntime(cfg)
}

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
	store, err := newSQLiteStore(cfg.Live.Runtime.StateDBPath)
	if err != nil {
		return err
	}
	if len(cfg.Live.Exchange.Symbols) == 0 {
		return fmt.Errorf("live.exchange.symbols is empty")
	}
	interval := cfg.Stream.Interval
	if interval == "" {
		interval = "1m"
	}
	bootstrapLimit := cfg.Stream.SnapshotLimit
	if bootstrapLimit <= 0 {
		bootstrapLimit = 10
	}
	var privateClient *bitget.Client
	var privateSource market.EventSource
	apiKey := os.Getenv(cfg.Live.Exchange.APIKeyEnv)
	apiSecret := os.Getenv(cfg.Live.Exchange.APISecretEnv)
	passphrase := os.Getenv(cfg.Live.Exchange.PassphraseEnv)
	if apiKey != "" && apiSecret != "" && passphrase != "" {
		creds := bitget.PrivateCredentials{Key: apiKey, Secret: apiSecret, Passphrase: passphrase}
		privateClient = newPrivateBitgetClient(cfg.Live.Exchange.RESTBaseURL, creds)
		privateSource = newPrivateWSSource(cfg.Live.Exchange.PrivateWSURL, creds, cfg.Live.Exchange.ProductType)
	}
	appendEvent := func(ctx context.Context, source string, evt market.MarketEvent, raw []byte) (int64, error) {
		return store.AppendEvent(ctx, source, evt, raw)
	}

	var runtimes []marketRuntimeRunner
	for _, item := range cfg.Live.Exchange.Symbols {
		symbol := item.Symbol
		if symbol == "" {
			continue
		}
		runtimes = append(runtimes, newMarketRuntime(market.RuntimeConfig{
			Symbol:         symbol,
			ProductType:    cfg.Live.Exchange.ProductType,
			MarginCoin:     marginCoinForProductType(cfg.Live.Exchange.ProductType),
			Interval:       interval,
			BootstrapLimit: bootstrapLimit,
			AppendEvent:    appendEvent,
			Loader:         newPublicBitgetClient(cfg.Live.Exchange.RESTBaseURL),
			Source:         newPublicWSSource(cfg.Live.Exchange.PublicWSURL, cfg.Live.Exchange.ProductType, interval, symbol),
			Decoder:        bitget.NewPublicWSDecoder(),
		}))
	}
	if privateClient != nil && privateSource != nil {
		runtimes = append(runtimes, newMarketRuntime(market.RuntimeConfig{
			ProductType:    cfg.Live.Exchange.ProductType,
			MarginCoin:     marginCoinForProductType(cfg.Live.Exchange.ProductType),
			Interval:       interval,
			BootstrapLimit: bootstrapLimit,
			AppendEvent:    appendEvent,
			PrivateLoader:  privateClient,
			PrivateSource:  privateSource,
			PrivateDecoder: market.PrivateDecoder(bitget.DecodePrivateEvents),
		}))
	}
	if len(runtimes) == 0 {
		return fmt.Errorf("no market runtimes configured")
	}

	errCh := make(chan error, len(runtimes))
	var wg sync.WaitGroup
	for _, runtime := range runtimes {
		wg.Add(1)
		go func(runtime marketRuntimeRunner) {
			defer wg.Done()
			errCh <- runtime.Run(ctx)
		}(runtime)
	}
	go func() {
		wg.Wait()
		close(errCh)
	}()
	for err := range errCh {
		if err != nil && err != context.Canceled {
			return err
		}
	}
	return nil
}

func marginCoinForProductType(productType string) string {
	parts := strings.SplitN(productType, "-", 2)
	if len(parts) == 0 || parts[0] == "" {
		return "USDT"
	}
	return parts[0]
}
