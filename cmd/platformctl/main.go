package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"quantlab/internal/adapters"
	"quantlab/internal/agent"
	platformjobs "quantlab/internal/platform/jobs"
	"quantlab/internal/platform/notifier"
	"quantlab/internal/strategybundle"
	"quantlab/internal/warehouse/catalog"
	warehouseexport "quantlab/internal/warehouse/export"
	"quantlab/internal/warehouse/ingest"
	"quantlab/internal/watchlist"
)

type warehouseConfig = catalog.Config

type warehouseHealthStatus = catalog.HealthStatus

var loadWarehouseConfig = catalog.LoadConfig
var runWarehouseMigrate = catalog.Migrate
var runWarehouseHealth = catalog.Health
var loadWatchlistFile = watchlist.Load
var loadRuntimeConfig = strategybundle.LoadConfig

var defaultHistoricalSymbols = []string{
	"BTCUSDT",
	"ETHUSDT",
	"SOLUSDT",
	"MSTRUSDT",
	"CRCLUSDT",
	"HOODUSDT",
	"BABAUSDT",
	"TAOUSDT",
	"EWYUSDT",
	"AAPLUSDT",
	"LINKUSDT",
	"SEIUSDT",
	"WLDUSDT",
	"CLUSDT",
	"DOGEUSDT",
}

type historicalRequest = ingest.Request
type historicalSyncResult = ingest.SyncResult
type historicalDatasetReport = ingest.DatasetReport

type aggregateRequest = ingest.AggregateRequest
type aggregateResult = ingest.AggregateResult
type aggregateDatasetReport = ingest.AggregateDatasetReport

type exportRequest = warehouseexport.ExportRequest
type exportResult = warehouseexport.ExportResult

type researchRequest = platformjobs.Request
type researchResult = platformjobs.Result

type watchlistApplyResult struct {
	WatchlistPath   string               `json:"watchlist_path"`
	LiveConfigPath  string               `json:"live_config_path"`
	LiveSymbols     []string             `json:"live_symbols"`
	Historical      historicalSyncResult `json:"historical"`
	Aggregated      *aggregateResult     `json:"aggregated,omitempty"`
	ResolvedProduct string               `json:"resolved_product_type"`
}

var platformctlNow = time.Now

var runResearchJob = func(ctx context.Context, request researchRequest) (researchResult, error) {
	client := agent.NewHTTPResponsesClient(agent.HTTPClientConfig{
		APIKey:  os.Getenv("OPENAI_API_KEY"),
		BaseURL: os.Getenv("OPENAI_BASE_URL"),
	})
	service := agent.NewService(client, agent.Config{Store: true})
	job := platformjobs.NewResearchJob(platformjobs.Config{Service: service, Now: platformctlNow})
	return job.Run(ctx, request)
}

var runHistoricalSync = func(ctx context.Context, cfg warehouseConfig, request historicalRequest) (historicalSyncResult, error) {
	db, err := catalog.Open(cfg)
	if err != nil {
		return historicalSyncResult{}, err
	}
	defer db.Close()
	job := ingest.NewHistoricalJob(ingest.JobConfig{
		Fetcher: adapters.NewClient(),
		Store:   ingest.NewPostgresStore(db),
	})
	return job.Sync(ctx, request)
}

var runAggregateSync = func(ctx context.Context, cfg warehouseConfig, request aggregateRequest) (aggregateResult, error) {
	db, err := catalog.Open(cfg)
	if err != nil {
		return aggregateResult{}, err
	}
	defer db.Close()
	job := ingest.NewAggregateJob(ingest.AggregateJobConfig{
		Store: ingest.NewPostgresStore(db),
	})
	return job.Run(ctx, request)
}

