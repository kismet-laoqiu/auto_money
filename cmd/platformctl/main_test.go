package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"quantlab/internal/config"
	"quantlab/internal/strategybundle"
	"quantlab/internal/watchlist"
)

func TestRunBacktestPostsRequestAndPrintsResponse(t *testing.T) {
	var gotMethod string
	var gotPath string
	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"final_score":0.91}`))
	}))
	defer server.Close()

	originalStdout := os.Stdout
	readPipe, writePipe, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe stdout: %v", err)
	}
	defer readPipe.Close()
	defer writePipe.Close()
	os.Stdout = writePipe
	defer func() { os.Stdout = originalStdout }()

	if err := runBacktest([]string{"run", "-addr", server.URL, "-config", "configs/baseline.yaml", "-refresh"}); err != nil {
		t.Fatalf("run backtest: %v", err)
	}
	_ = writePipe.Close()
	output, err := io.ReadAll(readPipe)
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	if gotMethod != http.MethodPost || gotPath != "/api/backtests/run" {
		t.Fatalf("unexpected request: method=%s path=%s", gotMethod, gotPath)
	}
	if gotBody["config_path"] != "configs/baseline.yaml" || gotBody["refresh"] != true {
		t.Fatalf("unexpected request body: %+v", gotBody)
	}
	if !strings.Contains(string(output), `"final_score": 0.91`) {
		t.Fatalf("unexpected stdout: %s", output)
	}
}

func TestRunPromotionRequestPostsRequestAndPrintsResponse(t *testing.T) {
	var gotMethod string
	var gotPath string
	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"promo-1","state":"backtest_passed"}`))
	}))
	defer server.Close()

	originalStdout := os.Stdout
	readPipe, writePipe, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe stdout: %v", err)
	}
	defer readPipe.Close()
	defer writePipe.Close()
	os.Stdout = writePipe
	defer func() { os.Stdout = originalStdout }()

	if err := runPromotion([]string{"request", "-addr", server.URL, "-strategy", "mstr-wave-fib", "-version", "v0.1.0", "-config", "configs/live.yaml"}); err != nil {
		t.Fatalf("run promotion request: %v", err)
	}
	_ = writePipe.Close()
	output, err := io.ReadAll(readPipe)
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	if gotMethod != http.MethodPost || gotPath != "/api/promotions/request" {
		t.Fatalf("unexpected request: method=%s path=%s", gotMethod, gotPath)
	}
	if gotBody["strategy_id"] != "mstr-wave-fib" || gotBody["version"] != "v0.1.0" || gotBody["config_path"] != "configs/live.yaml" {
		t.Fatalf("unexpected request body: %+v", gotBody)
	}
	if !strings.Contains(string(output), `"state": "backtest_passed"`) {
		t.Fatalf("unexpected stdout: %s", output)
	}
}

func TestRunStrategyVersionsGetsRequestAndPrintsResponse(t *testing.T) {
	var gotMethod string
	var gotPath string
	var gotQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"strategy_id":"mstr-wave-fib","version":"v0.1.0"}]`))
	}))
	defer server.Close()

	originalStdout := os.Stdout
	readPipe, writePipe, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe stdout: %v", err)
	}
	defer readPipe.Close()
	defer writePipe.Close()
	os.Stdout = writePipe
	defer func() { os.Stdout = originalStdout }()

	if err := runStrategy([]string{"versions", "-addr", server.URL, "-strategy", "mstr-wave-fib"}); err != nil {
		t.Fatalf("run strategy versions: %v", err)
	}
	_ = writePipe.Close()
	output, err := io.ReadAll(readPipe)
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	if gotMethod != http.MethodGet || gotPath != "/api/strategies/versions" || gotQuery != "strategy_id=mstr-wave-fib" {
		t.Fatalf("unexpected request: method=%s path=%s query=%s", gotMethod, gotPath, gotQuery)
	}
	if !strings.Contains(string(output), `"version": "v0.1.0"`) {
		t.Fatalf("unexpected stdout: %s", output)
	}
}

func TestRunLiveFlattenPostsRequestAndPrintsResponse(t *testing.T) {
	var gotMethod string
	var gotPath string
	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"symbol":"MSTRUSDT","status":"flat","qty":0}`))
	}))
	defer server.Close()

	originalStdout := os.Stdout
	readPipe, writePipe, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe stdout: %v", err)
	}
	defer readPipe.Close()
	defer writePipe.Close()
	os.Stdout = writePipe
	defer func() { os.Stdout = originalStdout }()

	if err := runLive([]string{"flatten", "-addr", server.URL, "-symbol", "MSTRUSDT"}); err != nil {
		t.Fatalf("run live flatten: %v", err)
	}
	_ = writePipe.Close()
	output, err := io.ReadAll(readPipe)
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	if gotMethod != http.MethodPost || gotPath != "/api/live/flatten" {
		t.Fatalf("unexpected request: method=%s path=%s", gotMethod, gotPath)
	}
	if gotBody["symbol"] != "MSTRUSDT" {
		t.Fatalf("unexpected request body: %+v", gotBody)
	}
	if !strings.Contains(string(output), `"status": "flat"`) {
		t.Fatalf("unexpected stdout: %s", output)
	}
}

