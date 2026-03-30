package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"quantlab/internal/backtest"
	"quantlab/internal/market"
	"quantlab/internal/platform/live"
	"quantlab/internal/platform/promotion"
	"quantlab/internal/platform/query"
	sqlitepkg "quantlab/internal/store/sqlite"
	"quantlab/internal/strategybundle"
	"quantlab/internal/trader"
)

func TestRouterHealthEndpoint(t *testing.T) {
	store := newTestStore(t)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/health", nil)

	NewHandler(HandlerConfig{Store: store}).ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status code: %d", recorder.Code)
	}
	var payload struct {
		OK bool `json:"ok"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode health payload: %v", err)
	}
	if !payload.OK {
		t.Fatalf("unexpected health payload: %+v", payload)
	}
}

func TestRouterStatusIncludesCheckpointAndCursors(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	if _, err := store.AppendEvent(ctx, "market.public", market.BarClosedEvent{
		EventIDValue: "bar-1",
		SymbolValue:  "MSTRUSDT",
		Interval:     "1m",
		Ts:           time.Unix(1710000000, 0).UTC(),
	}, []byte(`{}`)); err != nil {
		t.Fatalf("append event: %v", err)
	}
	if err := store.SaveConsumerCursor(ctx, "traderd", 1); err != nil {
		t.Fatalf("save cursor: %v", err)
	}
	if err := store.SaveCheckpoint(ctx, "trader.runtime", trader.EngineState{ArmingState: trader.ArmingSafe}); err != nil {
		t.Fatalf("save checkpoint: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	NewHandler(HandlerConfig{Store: store}).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status code: %d", recorder.Code)
	}
	var payload struct {
		LastSeq int64 `json:"last_seq"`
		Cursors []struct {
			ConsumerKey string `json:"consumer_key"`
			LastSeq     int64  `json:"last_seq"`
		} `json:"cursors"`
		Trader struct {
			ArmingState string `json:"arming_state"`
		} `json:"trader"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode status payload: %v", err)
	}
	if payload.LastSeq != 1 {
		t.Fatalf("unexpected last seq: %+v", payload)
	}
	if len(payload.Cursors) != 1 || payload.Cursors[0].ConsumerKey != "traderd" || payload.Cursors[0].LastSeq != 1 {
		t.Fatalf("unexpected cursor payload: %+v", payload.Cursors)
	}
	if payload.Trader.ArmingState != string(trader.ArmingSafe) {
		t.Fatalf("unexpected trader state: %+v", payload.Trader)
	}
}