var runParquetExport = func(ctx context.Context, cfg warehouseConfig, request exportRequest) (exportResult, error) {
	db, err := catalog.Open(cfg)
	if err != nil {
		return exportResult{}, err
	}
	defer db.Close()
	exporter, err := warehouseexport.NewParquetExporter(warehouseexport.ExporterConfig{
		Source: ingest.NewPostgresStore(db),
		Now:    platformctlNow,
	})
	if err != nil {
		return exportResult{}, err
	}
	return exporter.Export(ctx, request)
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	var err error
	switch os.Args[1] {
	case "status":
		err = runAPI(os.Args[2:], "/api/status")
	case "positions":
		err = runAPI(os.Args[2:], "/api/positions")
	case "orders":
		err = runAPI(os.Args[2:], "/api/orders")
	case "events":
		err = runAPI(os.Args[2:], "/api/events")
	case "backtest":
		err = runBacktest(os.Args[2:])
	case "research":
		err = runResearch(os.Args[2:])
	case "historical":
		err = runHistorical(os.Args[2:])
	case "watchlist":
		err = runWatchlist(os.Args[2:])
	case "aggregate":
		err = runAggregate(os.Args[2:])
	case "export":
		err = runExport(os.Args[2:])
	case "strategy":
		err = runStrategy(os.Args[2:])
	case "promotion":
		err = runPromotion(os.Args[2:])
	case "live":
		err = runLive(os.Args[2:])
	case "notify":
		err = runNotify(os.Args[2:])
	case "warehouse":
		err = runWarehouse(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: platformctl <status|positions|orders|events|backtest|research|historical|watchlist|aggregate|export|strategy|promotion|live|notify|warehouse> [flags]")
}

func runAPI(args []string, path string) error {
	fs := flag.NewFlagSet(path, flag.ContinueOnError)
	baseURL := fs.String("addr", "http://127.0.0.1:8080", "platformd base url")
	limit := fs.Int("limit", 20, "limit for orders/events")
	if err := fs.Parse(args); err != nil {
		return err
	}
	target := strings.TrimRight(*baseURL, "/") + path
	if path == "/api/orders" || path == "/api/events" {
		target = fmt.Sprintf("%s?limit=%d", target, *limit)
	}
	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, target, nil)
	if err != nil {
		return err
	}
	response, err := (&http.Client{Timeout: 10 * time.Second}).Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		body, _ := io.ReadAll(response.Body)
		return fmt.Errorf("platform api status=%d body=%s", response.StatusCode, strings.TrimSpace(string(body)))
	}
	var payload any
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return err
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(payload)
}

func runNotify(args []string) error {
	if len(args) == 0 || args[0] != "test" {
		return fmt.Errorf("usage: platformctl notify test --channel <telegram|dingtalk|all> [--message text]")
	}
	fs := flag.NewFlagSet("notify test", flag.ContinueOnError)
	channel := fs.String("channel", "all", "telegram|dingtalk|all")
	message := fs.String("message", "platform notify test", "notification message")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	service := notifier.New(notifier.Config{
		DingTalkWebhook:   os.Getenv("DINGTALK_WEBHOOK"),
		DingTalkSecret:    os.Getenv("DINGTALK_SECRET"),
		TelegramBotToken:  os.Getenv("TELEGRAM_BOT_TOKEN"),
		TelegramAlertChat: firstNonEmpty(os.Getenv("TELEGRAM_ALERT_CHAT_ID"), os.Getenv("TELEGRAM_COMMAND_CHAT_ID")),
	})

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	switch *channel {
	case "telegram":
		return service.Send(ctx, notifier.ChannelTelegram, *message)
	case "dingtalk":
		return service.Send(ctx, notifier.ChannelDingTalk, *message)
	case "all":
		if err := service.Send(ctx, notifier.ChannelTelegram, *message); err != nil {
			return err
		}
		return service.Send(ctx, notifier.ChannelDingTalk, *message)
	default:
		return fmt.Errorf("unsupported channel %q", *channel)
	}
}

func runBacktest(args []string) error {
	if len(args) == 0 || args[0] != "run" {
		return fmt.Errorf("usage: platformctl backtest run --config <path> [--refresh] [--addr url]")
	}
	fs := flag.NewFlagSet("backtest run", flag.ContinueOnError)
	baseURL := fs.String("addr", "http://127.0.0.1:8080", "platformd base url")
	configPath := fs.String("config", "", "backtest config path")
	refresh := fs.Bool("refresh", false, "refresh remote data")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	if *configPath == "" {
		return fmt.Errorf("config is required")
	}
	body, err := json.Marshal(map[string]any{
		"config_path": *configPath,
		"refresh":     *refresh,
	})
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(context.Background(), http.MethodPost, strings.TrimRight(*baseURL, "/")+"/api/backtests/run", strings.NewReader(string(body)))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := (&http.Client{Timeout: 10 * time.Second}).Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		respBody, _ := io.ReadAll(response.Body)
		return fmt.Errorf("platform api status=%d body=%s", response.StatusCode, strings.TrimSpace(string(respBody)))
	}
	var payload any
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return err
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(payload)
}