func TestRunWarehouseMigrateUsesCatalogRunner(t *testing.T) {
	originalLoad := loadWarehouseConfig
	originalMigrate := runWarehouseMigrate
	defer func() {
		loadWarehouseConfig = originalLoad
		runWarehouseMigrate = originalMigrate
	}()

	var gotConfigPath string
	var gotMigrationsDir string
	loadWarehouseConfig = func(path string) (warehouseConfig, error) {
		gotConfigPath = path
		return warehouseConfig{DSN: "postgres://warehouse"}, nil
	}
	runWarehouseMigrate = func(_ context.Context, cfg warehouseConfig, migrationsDir string) ([]string, error) {
		if cfg.DSN != "postgres://warehouse" {
			t.Fatalf("unexpected config: %+v", cfg)
		}
		gotMigrationsDir = migrationsDir
		return []string{"0001_init_market.sql", "0002_init_strategy.sql"}, nil
	}

	originalStdout := os.Stdout
	readPipe, writePipe, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe stdout: %v", err)
	}
	defer readPipe.Close()
	defer writePipe.Close()
	os.Stdout = writePipe
	defer func() { os.Stdout = originalStdout }()

	if err := runWarehouse([]string{"migrate", "-config", "configs/platform/warehouse.yaml", "-migrations", "migrations/postgres"}); err != nil {
		t.Fatalf("run warehouse migrate: %v", err)
	}
	_ = writePipe.Close()
	output, err := io.ReadAll(readPipe)
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	if gotConfigPath != "configs/platform/warehouse.yaml" || gotMigrationsDir != "migrations/postgres" {
		t.Fatalf("unexpected invocation: config=%s migrations=%s", gotConfigPath, gotMigrationsDir)
	}
	if !strings.Contains(string(output), `"applied": 2`) {
		t.Fatalf("unexpected stdout: %s", output)
	}
}

func TestRunWarehouseHealthPrintsStatus(t *testing.T) {
	originalLoad := loadWarehouseConfig
	originalHealth := runWarehouseHealth
	defer func() {
		loadWarehouseConfig = originalLoad
		runWarehouseHealth = originalHealth
	}()

	loadWarehouseConfig = func(path string) (warehouseConfig, error) {
		if path != "configs/platform/warehouse.yaml" {
			t.Fatalf("unexpected config path: %s", path)
		}
		return warehouseConfig{DSN: "postgres://warehouse"}, nil
	}
	runWarehouseHealth = func(_ context.Context, cfg warehouseConfig) (warehouseHealthStatus, error) {
		if cfg.DSN != "postgres://warehouse" {
			t.Fatalf("unexpected config: %+v", cfg)
		}
		return warehouseHealthStatus{OK: true, Database: "warehouse", Version: "17.4", PingMS: 12}, nil
	}

	originalStdout := os.Stdout
	readPipe, writePipe, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe stdout: %v", err)
	}
	defer readPipe.Close()
	defer writePipe.Close()
	os.Stdout = writePipe
	defer func() { os.Stdout = originalStdout }()

	if err := runWarehouse([]string{"health", "-config", "configs/platform/warehouse.yaml"}); err != nil {
		t.Fatalf("run warehouse health: %v", err)
	}
	_ = writePipe.Close()
	output, err := io.ReadAll(readPipe)
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	if !strings.Contains(string(output), `"database": "warehouse"`) {
		t.Fatalf("unexpected stdout: %s", output)
	}
}