func TestRouterPositionsReturnsLatestPerSymbol(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	appendPositionEvent(t, ctx, store, market.PositionEvent{
		EventIDValue: "pos-1",
		SymbolValue:  "MSTRUSDT",
		Ts:           time.Unix(1710000000, 0).UTC(),
		Qty:          0.01,
		KindValue:    "position_snapshot",
	})
	appendPositionEvent(t, ctx, store, market.PositionEvent{
		EventIDValue: "pos-2",
		SymbolValue:  "MSTRUSDT",
		Ts:           time.Unix(1710000100, 0).UTC(),
		Qty:          0.04,
		KindValue:    "position_update",
	})
	appendPositionEvent(t, ctx, store, market.PositionEvent{
		EventIDValue: "pos-3",
		SymbolValue:  "ETHUSDT",
		Ts:           time.Unix(1710000200, 0).UTC(),
		Qty:          0,
		KindValue:    "position_snapshot",
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/positions", nil)
	NewHandler(HandlerConfig{Store: store}).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status code: %d", recorder.Code)
	}
	var payload []struct {
		Symbol string  `json:"symbol"`
		Qty    float64 `json:"qty"`
		Kind   string  `json:"kind"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode position payload: %v", err)
	}
	if len(payload) != 2 || payload[0].Symbol != "ETHUSDT" || payload[1].Symbol != "MSTRUSDT" || payload[1].Qty != 0.04 || payload[1].Kind != "position_update" {
		t.Fatalf("unexpected position payload: %+v", payload)
	}
}

func TestRouterOrdersHonorsLimit(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	appendOrderEvent(t, ctx, store, market.OrderEvent{
		EventIDValue: "order-1",
		SymbolValue:  "MSTRUSDT",
		Ts:           time.Unix(1710000000, 0).UTC(),
		OrderID:      "1001",
		KindValue:    "order_update",
	})
	appendOrderEvent(t, ctx, store, market.OrderEvent{
		EventIDValue: "order-2",
		SymbolValue:  "MSTRUSDT",
		Ts:           time.Unix(1710000100, 0).UTC(),
		OrderID:      "1002",
		KindValue:    "order_fill",
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/orders?limit=1", nil)
	NewHandler(HandlerConfig{Store: store}).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status code: %d", recorder.Code)
	}
	var payload []struct {
		OrderID string `json:"order_id"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode order payload: %v", err)
	}
	if len(payload) != 1 || payload[0].OrderID != "1002" {
		t.Fatalf("unexpected order payload: %+v", payload)
	}
}

func TestRouterBacktestRunEndpoint(t *testing.T) {
	store := newTestStore(t)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/backtests/run", strings.NewReader(`{"config_path":"configs/baseline.yaml","refresh":true}`))
	request.Header.Set("Content-Type", "application/json")

	NewHandler(HandlerConfig{
		Store: store,
		Backtests: stubBacktestRunner{
			result: backtest.Result{
				GeneratedAt:    time.Unix(1710000000, 0).UTC(),
				ObjectiveScore: 0.86,
				FinalScore:     0.91,
			},
		},
	}).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status code: %d body=%s", recorder.Code, recorder.Body.String())
	}
	var payload struct {
		FinalScore float64 `json:"final_score"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode backtest payload: %v", err)
	}
	if payload.FinalScore != 0.91 {
		t.Fatalf("unexpected backtest payload: %+v", payload)
	}
}

func TestRouterBarsEndpoint(t *testing.T) {
	store := newTestStore(t)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/bars?config_path=configs/live.yaml&dataset=mstr_demo", nil)

	NewHandler(HandlerConfig{
		Store: store,
		Query: stubQueryService{
			bars: query.BarsResult{Name: "mstr_demo", Symbol: "MSTRUSDT", Interval: "1m"},
		},
	}).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status code: %d body=%s", recorder.Code, recorder.Body.String())
	}
	var payload struct {
		Name   string `json:"name"`
		Symbol string `json:"symbol"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode bars payload: %v", err)
	}
	if payload.Name != "mstr_demo" || payload.Symbol != "MSTRUSDT" {
		t.Fatalf("unexpected bars payload: %+v", payload)
	}
}

func TestRouterFeaturesEndpoint(t *testing.T) {
	store := newTestStore(t)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/features?config_path=configs/live.yaml&dataset=mstr_demo&offset=0", nil)

	NewHandler(HandlerConfig{
		Store: store,
		Query: stubQueryService{
			features: query.FeaturesResult{Symbol: "MSTRUSDT", Index: 12},
		},
	}).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status code: %d body=%s", recorder.Code, recorder.Body.String())
	}
	var payload struct {
		Symbol string `json:"symbol"`
		Index  int    `json:"index"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode features payload: %v", err)
	}
	if payload.Symbol != "MSTRUSDT" || payload.Index != 12 {
		t.Fatalf("unexpected features payload: %+v", payload)
	}
}

func TestRouterPromotionRequestEndpoint(t *testing.T) {
	store := newTestStore(t)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/promotions/request", strings.NewReader(`{"strategy_id":"mstr-wave-fib","version":"v0.1.0","config_path":"configs/live.yaml"}`))
	request.Header.Set("Content-Type", "application/json")

	NewHandler(HandlerConfig{
		Store: store,
		Promotions: stubPromotionManager{
			request: promotion.Request{
				ID:         "promo-1",
				StrategyID: "mstr-wave-fib",
				Version:    "v0.1.0",
				State:      promotion.StateBacktestPassed,
			},
		},
	}).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status code: %d body=%s", recorder.Code, recorder.Body.String())
	}
	var payload struct {
		ID    string          `json:"id"`
		State promotion.State `json:"state"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode promotion payload: %v", err)
	}
	if payload.ID != "promo-1" || payload.State != promotion.StateBacktestPassed {
		t.Fatalf("unexpected promotion payload: %+v", payload)
	}
}

