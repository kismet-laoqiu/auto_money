package api

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"quantlab/internal/backtest"
	"quantlab/internal/platform/insights"
	"quantlab/internal/platform/live"
	"quantlab/internal/platform/promotion"
	"quantlab/internal/platform/query"
	watchlistsvc "quantlab/internal/platform/watchlist"
	basewatchlist "quantlab/internal/watchlist"
	sqlitepkg "quantlab/internal/store/sqlite"
	"quantlab/internal/strategybundle"
	"quantlab/internal/trader"
)

type Reader interface {
	LastSeq(context.Context) (int64, error)
	ListConsumerCursors(context.Context) ([]sqlitepkg.ConsumerCursorRow, error)
	ListLatestPositions(context.Context) ([]sqlitepkg.PositionRow, error)
	ListLatestOrders(context.Context, int) ([]sqlitepkg.OrderRow, error)
	ListRecentEvents(context.Context, int) ([]sqlitepkg.EventEnvelope, error)
	LoadCheckpoint(context.Context, string) (trader.EngineState, error)
}

type HandlerConfig struct {
	Store      Reader
	Backtests  BacktestRunner
	Promotions PromotionManager
	Strategies StrategyRegistry
	Query      QueryService
	Live       LiveOps
	Dashboard  DashboardService
	Watchlist  WatchlistService
}

type BacktestRunner interface {
	RunConfig(ctx context.Context, configPath string, refresh bool) (backtest.Result, error)
}

type PromotionManager interface {
	CreateRequest(ctx context.Context, input promotion.CreateInput) (promotion.Request, error)
	Get(ctx context.Context, id string) (promotion.Request, error)
	List(ctx context.Context) ([]promotion.Request, error)
	StartShadow(ctx context.Context, id string) (promotion.Request, error)
	PassShadow(ctx context.Context, id string) (promotion.Request, error)
	StartCanary(ctx context.Context, id string) (promotion.Request, error)
	DegradeCanary(ctx context.Context, id string, reason string) (promotion.Request, error)
	Approve(ctx context.Context, id string) (promotion.Request, error)
	Rollback(ctx context.Context, id string, reason string) (promotion.Request, error)
}

type QueryService interface {
	BarsFromConfig(ctx context.Context, configPath, datasetName string, refresh bool) (query.BarsResult, error)
	FeaturesFromConfig(ctx context.Context, configPath, datasetName string, refresh bool, offset int) (query.FeaturesResult, error)
}

type LiveOps interface {
	FlattenSymbol(ctx context.Context, symbol string) (live.FlattenResult, error)
}

type DashboardService interface {
	Report(ctx context.Context) (insights.DashboardReport, error)
	RenderHTML(ctx context.Context, symbolsText string, flash string) (string, error)
}

type WatchlistService interface {
	SymbolsText() (string, error)
	SaveSymbols(raw string) (basewatchlist.File, error)
	Apply(ctx context.Context) (watchlistsvc.ApplyResult, error)
}

type StrategyRegistry interface {
	List(strategyID string) ([]strategybundle.VersionInfo, error)
}

type StrategyVersion struct {
	StrategyID     string            `json:"strategy_id"`
	Version        string            `json:"version"`
	RootPath       string            `json:"root_path"`
	ConfigPath     string            `json:"config_path"`
	ObjectiveScore float64           `json:"objective_score"`
	FinalScore     float64           `json:"final_score"`
	UpdatedAt      string            `json:"updated_at"`
	Promotion      StrategyPromotion `json:"promotion"`
}

type StrategyPromotion struct {
	ID    string          `json:"id"`
	State promotion.State `json:"state"`
}