func TestRunHistoricalSyncUsesDefaultAllowlistWhenSymbolsOmitted(t *testing.T) {
	originalLoad := loadWarehouseConfig
	originalHistorical := runHistoricalSync
	originalWatchlist := loadWatchlistFile
	defer func() {
		loadWarehouseConfig = originalLoad
		runHistoricalSync = originalHistorical
		loadWatchlistFile = originalWatchlist
	}()

	loadWarehouseConfig = func(path string) (warehouseConfig, error) {
		if path != "configs/platform/warehouse.yaml" {
			t.Fatalf("unexpected warehouse config path: %s", path)
		}
		return warehouseConfig{DSN: "postgres://warehouse"}, nil
	}
	loadWatchlistFile = func(path string) (watchlist.File, error) {
		if path != "configs/platform/watchlist.yaml" {
			t.Fatalf("unexpected watchlist path: %s", path)
		}
		return watchlist.File{
			Symbols: []config.LiveSymbolConfig{
				{Symbol: "BTCUSDT"},
				{Symbol: "ETHUSDT"},
			},
			HistoricalIntervals: []string{"15m", "1h", "4h", "1d", "1w"},
		}, nil
	}
	runHistoricalSync = func(_ context.Context, cfg warehouseConfig, request historicalRequest) (historicalSyncResult, error) {
		if cfg.DSN != "postgres://warehouse" {
			t.Fatalf("unexpected config: %+v", cfg)
		}
		if len(request.Symbols) != 2 || request.Symbols[0] != "BTCUSDT" || request.Symbols[1] != "ETHUSDT" {
			t.Fatalf("unexpected default symbols: %+v", request.Symbols)
		}
		if len(request.Intervals) != 1 || request.Intervals[0] != "1h" {
			t.Fatalf("unexpected intervals: %+v", request.Intervals)
		}
		return historicalSyncResult{ArtifactDir: "artifacts/platform/historical-sync/20260329T123000Z"}, nil
	}

	originalStdout := os.Stdout
	readPipe, writePipe, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe stdout: %v", err)
	}
	defer readPipe.Close()
	defer writePipe.Close()
	os.Stdout = writePipe
	defer func() { os.Stdout = originalStdout }()

	if err := runHistorical([]string{"sync", "-warehouse-config", "configs/platform/warehouse.yaml", "-provider", "bitget", "-product-type", "USDT-FUTURES", "-intervals", "1h"}); err != nil {
		t.Fatalf("run historical sync: %v", err)
	}
	_ = writePipe.Close()
	if _, err := io.ReadAll(readPipe); err != nil {
		t.Fatalf("read stdout: %v", err)
	}
}