func runStrategy(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: platformctl strategy versions [--strategy id] [--addr url]")
	}
	switch args[0] {
	case "versions":
		fs := flag.NewFlagSet("strategy versions", flag.ContinueOnError)
		baseURL := fs.String("addr", "http://127.0.0.1:8080", "platformd base url")
		strategyID := fs.String("strategy", "", "strategy id")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		target := strings.TrimRight(*baseURL, "/") + "/api/strategies/versions"
		if *strategyID != "" {
			target += "?strategy_id=" + url.QueryEscape(*strategyID)
		}
		return runJSONRequest(http.MethodGet, target, nil)
	default:
		return fmt.Errorf("unsupported strategy subcommand %q", args[0])
	}
}

func runPromotion(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: platformctl promotion <request|list|get|start-shadow|pass-shadow|start-canary|degrade-canary|approve|rollback> [flags]")
	}
	switch args[0] {
	case "request":
		fs := flag.NewFlagSet("promotion request", flag.ContinueOnError)
		baseURL := fs.String("addr", "http://127.0.0.1:8080", "platformd base url")
		strategyID := fs.String("strategy", "", "strategy id")
		version := fs.String("version", "", "strategy version")
		configPath := fs.String("config", "", "backtest config path")
		refresh := fs.Bool("refresh", false, "refresh remote data")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		body := map[string]any{
			"strategy_id": *strategyID,
			"version":     *version,
			"config_path": *configPath,
			"refresh":     *refresh,
		}
		return runJSONRequest(http.MethodPost, strings.TrimRight(*baseURL, "/")+"/api/promotions/request", body)
	case "list":
		fs := flag.NewFlagSet("promotion list", flag.ContinueOnError)
		baseURL := fs.String("addr", "http://127.0.0.1:8080", "platformd base url")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		return runJSONRequest(http.MethodGet, strings.TrimRight(*baseURL, "/")+"/api/promotions", nil)
	case "get":
		fs := flag.NewFlagSet("promotion get", flag.ContinueOnError)
		baseURL := fs.String("addr", "http://127.0.0.1:8080", "platformd base url")
		id := fs.String("id", "", "promotion id")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		return runJSONRequest(http.MethodGet, strings.TrimRight(*baseURL, "/")+"/api/promotions?id="+*id, nil)
	case "start-shadow":
		return runPromotionAction(args[1:], "/api/promotions/shadow", false)
	case "pass-shadow":
		return runPromotionAction(args[1:], "/api/promotions/shadow/pass", false)
	case "start-canary":
		return runPromotionAction(args[1:], "/api/promotions/canary", false)
	case "degrade-canary":
		return runPromotionAction(args[1:], "/api/promotions/canary/degrade", true)
	case "approve":
		return runPromotionAction(args[1:], "/api/promotions/approve", false)
	case "rollback":
		return runPromotionAction(args[1:], "/api/promotions/rollback", true)
	default:
		return fmt.Errorf("unsupported promotion subcommand %q", args[0])
	}
}

func runLive(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: platformctl live flatten --symbol <symbol> [--addr url]")
	}
	switch args[0] {
	case "flatten":
		fs := flag.NewFlagSet("live flatten", flag.ContinueOnError)
		baseURL := fs.String("addr", "http://127.0.0.1:8080", "platformd base url")
		symbol := fs.String("symbol", "", "symbol to flatten")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if *symbol == "" {
			return fmt.Errorf("symbol is required")
		}
		return runJSONRequest(http.MethodPost, strings.TrimRight(*baseURL, "/")+"/api/live/flatten", map[string]any{"symbol": *symbol})
	default:
		return fmt.Errorf("unsupported live subcommand %q", args[0])
	}
}

