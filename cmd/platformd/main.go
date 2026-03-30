package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"quantlab/internal/adapters"
	"quantlab/internal/backtest"
	"quantlab/internal/config"
	"quantlab/internal/exchange/bitget"
	"quantlab/internal/platform/api"
	"quantlab/internal/platform/live"
	"quantlab/internal/platform/promotion"
	"quantlab/internal/platform/query"
	sqlitepkg "quantlab/internal/store/sqlite"
	"quantlab/internal/strategybundle"
)

func main() {
	fs := flag.NewFlagSet("platformd", flag.ContinueOnError)
	configPath := fs.String("config", "configs/live.yaml", "config file")
	listenAddr := fs.String("listen", ":8080", "listen address")
	stateDBPath := fs.String("state-db", "", "override sqlite state db path")
	execdPath := fs.String("execd-path", "./execd", "execd binary path for live ops")
	if err := fs.Parse(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	cfg, _, err := strategybundle.LoadConfig(*configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if *stateDBPath == "" {
		*stateDBPath = cfg.Live.Runtime.StateDBPath
	}

	store, err := sqlitepkg.NewStore(*stateDBPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	backtests := backtest.NewService(backtest.Config{Loader: adapters.NewClient(), Store: store})
	queries := query.NewService(query.Config{Loader: adapters.NewClient()})
	var liveOps api.LiveOps
	if cfg.Live.Enabled {
		apiKey := os.Getenv(cfg.Live.Exchange.APIKeyEnv)
		apiSecret := os.Getenv(cfg.Live.Exchange.APISecretEnv)
		passphrase := os.Getenv(cfg.Live.Exchange.PassphraseEnv)
		if apiKey != "" && apiSecret != "" && passphrase != "" {
			reader := bitget.NewPrivateClient(cfg.Live.Exchange.RESTBaseURL, bitget.PrivateCredentials{
				Key:        apiKey,
				Secret:     apiSecret,
				Passphrase: passphrase,
			})
			liveOps = live.NewService(live.Config{
				ExecdPath:      *execdPath,
				ConfigPath:     *configPath,
				StateDBPath:    *stateDBPath,
				ProductType:    cfg.Live.Exchange.ProductType,
				MarginCoin:     "USDT",
				AllowedSymbols: allowedSymbols(cfg.Live.Exchange.Symbols),
				Reader:         reader,
			})
		}
	}
	registry := strategybundle.NewRegistry(filepath.Clean(filepath.Join(filepath.Dir(*configPath), "..", "strategies")))
	server := &http.Server{
		Addr: *listenAddr,
		Handler: api.NewHandler(api.HandlerConfig{
			Store:      store,
			Backtests:  backtests,
			Promotions: promotion.NewService(promotion.Config{Store: store, Backtests: backtests}),
			Strategies: registry,
			Query:      queries,
			Live:       liveOps,
		}),
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		_ = server.Shutdown(context.Background())
	}()
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func allowedSymbols(symbols []config.LiveSymbolConfig) []string {
	out := make([]string, 0, len(symbols))
	for _, symbol := range symbols {
		if symbol.Symbol == "" {
			continue
		}
		out = append(out, symbol.Symbol)
	}
	return out
}