func TestRunWatchlistApplyUsesWatchlistForHistoricalAndLive(t *testing.T) {
	originalLoad := loadWarehouseConfig
	originalHistorical := runHistoricalSync
	originalAggregate := runAggregateSync
	originalWatchlist := loadWatchlistFile
	originalRuntimeConfig := loadRuntimeConfig
	defer func() {
		loadWarehouseConfig = originalLoad
		runHistoricalSync = originalHistorical
		runAggregateSync = originalAggregate
		loadWatchlistFile = originalWatchlist
		loadRuntimeConfig = originalRuntimeConfig
	}()

	loadWatchlistFile = func(path string) (watchlist.File, error) {
		if path != "configs/platform/watchlist.yaml" {
			t.Fatalf("unexpected watchlist path: %s", path)
		}
		return watchlist.File{
			Provider:            "bitget",
			ProductType:         "USDT-FUTURES",
			HistoricalIntervals: []string{"15m", "1h", "4h", "1d", "1w"},
			HorizonDays:         1095,
			Symbols: []config.LiveSymbolConfig{
				{Symbol: "BTCUSDT", MaxNotional: 500, MaxTranches: 4},
				{Symbol: "ETHUSDT", MaxNotional: 500, MaxTranches: 4},
			},
		}, nil
	}
	loadWarehouseConfig = func(path string) (warehouseConfig, error) {
		if path != "configs/platform/warehouse.yaml" {
			t.Fatalf("unexpected warehouse path: %s", path)
		}
		return warehouseConfig{DSN: "postgres://warehouse"}, nil
	}
	runHistoricalSync = func(_ context.Context, cfg warehouseConfig, request historicalRequest) (historicalSyncResult, error) {
		if cfg.DSN != "postgres://warehouse" {
			t.Fatalf("unexpected cfg: %+v", cfg)
		}
		if len(request.Symbols) != 2 || request.Symbols[0] != "BTCUSDT" || request.Symbols[1] != "ETHUSDT" {
			t.Fatalf("unexpected symbols: %+v", request.Symbols)
		}
		if len(request.Intervals) != 1 || request.Intervals[0] != "15m" {
			t.Fatalf("unexpected intervals: %+v", request.Intervals)
		}
		if request.HorizonDays != 1095 {
			t.Fatalf("unexpected horizon: %d", request.HorizonDays)
		}
		return historicalSyncResult{ArtifactDir: "artifacts/platform/historical-sync/apply"}, nil
	}
	aggregateCalled := false
	runAggregateSync = func(_ context.Context, cfg warehouseConfig, request aggregateRequest) (aggregateResult, error) {
		aggregateCalled = true
		if cfg.DSN != "postgres://warehouse" {
			t.Fatalf("unexpected aggregate cfg: %+v", cfg)
		}
		if len(request.Symbols) != 2 || request.Symbols[0] != "BTCUSDT" || request.Symbols[1] != "ETHUSDT" {
			t.Fatalf("unexpected aggregate symbols: %+v", request.Symbols)
		}
		if len(request.Intervals) != 4 || request.Intervals[0] != "1h" || request.Intervals[3] != "1w" {
			t.Fatalf("unexpected aggregate intervals: %+v", request.Intervals)
		}
		return aggregateResult{Datasets: []aggregateDatasetReport{{Symbol: "BTCUSDT", Interval: "1w", RowCount: 145}}}, nil
	}
	loadRuntimeConfig = func(path string) (config.Config, *strategybundle.Bundle, error) {
		if path != "configs/live.yaml" {
			t.Fatalf("unexpected live config path: %s", path)
		}
		return config.Config{
			Live: config.LiveConfig{
				Exchange: config.ExchangeConfig{
					ProductType: "USDT-FUTURES",
					Symbols: []config.LiveSymbolConfig{
						{Symbol: "BTCUSDT"},
						{Symbol: "ETHUSDT"},
					},
				},
			},
		}, nil, nil
	}

	originalStdout := os.Stdout
	readPipe, writePipe, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe stdout: %v", err)
	}
	defer readPipe.Close()
	defer writePipe.Close()
	os.Stdout = writePipe
	defer func() { os.Stdout = originalStdout }()

	if err := runWatchlist([]string{"apply", "-watchlist", "configs/platform/watchlist.yaml", "-live-config", "configs/live.yaml", "-warehouse-config", "configs/platform/warehouse.yaml"}); err != nil {
		t.Fatalf("run watchlist apply: %v", err)
	}
	_ = writePipe.Close()
	output, err := io.ReadAll(readPipe)
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	if !aggregateCalled {
		t.Fatal("expected watchlist apply to run aggregate sync")
	}
	if !strings.Contains(string(output), `"live_symbols": [`) || !strings.Contains(string(output), `"artifact_dir": "artifacts/platform/historical-sync/apply"`) || !strings.Contains(string(output), `"aggregated":`) {
		t.Fatalf("unexpected stdout: %s", output)
	}
}

func TestRunHistoricalSyncUsesJobRunner(t *testing.T) {
	originalLoad := loadWarehouseConfig
	originalHistorical := runHistoricalSync
	defer func() {
		loadWarehouseConfig = originalLoad
		runHistoricalSync = originalHistorical
	}()

	loadWarehouseConfig = func(path string) (warehouseConfig, error) {
		if path != "configs/platform/warehouse.yaml" {
			t.Fatalf("unexpected warehouse config path: %s", path)
		}
		return warehouseConfig{DSN: "postgres://warehouse"}, nil
	}
	runHistoricalSync = func(_ context.Context, cfg warehouseConfig, request historicalRequest) (historicalSyncResult, error) {
		if cfg.DSN != "postgres://warehouse" {
			t.Fatalf("unexpected config: %+v", cfg)
		}
		if request.Provider != "bitget" || len(request.Symbols) != 1 || request.Symbols[0] != "MSTRUSDT" {
			t.Fatalf("unexpected request: %+v", request)
		}
		return historicalSyncResult{ArtifactDir: "artifacts/platform/historical-sync/20260329T120000Z", Datasets: []historicalDatasetReport{{Symbol: "MSTRUSDT", Interval: "1m", RowCount: 3}}}, nil
	}

	originalStdout := os.Stdout
	readPipe, writePipe, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe stdout: %v", err)
	}
	defer readPipe.Close()
	defer writePipe.Close()
	os.Stdout = writePipe
	defer func() { os.Stdout = originalStdout }()

	if err := runHistorical([]string{"sync", "-warehouse-config", "configs/platform/warehouse.yaml", "-provider", "bitget", "-symbols", "MSTRUSDT", "-product-type", "USDT-FUTURES"}); err != nil {
		t.Fatalf("run historical sync: %v", err)
	}
	_ = writePipe.Close()
	output, err := io.ReadAll(readPipe)
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	if !strings.Contains(string(output), `"artifact_dir": "artifacts/platform/historical-sync/20260329T120000Z"`) {
		t.Fatalf("unexpected stdout: %s", output)
	}
}