func runHistorical(args []string) error {
	if len(args) == 0 || args[0] != "sync" {
		return fmt.Errorf("usage: platformctl historical sync [flags]")
	}
	fs := flag.NewFlagSet("historical sync", flag.ContinueOnError)
	warehouseConfigPath := fs.String("warehouse-config", "configs/platform/warehouse.yaml", "warehouse config path")
	watchlistPath := fs.String("watchlist", "configs/platform/watchlist.yaml", "watchlist file path")
	provider := fs.String("provider", "bitget", "market data provider")
	productType := fs.String("product-type", "", "product type for provider-specific endpoints")
	symbols := fs.String("symbols", "", "comma separated symbols")
	intervals := fs.String("intervals", "", "comma separated intervals")
	limit := fs.Int("limit", 1000, "bars per request")
	horizonDays := fs.Int("horizon-days", 0, "historical backfill horizon in days")
	artifactRoot := fs.String("artifact-root", "artifacts/platform/historical-sync", "artifact output root")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	cfg, err := loadWarehouseConfig(*warehouseConfigPath)
	if err != nil {
		return err
	}
	result, err := runHistoricalSync(context.Background(), cfg, historicalRequest{
		Provider:     *provider,
		ProductType:  *productType,
		Symbols:      historicalSymbols(*symbols, *watchlistPath),
		Intervals:    splitCSV(*intervals),
		Limit:        *limit,
		HorizonDays:  *horizonDays,
		ArtifactRoot: *artifactRoot,
	})
	if err != nil {
		return err
	}
	return printJSON(result)
}

func runWatchlist(args []string) error {
	if len(args) == 0 || args[0] != "apply" {
		return fmt.Errorf("usage: platformctl watchlist apply [flags]")
	}
	fs := flag.NewFlagSet("watchlist apply", flag.ContinueOnError)
	watchlistPath := fs.String("watchlist", "configs/platform/watchlist.yaml", "watchlist file path")
	liveConfigPath := fs.String("live-config", "configs/live.yaml", "live config file path")
	warehouseConfigPath := fs.String("warehouse-config", "configs/platform/warehouse.yaml", "warehouse config path")
	limit := fs.Int("limit", 200, "bars per request")
	artifactRoot := fs.String("artifact-root", "artifacts/platform/historical-sync", "artifact output root")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	file, err := loadWatchlistFile(*watchlistPath)
	if err != nil {
		return err
	}
	cfg, err := loadWarehouseConfig(*warehouseConfigPath)
	if err != nil {
		return err
	}
	historical, err := runHistoricalSync(context.Background(), cfg, historicalRequest{
		Provider:     file.Provider,
		ProductType:  file.ProductType,
		Symbols:      file.SymbolNames(),
		Intervals:    watchlistHistoricalIntervals(file.HistoricalIntervals),
		Limit:        *limit,
		HorizonDays:  file.HorizonDays,
		ArtifactRoot: *artifactRoot,
	})
	if err != nil {
		return err
	}
	var aggregated *aggregateResult
	aggregateIntervals := watchlistAggregateIntervals(file.HistoricalIntervals)
	if len(aggregateIntervals) > 0 {
		result, err := runAggregateSync(context.Background(), cfg, aggregateRequest{
			Provider:    file.Provider,
			ProductType: file.ProductType,
			Symbols:     file.SymbolNames(),
			Intervals:   aggregateIntervals,
		})
		if err != nil {
			return err
		}
		aggregated = &result
	}
	resolved, _, err := loadRuntimeConfig(*liveConfigPath)
	if err != nil {
		return err
	}
	result := watchlistApplyResult{
		WatchlistPath:   *watchlistPath,
		LiveConfigPath:  *liveConfigPath,
		LiveSymbols:     make([]string, 0, len(resolved.Live.Exchange.Symbols)),
		Historical:      historical,
		Aggregated:      aggregated,
		ResolvedProduct: resolved.Live.Exchange.ProductType,
	}
	for _, item := range resolved.Live.Exchange.Symbols {
		if item.Symbol == "" {
			continue
		}
		result.LiveSymbols = append(result.LiveSymbols, item.Symbol)
	}
	return printJSON(result)
}

