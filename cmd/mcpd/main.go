package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"quantlab/internal/adapters"
	"quantlab/internal/backtest"
	"quantlab/internal/config"
	"quantlab/internal/exchange/bitget"
	"quantlab/internal/platform/live"
	"quantlab/internal/platform/mcp"
	"quantlab/internal/platform/promotion"
	"quantlab/internal/platform/query"
	sqlitepkg "quantlab/internal/store/sqlite"
	"quantlab/internal/strategybundle"
)

func main() {
	fs := flag.NewFlagSet("mcpd", flag.ContinueOnError)
	configPath := fs.String("config", "configs/demo-mstr-e2e.yaml", "config file")
	stateDBPath := fs.String("state-db", "", "override sqlite state db path")
	execdPath := fs.String("execd-path", "./execd", "execd binary path for live ops")
	writeToken := fs.String("write-token", "", "write auth token for quant_write tools")
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
	registry := strategybundle.NewRegistry(filepath.Clean(filepath.Join(filepath.Dir(*configPath), "..", "strategies")))
	promotions := promotion.NewService(promotion.Config{Store: store, Backtests: backtests})
	var liveOps mcp.LiveOps
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

	server := mcp.NewServer(mcp.Config{
		Store:          store,
		Query:          queries,
		Strategies:     registry,
		Backtests:      backtests,
		Promotions:     promotions,
		Live:           liveOps,
		WriteAuthToken: *writeToken,
	})
	if err := server.Serve(context.Background(), os.Stdin, os.Stdout); err != nil {
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
