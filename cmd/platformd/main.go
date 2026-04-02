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
	"quantlab/internal/platform/dashboard"
	"quantlab/internal/platform/insights"
	"quantlab/internal/platform/live"
	"quantlab/internal/platform/promotion"
	"quantlab/internal/platform/query"
	"quantlab/internal/platform/truth"
	watchlistsvc "quantlab/internal/platform/watchlist"
	sqlitepkg "quantlab/internal/store/sqlite"
	"quantlab/internal/strategybundle"
	"quantlab/internal/warehouse/catalog"
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
	resolvedConfigPath := resolveConfigPath(*configPath)

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
	dashboardService, watchlistService, closeWarehouse, err := newDashboardServices(cfg, resolvedConfigPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if closeWarehouse != nil {
		defer closeWarehouse()
	}
	server := &http.Server{
		Addr: *listenAddr,
		Handler: api.NewHandler(api.HandlerConfig{
			Store:      store,
			Backtests:  backtests,
			Promotions: promotion.NewService(promotion.Config{Store: store, Backtests: backtests}),
			Strategies: registry,
			Query:      queries,
			Live:       liveOps,
			Dashboard:  dashboardService,
			Watchlist:  watchlistService,
			Truth: truth.NewStore(truth.StoreConfig{
				SiteFactsPath:      filepath.Clean(filepath.Join(filepath.Dir(*configPath), "platform", "site-facts.yaml")),
				OperatorPolicyPath: filepath.Clean(filepath.Join(filepath.Dir(*configPath), "platform", "operator-policy.yaml")),
				LeadersPath:        filepath.Clean(filepath.Join(filepath.Dir(*configPath), "platform", "leaders.yaml")),
			}),
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

func newDashboardServices(cfg config.Config, configPath string) (*dashboard.Service, *watchlistsvc.Service, func() error, error) {
	if !cfg.Insights.Enabled || cfg.WarehouseConfigPath == "" || cfg.WatchlistPath == "" {
		return nil, nil, nil, nil
	}
	warehouseCfg, err := catalog.LoadConfig(cfg.WarehouseConfigPath)
	if err != nil {
		return nil, nil, nil, err
	}
	db, err := catalog.Open(warehouseCfg)
	if err != nil {
		return nil, nil, nil, err
	}
	closeFn := func() error { return db.Close() }
	insightService := insights.NewService(insights.Config{
		WatchlistPath: cfg.WatchlistPath,
		Provider:      cfg.Live.Exchange.Venue,
		ProductType:   cfg.Live.Exchange.ProductType,
		Strategy:      cfg.Strategy,
		Insights:      cfg.Insights,
		Store:         insights.NewSQLStore(db),
		PriceReader:   bitget.NewClient(cfg.Live.Exchange.RESTBaseURL),
	})
	return dashboard.NewService(dashboard.Config{Insights: insightService}),
		watchlistsvc.NewService(watchlistsvc.Config{
			WatchlistPath:       cfg.WatchlistPath,
			LiveConfigPath:      configPath,
			WarehouseConfigPath: cfg.WarehouseConfigPath,
			PlatformctlPath:     cfg.Insights.Dashboard.PlatformctlPath,
			MarketdRestartUnit:  cfg.Insights.Dashboard.MarketdRestartUnit,
		}),
		closeFn,
		nil
}

func resolveConfigPath(path string) string {
	if path == "" || filepath.IsAbs(path) {
		return path
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return filepath.Clean(absolute)
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