func watchlistAggregateIntervals(intervals []string) []string {
	out := make([]string, 0, len(intervals))
	seen := map[string]bool{}
	for _, interval := range intervals {
		switch normalized := strings.ToLower(strings.TrimSpace(interval)); normalized {
		case "1h", "4h", "1d", "1w":
			if seen[normalized] {
				continue
			}
			seen[normalized] = true
			out = append(out, normalized)
		}
	}
	return out
}

func watchlistHistoricalIntervals(intervals []string) []string {
	for _, interval := range intervals {
		if strings.EqualFold(strings.TrimSpace(interval), "15m") {
			return []string{"15m"}
		}
	}
	return append([]string(nil), intervals...)
}

func runAggregate(args []string) error {
	if len(args) == 0 || args[0] != "run" {
		return fmt.Errorf("usage: platformctl aggregate run [flags]")
	}
	fs := flag.NewFlagSet("aggregate run", flag.ContinueOnError)
	warehouseConfigPath := fs.String("warehouse-config", "configs/platform/warehouse.yaml", "warehouse config path")
	provider := fs.String("provider", "bitget", "market data provider")
	productType := fs.String("product-type", "", "product type for provider-specific endpoints")
	symbols := fs.String("symbols", "", "comma separated symbols")
	intervals := fs.String("intervals", "", "comma separated target intervals")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	normalizedSymbols := splitCSV(*symbols)
	if len(normalizedSymbols) == 0 {
		return fmt.Errorf("symbols are required")
	}
	cfg, err := loadWarehouseConfig(*warehouseConfigPath)
	if err != nil {
		return err
	}
	result, err := runAggregateSync(context.Background(), cfg, aggregateRequest{
		Provider:    *provider,
		ProductType: *productType,
		Symbols:     normalizedSymbols,
		Intervals:   splitCSV(*intervals),
	})
	if err != nil {
		return err
	}
	return printJSON(result)
}

func runExport(args []string) error {
	if len(args) == 0 || args[0] != "parquet" {
		return fmt.Errorf("usage: platformctl export parquet [flags]")
	}
	fs := flag.NewFlagSet("export parquet", flag.ContinueOnError)
	warehouseConfigPath := fs.String("warehouse-config", "configs/platform/warehouse.yaml", "warehouse config path")
	provider := fs.String("provider", "bitget", "market data provider")
	symbols := fs.String("symbols", "", "comma separated symbols")
	intervals := fs.String("intervals", "", "comma separated intervals")
	artifactRoot := fs.String("artifact-root", "artifacts/platform/export", "artifact output root")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	normalizedSymbols := splitCSV(*symbols)
	if len(normalizedSymbols) == 0 {
		return fmt.Errorf("symbols are required")
	}
	normalizedIntervals := splitCSV(*intervals)
	if len(normalizedIntervals) == 0 {
		return fmt.Errorf("intervals are required")
	}
	cfg, err := loadWarehouseConfig(*warehouseConfigPath)
	if err != nil {
		return err
	}
	rootDir := filepath.Join(*artifactRoot, platformctlNow().UTC().Format("20060102T150405Z"))
	result, err := runParquetExport(context.Background(), cfg, exportRequest{
		Provider:  *provider,
		Symbols:   normalizedSymbols,
		Intervals: normalizedIntervals,
		RootDir:   rootDir,
	})
	if err != nil {
		return err
	}
	result.LatestRootDir = result.RootDir
	body, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(*artifactRoot, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(*artifactRoot, "summary.json"), append(body, 0x0a), 0o644); err != nil {
		return err
	}
	return printJSON(result)
}