func TestRunAggregateUsesJobRunner(t *testing.T) {
	originalLoad := loadWarehouseConfig
	originalAggregate := runAggregateSync
	defer func() {
		loadWarehouseConfig = originalLoad
		runAggregateSync = originalAggregate
	}()

	loadWarehouseConfig = func(path string) (warehouseConfig, error) {
		if path != "configs/platform/warehouse.yaml" {
			t.Fatalf("unexpected warehouse config path: %s", path)
		}
		return warehouseConfig{DSN: "postgres://warehouse"}, nil
	}
	runAggregateSync = func(_ context.Context, cfg warehouseConfig, request aggregateRequest) (aggregateResult, error) {
		if cfg.DSN != "postgres://warehouse" {
			t.Fatalf("unexpected config: %+v", cfg)
		}
		if request.Provider != "bitget" || request.ProductType != "USDT-FUTURES" {
			t.Fatalf("unexpected request metadata: %+v", request)
		}
		if len(request.Symbols) != 2 || request.Symbols[0] != "BTCUSDT" || request.Symbols[1] != "ETHUSDT" {
			t.Fatalf("unexpected symbols: %+v", request.Symbols)
		}
		if len(request.Intervals) != 2 || request.Intervals[0] != "5m" || request.Intervals[1] != "4h" {
			t.Fatalf("unexpected intervals: %+v", request.Intervals)
		}
		return aggregateResult{Datasets: []aggregateDatasetReport{{Symbol: "BTCUSDT", Interval: "5m", RowCount: 12}}}, nil
	}

	originalStdout := os.Stdout
	readPipe, writePipe, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe stdout: %v", err)
	}
	defer readPipe.Close()
	defer writePipe.Close()
	os.Stdout = writePipe
	defer func() { os.Stdout = originalStdout }()

	if err := runAggregate([]string{"run", "-warehouse-config", "configs/platform/warehouse.yaml", "-provider", "bitget", "-product-type", "USDT-FUTURES", "-symbols", "BTCUSDT,ETHUSDT", "-intervals", "5m,4h"}); err != nil {
		t.Fatalf("run aggregate: %v", err)
	}
	_ = writePipe.Close()
	output, err := io.ReadAll(readPipe)
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	if !strings.Contains(string(output), `"interval": "5m"`) {
		t.Fatalf("unexpected stdout: %s", output)
	}
}