func NewHandler(cfg HandlerConfig) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		handleDashboardPage(writer, request, cfg.Dashboard, cfg.Watchlist)
	})
	mux.HandleFunc("/health", func(writer http.ResponseWriter, request *http.Request) {
		writeJSON(writer, http.StatusOK, map[string]bool{"ok": true})
	})
	mux.HandleFunc("/api/status", func(writer http.ResponseWriter, request *http.Request) {
		handleStatus(writer, request, cfg.Store)
	})
	mux.HandleFunc("/api/positions", func(writer http.ResponseWriter, request *http.Request) {
		handlePositions(writer, request, cfg.Store)
	})
	mux.HandleFunc("/api/orders", func(writer http.ResponseWriter, request *http.Request) {
		handleOrders(writer, request, cfg.Store)
	})
	mux.HandleFunc("/api/events", func(writer http.ResponseWriter, request *http.Request) {
		handleEvents(writer, request, cfg.Store)
	})
	mux.HandleFunc("/api/backtests/run", func(writer http.ResponseWriter, request *http.Request) {
		handleBacktestRun(writer, request, cfg.Backtests)
	})
	mux.HandleFunc("/api/bars", func(writer http.ResponseWriter, request *http.Request) {
		handleBars(writer, request, cfg.Query)
	})
	mux.HandleFunc("/api/features", func(writer http.ResponseWriter, request *http.Request) {
		handleFeatures(writer, request, cfg.Query)
	})
	mux.HandleFunc("/api/strategies/versions", func(writer http.ResponseWriter, request *http.Request) {
		handleStrategyVersions(writer, request, cfg.Promotions, cfg.Strategies)
	})
	mux.HandleFunc("/api/promotions", func(writer http.ResponseWriter, request *http.Request) {
		handlePromotions(writer, request, cfg.Promotions)
	})
	mux.HandleFunc("/api/promotions/request", func(writer http.ResponseWriter, request *http.Request) {
		handlePromotionRequest(writer, request, cfg.Promotions)
	})
	mux.HandleFunc("/api/promotions/shadow", func(writer http.ResponseWriter, request *http.Request) {
		handlePromotionAction(writer, request, cfg.Promotions, promotion.ActionStartShadow)
	})
	mux.HandleFunc("/api/promotions/shadow/pass", func(writer http.ResponseWriter, request *http.Request) {
		handlePromotionAction(writer, request, cfg.Promotions, promotion.ActionPassShadow)
	})
	mux.HandleFunc("/api/promotions/canary", func(writer http.ResponseWriter, request *http.Request) {
		handlePromotionAction(writer, request, cfg.Promotions, promotion.ActionStartCanary)
	})
	mux.HandleFunc("/api/promotions/canary/degrade", func(writer http.ResponseWriter, request *http.Request) {
		handlePromotionAction(writer, request, cfg.Promotions, promotion.ActionDegradeCanary)
	})
	mux.HandleFunc("/api/promotions/approve", func(writer http.ResponseWriter, request *http.Request) {
		handlePromotionAction(writer, request, cfg.Promotions, promotion.ActionApprove)
	})
	mux.HandleFunc("/api/promotions/rollback", func(writer http.ResponseWriter, request *http.Request) {
		handlePromotionAction(writer, request, cfg.Promotions, promotion.ActionRollback)
	})
	mux.HandleFunc("/api/live/flatten", func(writer http.ResponseWriter, request *http.Request) {
		handleLiveFlatten(writer, request, cfg.Live)
	})
	mux.HandleFunc("/api/dashboard", func(writer http.ResponseWriter, request *http.Request) {
		handleDashboardJSON(writer, request, cfg.Dashboard)
	})
	mux.HandleFunc("/api/watchlist/save", func(writer http.ResponseWriter, request *http.Request) {
		handleWatchlistSave(writer, request, cfg.Dashboard, cfg.Watchlist)
	})
	mux.HandleFunc("/api/watchlist/apply", func(writer http.ResponseWriter, request *http.Request) {
		handleWatchlistApply(writer, request, cfg.Dashboard, cfg.Watchlist)
	})
	return mux
}