func TestRouterStrategyVersionsEndpoint(t *testing.T) {
	store := newTestStore(t)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/strategies/versions?strategy_id=mstr-wave-fib", nil)

	NewHandler(HandlerConfig{
		Store: store,
		Promotions: stubPromotionManager{
			requests: []promotion.Request{
				{
					ID:             "promo-2",
					StrategyID:     "mstr-wave-fib",
					Version:        "v0.1.1",
						ConfigPath:     "configs/live.yaml",
					State:          promotion.StateShadowPassed,
					ObjectiveScore: 0.81,
					FinalScore:     0.83,
					UpdatedAt:      time.Unix(1710000200, 0).UTC(),
				},
				{
					ID:             "promo-1",
					StrategyID:     "mstr-wave-fib",
					Version:        "v0.1.0",
						ConfigPath:     "configs/live.yaml",
					State:          promotion.StateBacktestPassed,
					ObjectiveScore: 0.79,
					FinalScore:     0.8,
					UpdatedAt:      time.Unix(1710000000, 0).UTC(),
				},
			},
		},
	}).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status code: %d body=%s", recorder.Code, recorder.Body.String())
	}
	var payload []struct {
		StrategyID string `json:"strategy_id"`
		Version    string `json:"version"`
		Promotion  struct {
			ID    string `json:"id"`
			State string `json:"state"`
		} `json:"promotion"`
		FinalScore float64 `json:"final_score"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode strategy versions payload: %v", err)
	}
	if len(payload) != 2 {
		t.Fatalf("unexpected strategy versions payload: %+v", payload)
	}
	if payload[0].Version != "v0.1.1" || payload[0].Promotion.ID != "promo-2" || payload[1].Version != "v0.1.0" {
		t.Fatalf("unexpected strategy versions ordering: %+v", payload)
	}
	if payload[0].StrategyID != "mstr-wave-fib" || payload[0].FinalScore != 0.83 {
		t.Fatalf("unexpected strategy versions payload: %+v", payload)
	}
}

func TestRouterLiveFlattenEndpoint(t *testing.T) {
	store := newTestStore(t)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/live/flatten", strings.NewReader(`{"symbol":"MSTRUSDT"}`))
	request.Header.Set("Content-Type", "application/json")

	NewHandler(HandlerConfig{
		Store: store,
		Live: stubLiveOps{
			result: live.FlattenResult{
				Symbol:      "MSTRUSDT",
				Status:      "flat",
				Qty:         0,
				ProductType: "USDT-FUTURES",
				MarginCoin:  "USDT",
				CheckedAt:   time.Unix(1710000300, 0).UTC(),
			},
		},
	}).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status code: %d body=%s", recorder.Code, recorder.Body.String())
	}
	var payload struct {
		Symbol string  `json:"symbol"`
		Status string  `json:"status"`
		Qty    float64 `json:"qty"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode live flatten payload: %v", err)
	}
	if payload.Symbol != "MSTRUSDT" || payload.Status != "flat" || payload.Qty != 0 {
		t.Fatalf("unexpected live flatten payload: %+v", payload)
	}
}

func newTestStore(t *testing.T) *sqlitepkg.Store {
	t.Helper()
	path := filepath.Join(t.TempDir(), "platform-state.db")
	store, err := sqlitepkg.NewStore(path)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	return store
}

func appendPositionEvent(t *testing.T, ctx context.Context, store *sqlitepkg.Store, event market.PositionEvent) {
	t.Helper()
	body, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal position event: %v", err)
	}
	if _, err := store.AppendEvent(ctx, "market.private", event, body); err != nil {
		t.Fatalf("append position event: %v", err)
	}
}

func appendOrderEvent(t *testing.T, ctx context.Context, store *sqlitepkg.Store, event market.OrderEvent) {
	t.Helper()
	body, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal order event: %v", err)
	}
	if _, err := store.AppendEvent(ctx, "market.private", event, body); err != nil {
		t.Fatalf("append order event: %v", err)
	}
}

type stubBacktestRunner struct {
	result backtest.Result
	err    error
}

func (runner stubBacktestRunner) RunConfig(_ context.Context, _ string, _ bool) (backtest.Result, error) {
	return runner.result, runner.err
}

type stubPromotionManager struct {
	request  promotion.Request
	requests []promotion.Request
	err      error
}

func (manager stubPromotionManager) CreateRequest(_ context.Context, _ promotion.CreateInput) (promotion.Request, error) {
	return manager.request, manager.err
}