func TestRunExportParquetUsesJobRunnerAndTimestampedArtifactRoot(t *testing.T) {
	originalLoad := loadWarehouseConfig
	originalExport := runParquetExport
	originalNow := platformctlNow
	defer func() {
		loadWarehouseConfig = originalLoad
		runParquetExport = originalExport
		platformctlNow = originalNow
	}()

	artifactRoot := t.TempDir()
	expectedRoot := filepath.Join(artifactRoot, "20260329T130000Z")
	loadWarehouseConfig = func(path string) (warehouseConfig, error) {
		if path != "configs/platform/warehouse.yaml" {
			t.Fatalf("unexpected warehouse config path: %s", path)
		}
		return warehouseConfig{DSN: "postgres://warehouse"}, nil
	}
	platformctlNow = func() time.Time {
		return time.Date(2026, 3, 29, 13, 0, 0, 0, time.UTC)
	}
	runParquetExport = func(_ context.Context, cfg warehouseConfig, request exportRequest) (exportResult, error) {
		if cfg.DSN != "postgres://warehouse" {
			t.Fatalf("unexpected config: %+v", cfg)
		}
		if request.Provider != "bitget" {
			t.Fatalf("unexpected provider: %+v", request)
		}
		if len(request.Symbols) != 2 || request.Symbols[0] != "BTCUSDT" || request.Symbols[1] != "ETHUSDT" {
			t.Fatalf("unexpected symbols: %+v", request.Symbols)
		}
		if len(request.Intervals) != 2 || request.Intervals[0] != "1h" || request.Intervals[1] != "4h" {
			t.Fatalf("unexpected intervals: %+v", request.Intervals)
		}
		if request.RootDir != expectedRoot {
			t.Fatalf("unexpected root dir: %s", request.RootDir)
		}
		return exportResult{RootDir: expectedRoot, SummaryPath: filepath.Join(expectedRoot, "summary.json"), TotalRows: 96}, nil
	}

	originalStdout := os.Stdout
	readPipe, writePipe, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe stdout: %v", err)
	}
	defer readPipe.Close()
	defer writePipe.Close()
	os.Stdout = writePipe
	defer func() { os.Stdout = originalStdout }()

	if err := runExport([]string{"parquet", "-warehouse-config", "configs/platform/warehouse.yaml", "-provider", "bitget", "-symbols", "BTCUSDT,ETHUSDT", "-intervals", "1h,4h", "-artifact-root", artifactRoot}); err != nil {
		t.Fatalf("run export parquet: %v", err)
	}
	_ = writePipe.Close()
	output, err := io.ReadAll(readPipe)
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	if !strings.Contains(string(output), `"total_rows": 96`) || !strings.Contains(string(output), expectedRoot) {
		t.Fatalf("unexpected stdout: %s", output)
	}
	body, err := os.ReadFile(filepath.Join(artifactRoot, "summary.json"))
	if err != nil {
		t.Fatalf("read latest export summary: %v", err)
	}
	var written exportResult
	if err := json.Unmarshal(body, &written); err != nil {
		t.Fatalf("unmarshal latest export summary: %v", err)
	}
	if written.LatestRootDir != expectedRoot || written.SummaryPath != filepath.Join(expectedRoot, "summary.json") || written.TotalRows != 96 {
		t.Fatalf("unexpected latest export summary: %+v", written)
	}
}

func TestRunResearchRunUsesJobRunnerAndPrintsResponse(t *testing.T) {
	originalResearch := runResearchJob
	defer func() {
		runResearchJob = originalResearch
	}()

	var gotRequest researchRequest
	runResearchJob = func(_ context.Context, request researchRequest) (researchResult, error) {
		gotRequest = request
		return researchResult{
			RequestID:      "resp-1",
			Status:         "completed",
			ArtifactDir:    "artifacts/platform/research/20260329T150000Z-nightly-report-mstr-wave-fib",
			OutputArtifact: "artifacts/platform/research/20260329T150000Z-nightly-report-mstr-wave-fib/output.txt",
		}, nil
	}

	originalStdout := os.Stdout
	readPipe, writePipe, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe stdout: %v", err)
	}
	defer readPipe.Close()
	defer writePipe.Close()
	os.Stdout = writePipe
	defer func() { os.Stdout = originalStdout }()

	if err := runResearch([]string{"run", "-kind", "nightly_report", "-strategy", "mstr-wave-fib", "-datasets", "BTCUSDT:1h,ETHUSDT:4h", "-artifact-root", "artifacts/platform/research", "-poll-interval", "2s", "-timeout", "45s"}); err != nil {
		t.Fatalf("run research: %v", err)
	}
	_ = writePipe.Close()
	output, err := io.ReadAll(readPipe)
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	if gotRequest.Kind != "nightly_report" || gotRequest.StrategyID != "mstr-wave-fib" {
		t.Fatalf("unexpected request: %+v", gotRequest)
	}
	if len(gotRequest.Datasets) != 2 || gotRequest.Datasets[0] != "BTCUSDT:1h" || gotRequest.Datasets[1] != "ETHUSDT:4h" {
		t.Fatalf("unexpected datasets: %+v", gotRequest.Datasets)
	}
	if gotRequest.ArtifactRoot != "artifacts/platform/research" || gotRequest.PollInterval != 2*time.Second || gotRequest.Timeout != 45*time.Second {
		t.Fatalf("unexpected request timing: %+v", gotRequest)
	}
	if !strings.Contains(string(output), `"request_id": "resp-1"`) {
		t.Fatalf("unexpected stdout: %s", output)
	}
}