func runWarehouse(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: platformctl warehouse <migrate|health> [flags]")
	}
	switch args[0] {
	case "migrate":
		fs := flag.NewFlagSet("warehouse migrate", flag.ContinueOnError)
		configPath := fs.String("config", "configs/platform/warehouse.yaml", "warehouse config path")
		migrationsDir := fs.String("migrations", "migrations/postgres", "warehouse migrations directory")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		cfg, err := loadWarehouseConfig(*configPath)
		if err != nil {
			return err
		}
		applied, err := runWarehouseMigrate(context.Background(), cfg, *migrationsDir)
		if err != nil {
			return err
		}
		return printJSON(map[string]any{"applied": len(applied), "migrations": applied})
	case "health":
		fs := flag.NewFlagSet("warehouse health", flag.ContinueOnError)
		configPath := fs.String("config", "configs/platform/warehouse.yaml", "warehouse config path")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		cfg, err := loadWarehouseConfig(*configPath)
		if err != nil {
			return err
		}
		status, err := runWarehouseHealth(context.Background(), cfg)
		if err != nil {
			return err
		}
		return printJSON(status)
	default:
		return fmt.Errorf("unsupported warehouse subcommand %q", args[0])
	}
}

func runPromotionAction(args []string, path string, withReason bool) error {
	fs := flag.NewFlagSet(path, flag.ContinueOnError)
	baseURL := fs.String("addr", "http://127.0.0.1:8080", "platformd base url")
	id := fs.String("id", "", "promotion id")
	reason := fs.String("reason", "", "action reason")
	if err := fs.Parse(args); err != nil {
		return err
	}
	body := map[string]any{"id": *id}
	if withReason {
		body["reason"] = *reason
	}
	return runJSONRequest(http.MethodPost, strings.TrimRight(*baseURL, "/")+path, body)
}

func runJSONRequest(method, target string, body any) error {
	var requestBody io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return err
		}
		requestBody = strings.NewReader(string(payload))
	}
	request, err := http.NewRequestWithContext(context.Background(), method, target, requestBody)
	if err != nil {
		return err
	}
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := (&http.Client{Timeout: 10 * time.Second}).Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		respBody, _ := io.ReadAll(response.Body)
		return fmt.Errorf("platform api status=%d body=%s", response.StatusCode, strings.TrimSpace(string(respBody)))
	}
	var payload any
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return err
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(payload)
}

func historicalSymbols(value, watchlistPath string) []string {
	symbols := splitCSV(value)
	if len(symbols) > 0 {
		return symbols
	}
	if watchlistPath != "" {
		file, err := loadWatchlistFile(watchlistPath)
		if err == nil {
			symbols = file.SymbolNames()
			if len(symbols) > 0 {
				return symbols
			}
		}
	}
	return append([]string(nil), defaultHistoricalSymbols...)
}

func splitCSV(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func printJSON(payload any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(payload)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func runResearch(args []string) error {
	if len(args) == 0 || args[0] != "run" {
		return fmt.Errorf("usage: platformctl research run --kind <kind> [flags]")
	}
	fs := flag.NewFlagSet("research run", flag.ContinueOnError)
	kind := fs.String("kind", "", "research job kind")
	strategy := fs.String("strategy", "", "strategy id")
	subject := fs.String("subject", "", "job subject")
	body := fs.String("body", "", "job body override")
	configPath := fs.String("config-path", "", "related config path")
	baselineRunID := fs.String("baseline-run", "", "baseline backtest run id")
	candidateRunID := fs.String("candidate-run", "", "candidate backtest run id")
	datasets := fs.String("datasets", "", "comma separated dataset descriptors")
	artifactRoot := fs.String("artifact-root", "artifacts/platform/research", "artifact output root")
	pollInterval := fs.Duration("poll-interval", 2*time.Second, "poll interval")
	timeout := fs.Duration("timeout", 2*time.Minute, "overall timeout")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	if *kind == "" {
		return fmt.Errorf("kind is required")
	}
	result, err := runResearchJob(context.Background(), researchRequest{
		Kind:           platformjobs.Kind(*kind),
		StrategyID:     *strategy,
		Subject:        *subject,
		Body:           *body,
		ConfigPath:     *configPath,
		BaselineRunID:  *baselineRunID,
		CandidateRunID: *candidateRunID,
		Datasets:       splitCSV(*datasets),
		ArtifactRoot:   *artifactRoot,
		PollInterval:   *pollInterval,
		Timeout:        *timeout,
	})
	if err != nil {
		return err
	}
	return printJSON(result)
}