func handleStatus(writer http.ResponseWriter, request *http.Request, store Reader) {
	if store == nil {
		writeError(writer, http.StatusInternalServerError, "store is nil")
		return
	}
	ctx := request.Context()
	lastSeq, err := store.LastSeq(ctx)
	if err != nil {
		writeError(writer, http.StatusInternalServerError, err.Error())
		return
	}
	cursors, err := store.ListConsumerCursors(ctx)
	if err != nil {
		writeError(writer, http.StatusInternalServerError, err.Error())
		return
	}
	checkpoint, err := store.LoadCheckpoint(ctx, "trader.runtime")
	if err != nil {
		writeError(writer, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(writer, http.StatusOK, map[string]any{
		"last_seq": lastSeq,
		"cursors":  cursors,
		"trader": map[string]any{
			"arming_state": checkpoint.ArmingState,
			"symbols":      checkpoint.Symbols,
		},
	})
}

func handlePositions(writer http.ResponseWriter, request *http.Request, store Reader) {
	if store == nil {
		writeError(writer, http.StatusInternalServerError, "store is nil")
		return
	}
	positions, err := store.ListLatestPositions(request.Context())
	if err != nil {
		writeError(writer, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(writer, http.StatusOK, positions)
}

func handleOrders(writer http.ResponseWriter, request *http.Request, store Reader) {
	if store == nil {
		writeError(writer, http.StatusInternalServerError, "store is nil")
		return
	}
	limit := parseLimit(request, 20)
	orders, err := store.ListLatestOrders(request.Context(), limit)
	if err != nil {
		writeError(writer, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(writer, http.StatusOK, orders)
}

func handleEvents(writer http.ResponseWriter, request *http.Request, store Reader) {
	if store == nil {
		writeError(writer, http.StatusInternalServerError, "store is nil")
		return
	}
	events, err := store.ListRecentEvents(request.Context(), parseLimit(request, 50))
	if err != nil {
		writeError(writer, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(writer, http.StatusOK, events)
}

func handleBacktestRun(writer http.ResponseWriter, request *http.Request, runner BacktestRunner) {
	if request.Method != http.MethodPost {
		writeError(writer, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if runner == nil {
		writeError(writer, http.StatusInternalServerError, "backtest runner is nil")
		return
	}
	var body struct {
		ConfigPath string `json:"config_path"`
		Refresh    bool   `json:"refresh"`
	}
	if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
		writeError(writer, http.StatusBadRequest, err.Error())
		return
	}
	if body.ConfigPath == "" {
		writeError(writer, http.StatusBadRequest, "config_path is required")
		return
	}
	result, err := runner.RunConfig(request.Context(), body.ConfigPath, body.Refresh)
	if err != nil {
		writeError(writer, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(writer, http.StatusOK, result)
}

func handleBars(writer http.ResponseWriter, request *http.Request, service QueryService) {
	if request.Method != http.MethodGet {
		writeError(writer, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if service == nil {
		writeError(writer, http.StatusInternalServerError, "query service is nil")
		return
	}
	configPath := request.URL.Query().Get("config_path")
	if configPath == "" {
		writeError(writer, http.StatusBadRequest, "config_path is required")
		return
	}
	result, err := service.BarsFromConfig(request.Context(), configPath, request.URL.Query().Get("dataset"), request.URL.Query().Get("refresh") == "true")
	if err != nil {
		writeError(writer, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(writer, http.StatusOK, result)
}

func handleDashboardPage(writer http.ResponseWriter, request *http.Request, dashboard DashboardService, watchlist WatchlistService) {
	if request.Method != http.MethodGet {
		writeError(writer, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if dashboard == nil || watchlist == nil {
		writeError(writer, http.StatusInternalServerError, "dashboard or watchlist service is nil")
		return
	}
	symbolsText, err := watchlist.SymbolsText()
	if err != nil {
		writeError(writer, http.StatusInternalServerError, err.Error())
		return
	}
	body, err := dashboard.RenderHTML(request.Context(), symbolsText, request.URL.Query().Get("flash"))
	if err != nil {
		writeError(writer, http.StatusInternalServerError, err.Error())
		return
	}
	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = writer.Write([]byte(body))
}

func handleDashboardJSON(writer http.ResponseWriter, request *http.Request, dashboard DashboardService) {
	if request.Method != http.MethodGet {
		writeError(writer, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if dashboard == nil {
		writeError(writer, http.StatusInternalServerError, "dashboard service is nil")
		return
	}
	report, err := dashboard.Report(request.Context())
	if err != nil {
		writeError(writer, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(writer, http.StatusOK, report)
}

func handleWatchlistSave(writer http.ResponseWriter, request *http.Request, dashboard DashboardService, watchlist WatchlistService) {
	if request.Method != http.MethodPost {
		writeError(writer, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if watchlist == nil {
		writeError(writer, http.StatusInternalServerError, "watchlist service is nil")
		return
	}
	raw, err := readSymbolsPayload(request)
	if err != nil {
		writeError(writer, http.StatusBadRequest, err.Error())
		return
	}
	file, err := watchlist.SaveSymbols(raw)
	if err != nil {
		writeError(writer, http.StatusBadRequest, err.Error())
		return
	}
	if wantsHTML(request) && dashboard != nil {
		symbolsText, err := watchlist.SymbolsText()
		if err != nil {
			writeError(writer, http.StatusInternalServerError, err.Error())
			return
		}
		body, err := dashboard.RenderHTML(request.Context(), symbolsText, "watchlist saved")
		if err != nil {
			writeError(writer, http.StatusInternalServerError, err.Error())
			return
		}
		writer.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = writer.Write([]byte(body))
		return
	}
	writeJSON(writer, http.StatusOK, map[string]any{"symbols": file.SymbolNames()})
}

func handleWatchlistApply(writer http.ResponseWriter, request *http.Request, dashboard DashboardService, watchlist WatchlistService) {
	if request.Method != http.MethodPost {
		writeError(writer, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if watchlist == nil {
		writeError(writer, http.StatusInternalServerError, "watchlist service is nil")
		return
	}
	result, err := watchlist.Apply(request.Context())
	if err != nil {
		writeError(writer, http.StatusInternalServerError, err.Error())
		return
	}
	if wantsHTML(request) && dashboard != nil {
		symbolsText, symbolErr := watchlist.SymbolsText()
		if symbolErr != nil {
			writeError(writer, http.StatusInternalServerError, symbolErr.Error())
			return
		}
		body, renderErr := dashboard.RenderHTML(request.Context(), symbolsText, "watchlist apply completed")
		if renderErr != nil {
			writeError(writer, http.StatusInternalServerError, renderErr.Error())
			return
		}
		writer.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = writer.Write([]byte(body))
		return
	}
	writeJSON(writer, http.StatusOK, result)
}

func readSymbolsPayload(request *http.Request) (string, error) {
	if strings.Contains(request.Header.Get("Content-Type"), "application/json") {
		var body struct {
			Symbols string `json:"symbols"`
		}
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			return "", err
		}
		if strings.TrimSpace(body.Symbols) == "" {
			return "", http.ErrMissingFile
		}
		return body.Symbols, nil
	}
	if err := request.ParseForm(); err != nil {
		return "", err
	}
	value := request.Form.Get("symbols")
	if strings.TrimSpace(value) == "" {
		return "", http.ErrMissingFile
	}
	return value, nil
}

func wantsHTML(request *http.Request) bool {
	if strings.Contains(request.Header.Get("Accept"), "text/html") {
		return true
	}
	return strings.Contains(request.Header.Get("Content-Type"), "application/x-www-form-urlencoded")
}

func handleFeatures(writer http.ResponseWriter, request *http.Request, service QueryService) {
	if request.Method != http.MethodGet {
		writeError(writer, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if service == nil {
		writeError(writer, http.StatusInternalServerError, "query service is nil")
		return
	}
	configPath := request.URL.Query().Get("config_path")
	if configPath == "" {
		writeError(writer, http.StatusBadRequest, "config_path is required")
		return
	}
	result, err := service.FeaturesFromConfig(request.Context(), configPath, request.URL.Query().Get("dataset"), request.URL.Query().Get("refresh") == "true", parseLimitParam(request, "offset", 0))
	if err != nil {
		writeError(writer, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(writer, http.StatusOK, result)
}

func handleStrategyVersions(writer http.ResponseWriter, request *http.Request, manager PromotionManager, registry StrategyRegistry) {
	if request.Method != http.MethodGet {
		writeError(writer, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if manager == nil && registry == nil {
		writeError(writer, http.StatusInternalServerError, "strategy registry and promotion manager are nil")
		return
	}
	strategyID := request.URL.Query().Get("strategy_id")
	requests := []promotion.Request(nil)
	if manager != nil {
		var err error
		requests, err = manager.List(request.Context())
		if err != nil {
			writeError(writer, http.StatusInternalServerError, err.Error())
			return
		}
	}
	registered := []strategybundle.VersionInfo(nil)
	if registry != nil {
		var err error
		registered, err = registry.List(strategyID)
		if err != nil {
			writeError(writer, http.StatusInternalServerError, err.Error())
			return
		}
	}
	writeJSON(writer, http.StatusOK, buildStrategyVersions(registered, requests, strategyID))
}

func handlePromotions(writer http.ResponseWriter, request *http.Request, manager PromotionManager) {
	if request.Method != http.MethodGet {
		writeError(writer, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if manager == nil {
		writeError(writer, http.StatusInternalServerError, "promotion manager is nil")
		return
	}
	id := request.URL.Query().Get("id")
	if id != "" {
		result, err := manager.Get(request.Context(), id)
		if err != nil {
			writeError(writer, http.StatusNotFound, err.Error())
			return
		}
		writeJSON(writer, http.StatusOK, result)
		return
	}
	result, err := manager.List(request.Context())
	if err != nil {
		writeError(writer, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(writer, http.StatusOK, result)
}

func handlePromotionRequest(writer http.ResponseWriter, request *http.Request, manager PromotionManager) {
	if request.Method != http.MethodPost {
		writeError(writer, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if manager == nil {
		writeError(writer, http.StatusInternalServerError, "promotion manager is nil")
		return
	}
	var body struct {
		StrategyID string `json:"strategy_id"`
		Version    string `json:"version"`
		ConfigPath string `json:"config_path"`
		Refresh    bool   `json:"refresh"`
	}
	if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
		writeError(writer, http.StatusBadRequest, err.Error())
		return
	}
	result, err := manager.CreateRequest(request.Context(), promotion.CreateInput{
		StrategyID: body.StrategyID,
		Version:    body.Version,
		ConfigPath: body.ConfigPath,
		Refresh:    body.Refresh,
	})
	if err != nil {
		writeError(writer, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(writer, http.StatusOK, result)
}

func handlePromotionAction(writer http.ResponseWriter, request *http.Request, manager PromotionManager, action promotion.Action) {
	if request.Method != http.MethodPost {
		writeError(writer, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if manager == nil {
		writeError(writer, http.StatusInternalServerError, "promotion manager is nil")
		return
	}
	var body struct {
		ID     string `json:"id"`
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
		writeError(writer, http.StatusBadRequest, err.Error())
		return
	}
	if body.ID == "" {
		writeError(writer, http.StatusBadRequest, "id is required")
		return
	}
	var (
		result promotion.Request
		err    error
	)
	switch action {
	case promotion.ActionStartShadow:
		result, err = manager.StartShadow(request.Context(), body.ID)
	case promotion.ActionPassShadow:
		result, err = manager.PassShadow(request.Context(), body.ID)
	case promotion.ActionStartCanary:
		result, err = manager.StartCanary(request.Context(), body.ID)
	case promotion.ActionDegradeCanary:
		result, err = manager.DegradeCanary(request.Context(), body.ID, body.Reason)
	case promotion.ActionApprove:
		result, err = manager.Approve(request.Context(), body.ID)
	case promotion.ActionRollback:
		result, err = manager.Rollback(request.Context(), body.ID, body.Reason)
	default:
		writeError(writer, http.StatusInternalServerError, "unsupported promotion action")
		return
	}
	if err != nil {
		writeError(writer, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(writer, http.StatusOK, result)
}

func handleLiveFlatten(writer http.ResponseWriter, request *http.Request, ops LiveOps) {
	if request.Method != http.MethodPost {
		writeError(writer, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if ops == nil {
		writeError(writer, http.StatusInternalServerError, "live ops is nil")
		return
	}
	var body struct {
		Symbol string `json:"symbol"`
	}
	if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
		writeError(writer, http.StatusBadRequest, err.Error())
		return
	}
	if body.Symbol == "" {
		writeError(writer, http.StatusBadRequest, "symbol is required")
		return
	}
	result, err := ops.FlattenSymbol(request.Context(), body.Symbol)
	if err != nil {
		writeError(writer, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(writer, http.StatusOK, result)
}

func buildStrategyVersions(registered []strategybundle.VersionInfo, requests []promotion.Request, strategyID string) []StrategyVersion {
	versions := make(map[string]StrategyVersion, len(registered)+len(requests))
	for _, registeredVersion := range registered {
		if strategyID != "" && registeredVersion.StrategyID != strategyID {
			continue
		}
		updatedAt := ""
		if !registeredVersion.UpdatedAt.IsZero() {
			updatedAt = registeredVersion.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z07:00")
		}
		key := registeredVersion.StrategyID + ":" + registeredVersion.Version
		versions[key] = StrategyVersion{
			StrategyID: registeredVersion.StrategyID,
			Version:    registeredVersion.Version,
			RootPath:   registeredVersion.RootPath,
			UpdatedAt:  updatedAt,
		}
	}
	for _, request := range requests {
		if strategyID != "" && request.StrategyID != strategyID {
			continue
		}
		key := request.StrategyID + ":" + request.Version
		current := versions[key]
		current.StrategyID = request.StrategyID
		current.Version = request.Version
		if request.ConfigPath != "" {
			current.ConfigPath = request.ConfigPath
		}
		current.ObjectiveScore = request.ObjectiveScore
		current.FinalScore = request.FinalScore
		if !request.UpdatedAt.IsZero() {
			requestUpdatedAt := request.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z07:00")
			if requestUpdatedAt > current.UpdatedAt {
				current.UpdatedAt = requestUpdatedAt
			}
		}
		current.Promotion = StrategyPromotion{
			ID:    request.ID,
			State: request.State,
		}
		versions[key] = current
	}
	out := make([]StrategyVersion, 0, len(versions))
	for _, version := range versions {
		out = append(out, version)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].UpdatedAt == out[j].UpdatedAt {
			if out[i].StrategyID == out[j].StrategyID {
				return strategybundle.CompareVersions(out[i].Version, out[j].Version) > 0
			}
			return out[i].StrategyID < out[j].StrategyID
		}
		return out[i].UpdatedAt > out[j].UpdatedAt
	})
	return out
}

func parseLimit(request *http.Request, defaultValue int) int {
	return parseLimitParam(request, "limit", defaultValue)
}

func parseLimitParam(request *http.Request, key string, defaultValue int) int {
	value := request.URL.Query().Get(key)
	if value == "" {
		return defaultValue
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 0 {
		return defaultValue
	}
	return parsed
}

func writeError(writer http.ResponseWriter, status int, message string) {
	writeJSON(writer, status, map[string]string{"error": message})
}

func writeJSON(writer http.ResponseWriter, status int, payload any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(payload)
}