func (manager stubPromotionManager) Get(_ context.Context, _ string) (promotion.Request, error) {
	return manager.request, manager.err
}

func (manager stubPromotionManager) List(_ context.Context) ([]promotion.Request, error) {
	if manager.requests != nil {
		return manager.requests, manager.err
	}
	return []promotion.Request{manager.request}, manager.err
}

func (manager stubPromotionManager) StartShadow(_ context.Context, _ string) (promotion.Request, error) {
	return manager.request, manager.err
}

func (manager stubPromotionManager) PassShadow(_ context.Context, _ string) (promotion.Request, error) {
	return manager.request, manager.err
}

func (manager stubPromotionManager) StartCanary(_ context.Context, _ string) (promotion.Request, error) {
	return manager.request, manager.err
}

func (manager stubPromotionManager) DegradeCanary(_ context.Context, _ string, _ string) (promotion.Request, error) {
	return manager.request, manager.err
}

func (manager stubPromotionManager) Approve(_ context.Context, _ string) (promotion.Request, error) {
	return manager.request, manager.err
}

func (manager stubPromotionManager) Rollback(_ context.Context, _ string, _ string) (promotion.Request, error) {
	return manager.request, manager.err
}

type stubQueryService struct {
	bars     query.BarsResult
	features query.FeaturesResult
	err      error
}

func (service stubQueryService) BarsFromConfig(_ context.Context, _ string, _ string, _ bool) (query.BarsResult, error) {
	return service.bars, service.err
}

func (service stubQueryService) FeaturesFromConfig(_ context.Context, _ string, _ string, _ bool, _ int) (query.FeaturesResult, error) {
	return service.features, service.err
}

type stubLiveOps struct {
	result live.FlattenResult
	err    error
}

func (ops stubLiveOps) FlattenSymbol(_ context.Context, _ string) (live.FlattenResult, error) {
	return ops.result, ops.err
}

func TestRouterStrategyVersionsEndpointIncludesRegistryEntriesWithoutPromotion(t *testing.T) {
	store := newTestStore(t)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/strategies/versions?strategy_id=mstr-wave-fib", nil)

	NewHandler(HandlerConfig{
		Store: store,
		Promotions: stubPromotionManager{
			requests: []promotion.Request{{
				ID:             "promo-1",
				StrategyID:     "mstr-wave-fib",
				Version:        "v0.1.0",
				ConfigPath:     "configs/live.yaml",
				State:          promotion.StateBacktestPassed,
				ObjectiveScore: 0.79,
				FinalScore:     0.8,
				UpdatedAt:      time.Unix(1710000000, 0).UTC(),
			}},
		},
		Strategies: stubStrategyRegistry{versions: []strategybundle.VersionInfo{
			{StrategyID: "mstr-wave-fib", Version: "v0.2.0", RootPath: "strategies/mstr-wave-fib/versions/v0.2.0", UpdatedAt: time.Unix(1710000300, 0).UTC()},
			{StrategyID: "mstr-wave-fib", Version: "v0.1.0", RootPath: "strategies/mstr-wave-fib/versions/v0.1.0", UpdatedAt: time.Unix(1710000000, 0).UTC()},
		}},
	}).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status code: %d body=%s", recorder.Code, recorder.Body.String())
	}
	var payload []struct {
		Version   string `json:"version"`
		RootPath  string `json:"root_path"`
		Promotion struct {
			ID string `json:"id"`
		} `json:"promotion"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode strategy versions payload: %v", err)
	}
	if len(payload) != 2 {
		t.Fatalf("unexpected strategy versions payload: %+v", payload)
	}
	if payload[0].Version != "v0.2.0" || payload[0].RootPath != "strategies/mstr-wave-fib/versions/v0.2.0" || payload[0].Promotion.ID != "" {
		t.Fatalf("unexpected registry-first payload: %+v", payload)
	}
	if payload[1].Version != "v0.1.0" || payload[1].Promotion.ID != "promo-1" {
		t.Fatalf("unexpected promotion-enriched payload: %+v", payload)
	}
}

type stubStrategyRegistry struct {
	versions []strategybundle.VersionInfo
	err      error
}

func (registry stubStrategyRegistry) List(_ string) ([]strategybundle.VersionInfo, error) {
	return registry.versions, registry.err
}
