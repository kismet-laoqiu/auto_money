# Agentic Runtime Closure Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Finish the current scaffold into a testable Bitget `USDT-FUTURES` runtime where `marketd` writes real futures public/private events into SQLite/WAL, `traderd` tails that log with deterministic rules and execution gates, `lab replay` uses the same runtime kernel, and `agentd` consumes trader advisories without execution authority. Exchange-facing work is not complete until it passes against a real Bitget environment on ECS, not only against mocks or fake servers.

**Architecture:** Keep the existing modular monolith and the approved four-process split. Replace the current ad hoc `market_event_log` with a sequenced generic `event_log` plus consumer cursors in SQLite/WAL, wire `marketd` as the only exchange ingress writer, wire `traderd` as the only deterministic decision and execution writer, and keep `agentd` read-only by consuming only trader advisory and risk events. Replay must reuse the same trader runtime entrypoint instead of calling `LegacyRuleProfile` directly.

**Tech Stack:** Go, SQLite/WAL, Bitget futures REST/WebSocket, Gorilla WebSocket, YAML, OpenAI Responses API, existing `quantlab/internal/core` compatibility path.

---

## Current Code Truth

- `cmd/marketd/main.go` opens the SQLite store but passes `nil` into `market.NewPublicFeed`, so no real source is connected yet.
- `cmd/traderd/main.go` opens the SQLite store, instantiates `trader.NewEngine`, and blocks without consuming any event.
- `cmd/agentd/main.go` only validates `live.agent.advisory_only` and `OPENAI_API_KEY`, then blocks.
- `cmd/lab/main.go` replay still builds `LegacyRuleProfile{StrategyCfg: cfg.Strategy}` directly.
- `internal/exchange/bitget/public_ws.go` decodes `trade` frames only; `ticker` and candle channels still return `nil`.
- `internal/exchange/bitget/private_ws.go` decodes `orders` only; `positions` and account channels still return `nil`.
- `internal/adapters/data.go` still fetches Bitget bars from `/api/v2/spot/market/candles`, which is the wrong endpoint for futures replay parity.
- `internal/store/sqlite/schema.sql` has no monotonic event sequence and no consumer cursor table.

## Approach Choice

1. Recommended: keep `marketd`, `traderd`, `agentd`, and `lab` split, converge them on one sequenced SQLite/WAL event log, and reuse one deterministic trader runtime from both live and replay.
2. Rejected: collapse everything into one process. That is faster short-term but breaks restart isolation and violates the approved runtime split from the design.
3. Rejected: rewrite the trader kernel wholesale before reconnecting I/O. The current code already has reusable parts (`StrategyRouter`, `PositionPolicy`, `RiskEngine`, `Reconciler`, `ExecutionCoordinator`), so a staged closure is safer and smaller.

## Real Bitget Verification Contract

- Every exchange-facing task in this plan has two verification layers: local package tests or fake servers for iteration speed, and real Bitget verification on ECS before the task can be called done.
- Real Bitget verification is mandatory for public REST market reads, public WebSocket trade and candle streams, private REST account/position/order reads, private WebSocket orders/positions/account streams, and the write path from order placement through order query and exit handling.
- ECS credentials must stay in environment variables only. The required runtime mapping for this plan is `BITGET_API_KEY`, `BITGET_API_SECRET`, and `BITGET_PASSPHRASE`, wired through `live.exchange.api_key_env`, `api_secret_env`, and `passphrase_env`. Raw secrets must never be written into tracked config files, plan docs, logs, or committed helper scripts.
- `BITGET_PASSPHRASE` is a hard prerequisite for any private REST, private WebSocket, order placement, order query, cancel, or reduce-only exit verification. Without it, any private/account/order task remains incomplete by definition.
- `go test ./...`, replay, and fake-server coverage are necessary development signals, but they are not the finish line. `mock pass != done`; real Bitget verification pass is the completion gate.

## Assumptions

- There is no released production runtime state that requires backwards-compatible DB migration; rename and reshape the SQLite schema directly instead of dual-write shims.
- Manual arming stays intentionally simple in phase 1: operator edits `live.runtime.arming_state` and restarts `traderd`. The runtime may auto-downgrade to `degraded` or `halted`, but it must never auto-promote back to `armed`.
- `agentd` remains advisory-only for the entire scope of this plan. It may read `traderd` events and write explanatory artifacts, but it must never gain exchange credentials or order routes.
- All real exchange verification runs on ECS against the actual Bitget environment configured by the live credentials. Public-surface checks may proceed with public endpoints alone, but private/account/order checks require valid key, secret, and passphrase in the ECS environment.
- Live order verification uses the smallest allowed `USDT-FUTURES` order size, isolated margin, and a mandatory reduce-only cleanup path whenever a test order creates exposure. Leaving a verification position open is a plan failure.

## File Structure Map

### Existing files to modify

- Modify: `internal/store/sqlite/schema.sql`
- Modify: `internal/store/sqlite/live_store.go`
- Modify: `internal/store/sqlite/live_store_test.go`
- Modify: `internal/market/event.go`
- Modify: `internal/market/public_feed.go`
- Modify: `internal/market/private_feed.go`
- Modify: `internal/exchange/bitget/rest_client.go`
- Modify: `internal/exchange/bitget/models.go`
- Modify: `internal/exchange/bitget/futures_market.go`
- Modify: `internal/exchange/bitget/futures_trade.go`
- Modify: `internal/exchange/bitget/public_ws.go`
- Modify: `internal/exchange/bitget/private_ws.go`
- Modify: `internal/exchange/bitget/public_ws_test.go`
- Modify: `internal/exchange/bitget/private_ws_test.go`
- Modify: `internal/adapters/data.go`
- Modify: `internal/trader/state.go`
- Modify: `internal/trader/engine.go`
- Modify: `internal/trader/engine_test.go`
- Modify: `internal/trader/strategy_router.go`
- Modify: `internal/trader/position_policy.go`
- Modify: `internal/trader/risk_engine.go`
- Modify: `internal/trader/execution_coordinator.go`
- Modify: `internal/trader/reconciler.go`
- Modify: `internal/trader/reconciler_test.go`
- Modify: `internal/trader/execution_coordinator_test.go`
- Modify: `internal/replay/event_source.go`
- Modify: `internal/replay/harness.go`
- Modify: `internal/replay/harness_test.go`
- Modify: `internal/agent/service.go`
- Modify: `internal/agent/jobs.go`
- Modify: `internal/agent/mcp.go`
- Modify: `internal/agent/service_test.go`
- Modify: `cmd/lab/main.go`
- Modify: `cmd/marketd/main.go`
- Modify: `cmd/traderd/main.go`
- Modify: `cmd/agentd/main.go`
- Modify: `scripts/measure.sh`
- Modify: `configs/demo-bitget.yaml`
- Modify: `configs/live-bitget.yaml`
- Modify: `plan/rollout-checklist.md`

### New files to create

- Create: `internal/exchange/bitget/futures_account.go`
- Create: `internal/exchange/bitget/futures_account_test.go`
- Create: `internal/exchange/bitget/ws_source.go`
- Create: `internal/exchange/bitget/ws_source_test.go`
- Create: `internal/market/gap_fill.go`
- Create: `internal/market/gap_fill_test.go`
- Create: `internal/market/runtime.go`
- Create: `internal/market/runtime_test.go`
- Create: `internal/trader/runtime.go`
- Create: `internal/trader/runtime_test.go`
- Create: `internal/trader/runtime_events.go`
- Create: `internal/trader/runtime_events_test.go`
- Create: `internal/agent/runtime.go`
- Create: `internal/agent/runtime_test.go`

## Task 1: Replace `market_event_log` with a generic sequenced event log and consumer cursors

**Files:**
- Modify: `internal/store/sqlite/schema.sql`
- Modify: `internal/store/sqlite/live_store.go`
- Modify: `internal/store/sqlite/live_store_test.go`

- [ ] **Step 1: Write the failing storage tests**

```go
func TestAppendEventReturnsMonotonicSeqAndListEventsAfter(t *testing.T) {
	store := newTestStore(t)
	first, err := store.AppendEvent(context.Background(), "market", market.BarClosedEvent{
		EventIDValue: "evt-1",
		SymbolValue:  "BTCUSDT",
		Interval:     "1m",
		Ts:           time.Unix(1710000000, 0),
	}, []byte(`{"close":"62000"}`))
	if err != nil {
		t.Fatalf("append first: %v", err)
	}
	second, err := store.AppendEvent(context.Background(), "trader", market.BarClosedEvent{
		EventIDValue: "evt-2",
		SymbolValue:  "BTCUSDT",
		Interval:     "1m",
		Ts:           time.Unix(1710000060, 0),
	}, []byte(`{"close":"62050"}`))
	if err != nil {
		t.Fatalf("append second: %v", err)
	}
	if second <= first {
		t.Fatalf("seq must increase: first=%d second=%d", first, second)
	}

	events, err := store.ListEventsAfter(context.Background(), first, 10, "trader")
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	if len(events) != 1 || events[0].Seq != second || events[0].Source != "trader" {
		t.Fatalf("unexpected events: %+v", events)
	}
}

func TestConsumerCursorRoundTrip(t *testing.T) {
	store := newTestStore(t)
	if err := store.SaveConsumerCursor(context.Background(), "traderd", 42); err != nil {
		t.Fatalf("save cursor: %v", err)
	}
	got, err := store.LoadConsumerCursor(context.Background(), "traderd")
	if err != nil {
		t.Fatalf("load cursor: %v", err)
	}
	if got != 42 {
		t.Fatalf("unexpected cursor: %d", got)
	}
}
```

- [ ] **Step 2: Run the storage tests to verify they fail**

Run: `PATH=/usr/local/go/bin:$PATH go test ./internal/store/sqlite -run 'TestAppendEventReturnsMonotonicSeqAndListEventsAfter|TestConsumerCursorRoundTrip' -v`

Expected: FAIL because `AppendEvent` does not accept `source` or return `seq`, `ListEventsAfter` does not exist, and no consumer cursor API exists.

- [ ] **Step 3: Implement the generic event log and cursor API**

```sql
CREATE TABLE IF NOT EXISTS event_log (
  seq INTEGER PRIMARY KEY AUTOINCREMENT,
  source TEXT NOT NULL,
  event_id TEXT NOT NULL UNIQUE,
  symbol TEXT NOT NULL,
  event_kind TEXT NOT NULL,
  exchange_ts TEXT NOT NULL,
  received_ts TEXT NOT NULL,
  payload_json BLOB NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_event_log_source_seq ON event_log(source, seq);
CREATE INDEX IF NOT EXISTS idx_event_log_kind_seq ON event_log(event_kind, seq);

CREATE TABLE IF NOT EXISTS consumer_cursor (
  consumer_key TEXT PRIMARY KEY,
  last_seq INTEGER NOT NULL,
  updated_at TEXT NOT NULL
);
```

```go
type LogEvent interface {
	EventID() string
	Symbol() string
	EventTime() time.Time
	Kind() string
}

type EventEnvelope struct {
	Seq        int64
	Source     string
	EventID    string
	Symbol     string
	Kind       string
	ExchangeTS time.Time
	ReceivedTS time.Time
	Payload    json.RawMessage
}

func (store *Store) AppendEvent(ctx context.Context, source string, evt LogEvent, raw []byte) (int64, error) {
	payload := raw
	if payload == nil {
		payload = []byte("null")
	}
	result, err := store.db.ExecContext(ctx, `
		INSERT INTO event_log (source, event_id, symbol, event_kind, exchange_ts, received_ts, payload_json)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, source, evt.EventID(), evt.Symbol(), evt.Kind(), evt.EventTime().UTC().Format(time.RFC3339Nano), time.Now().UTC().Format(time.RFC3339Nano), payload)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (store *Store) ListEventsAfter(ctx context.Context, afterSeq int64, limit int, sources ...string) ([]EventEnvelope, error) {
	// Build `WHERE seq > ?` plus optional `source IN (...)`, ordered by `seq ASC`, bounded by `limit`.
}

func (store *Store) SaveConsumerCursor(ctx context.Context, consumer string, seq int64) error {
	_, err := store.db.ExecContext(ctx, `
		INSERT INTO consumer_cursor (consumer_key, last_seq, updated_at)
		VALUES (?, ?, ?)
		ON CONFLICT(consumer_key) DO UPDATE SET last_seq=excluded.last_seq, updated_at=excluded.updated_at
	`, consumer, seq, time.Now().UTC().Format(time.RFC3339Nano))
	return err
}

func (store *Store) LoadConsumerCursor(ctx context.Context, consumer string) (int64, error) {
	var seq int64
	err := store.db.QueryRowContext(ctx, `SELECT last_seq FROM consumer_cursor WHERE consumer_key = ?`, consumer).Scan(&seq)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return seq, err
}
```

- [ ] **Step 4: Run the storage package tests**

Run: `PATH=/usr/local/go/bin:$PATH go test ./internal/store/sqlite -v`

Expected: PASS, including the new sequence and cursor tests plus the existing checkpoint round-trip tests.

- [ ] **Step 5: Commit the storage change**

```bash
git add internal/store/sqlite/schema.sql internal/store/sqlite/live_store.go internal/store/sqlite/live_store_test.go
git commit -m "feat: add sequenced runtime event log"
```

## Task 2: Complete the futures event model and Bitget futures decode/client surface

**Files:**
- Modify: `internal/market/event.go`
- Modify: `internal/exchange/bitget/rest_client.go`
- Modify: `internal/exchange/bitget/models.go`
- Modify: `internal/exchange/bitget/futures_market.go`
- Modify: `internal/exchange/bitget/futures_trade.go`
- Modify: `internal/exchange/bitget/public_ws.go`
- Modify: `internal/exchange/bitget/private_ws.go`
- Modify: `internal/exchange/bitget/public_ws_test.go`
- Modify: `internal/exchange/bitget/private_ws_test.go`
- Create: `internal/exchange/bitget/futures_account.go`
- Create: `internal/exchange/bitget/futures_account_test.go`

- [ ] **Step 1: Write the failing decode and private REST tests**

```go
func TestDecodePublicCandleFrameIntoBarClosedEvent(t *testing.T) {
	raw := []byte(`{"arg":{"instId":"BTCUSDT","channel":"candle1m"},"data":[["1710000000000","62000","62100","61950","62080","12.5"]]}`)
	events, err := DecodePublicEvents(raw)
	if err != nil {
		t.Fatalf("decode public candle: %v", err)
	}
	if len(events) != 1 || events[0].Kind() != "bar_closed" {
		t.Fatalf("unexpected events: %+v", events)
	}
}

func TestDecodePrivatePositionEvent(t *testing.T) {
	raw := []byte(`{"arg":{"channel":"positions","instId":"BTCUSDT"},"data":[{"instId":"BTCUSDT","holdSide":"long","total":"0.02","openPriceAvg":"62010","marginMode":"isolated","posMode":"one_way_mode","uTime":"1710000000000"}]}`)
	events, err := DecodePrivateEvents(raw)
	if err != nil {
		t.Fatalf("decode private positions: %v", err)
	}
	if len(events) != 1 || events[0].Kind() != "position_snapshot" {
		t.Fatalf("unexpected events: %+v", events)
	}
}

func TestDecodePrivateAccountEvent(t *testing.T) {
	raw := []byte(`{"arg":{"channel":"account","instId":"USDT"},"data":[{"marginCoin":"USDT","available":"1000","accountEquity":"1012.5","uTime":"1710000000000"}]}`)
	events, err := DecodePrivateEvents(raw)
	if err != nil {
		t.Fatalf("decode private account: %v", err)
	}
	if len(events) != 1 || events[0].Kind() != "account_snapshot" {
		t.Fatalf("unexpected events: %+v", events)
	}
}

func TestDoPrivateSetsBitgetHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("ACCESS-KEY") != "key-1" {
			t.Fatalf("missing ACCESS-KEY: %+v", r.Header)
		}
		if r.Header.Get("ACCESS-SIGN") == "" || r.Header.Get("ACCESS-PASSPHRASE") != "pass-1" {
			t.Fatalf("unexpected auth headers: %+v", r.Header)
		}
		w.Write([]byte(`{"code":"00000","data":[]}`))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	_, err := client.doPrivate(context.Background(), http.MethodGet, "/api/v2/mix/account/accounts?productType=USDT-FUTURES", nil, PrivateCredentials{
		Key: "key-1", Secret: "secret-1", Passphrase: "pass-1",
	})
	if err != nil {
		t.Fatalf("do private: %v", err)
	}
}
```

- [ ] **Step 2: Run the exchange tests to verify they fail**

Run: `PATH=/usr/local/go/bin:$PATH go test ./internal/exchange/bitget -run 'TestDecodePublicCandleFrameIntoBarClosedEvent|TestDecodePrivatePositionEvent|TestDecodePrivateAccountEvent|TestDoPrivateSetsBitgetHeaders' -v`

Expected: FAIL because candle, position, and account decoders are incomplete and `doPrivate` does not exist.

- [ ] **Step 3: Implement the futures event types, decode paths, and private client**

```go
type PositionEvent struct {
	EventIDValue string
	SymbolValue  string
	Ts           time.Time
	Qty          float64
	AvgPrice     float64
	MarginMode   string
	PositionMode string
	KindValue    string
}

type AccountEvent struct {
	EventIDValue string
	SymbolValue  string
	Ts           time.Time
	MarginCoin   string
	Available    float64
	Equity       float64
	KindValue    string
}
```

```go
func (client *Client) doPrivate(ctx context.Context, method, path string, body []byte, creds PrivateCredentials) ([]byte, error) {
	ts := strconv.FormatInt(time.Now().UTC().UnixMilli(), 10)
	signer := NewSigner(creds.Secret)
	req, err := http.NewRequestWithContext(ctx, method, client.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("ACCESS-KEY", creds.Key)
	req.Header.Set("ACCESS-PASSPHRASE", creds.Passphrase)
	req.Header.Set("ACCESS-TIMESTAMP", ts)
	req.Header.Set("ACCESS-SIGN", signer.Sign(ts, method, path, string(body)))
	req.Header.Set("Content-Type", "application/json")
	return client.do(req)
}
```

```go
func DecodePublicEvents(raw []byte) ([]market.MarketEvent, error) {
	var msg publicMessage
	if err := json.Unmarshal(raw, &msg); err != nil {
		return nil, err
	}
	switch msg.Arg.Channel {
	case "trade":
		return decodeTrades(msg)
	case "ticker":
		return decodeTickers(msg)
	default:
		if strings.HasPrefix(msg.Arg.Channel, "candle") {
			return decodeCandles(msg)
		}
		return nil, nil
	}
}

func decodeCandles(msg publicMessage) ([]market.MarketEvent, error) {
	// Parse ts/open/high/low/close/volume and emit `market.BarClosedEvent`.
}

func decodePositionEvents(msg privateMessage) ([]market.MarketEvent, error) {
	// Parse qty, avg price, margin mode, position mode, and `uTime`.
}

func decodeAccountEvents(msg privateMessage) ([]market.MarketEvent, error) {
	// Parse available balance and account equity into `market.AccountEvent`.
}
```

```go
func (client *Client) FetchFuturesPositions(ctx context.Context, creds PrivateCredentials, productType string) ([]market.PositionEvent, error) {
	body, err := client.doPrivate(ctx, http.MethodGet, "/api/v2/mix/position/all-position?productType="+url.QueryEscape(productType), nil, creds)
	if err != nil {
		return nil, err
	}
	return decodePositionsResponse(body)
}

func (client *Client) FetchFuturesAccount(ctx context.Context, creds PrivateCredentials, productType string) ([]market.AccountEvent, error) {
	body, err := client.doPrivate(ctx, http.MethodGet, "/api/v2/mix/account/accounts?productType="+url.QueryEscape(productType), nil, creds)
	if err != nil {
		return nil, err
	}
	return decodeAccountsResponse(body)
}
```

- [ ] **Step 4: Run the exchange package tests**

Run: `PATH=/usr/local/go/bin:$PATH go test ./internal/exchange/bitget -v`

Expected: PASS, including the old signer and metadata tests plus the new candle, position, account, and private request tests.

- [ ] **Step 5: Run real Bitget read-side verification on ECS**

Run:
- `PATH=/usr/local/go/bin:$PATH RUN_BITGET_REAL=1 BITGET_API_KEY=$BITGET_API_KEY BITGET_API_SECRET=$BITGET_API_SECRET BITGET_PASSPHRASE=$BITGET_PASSPHRASE go test ./internal/exchange/bitget -run RealBitget -v`

Expected:
- Real Bitget market reads prove futures metadata and candle/bootstrap reads work against the actual Bitget environment instead of a fake server.
- Real Bitget private reads prove signed account and position queries succeed with the ECS-provided credentials.
- If `BITGET_PASSPHRASE` is absent or invalid, the private read portion must fail loudly or skip loudly, and Task 2 remains incomplete.

- [ ] **Step 6: Commit the futures decode and private client change**

```bash
git add internal/market/event.go internal/exchange/bitget/rest_client.go internal/exchange/bitget/models.go internal/exchange/bitget/futures_market.go internal/exchange/bitget/futures_trade.go internal/exchange/bitget/public_ws.go internal/exchange/bitget/private_ws.go internal/exchange/bitget/public_ws_test.go internal/exchange/bitget/private_ws_test.go internal/exchange/bitget/futures_account.go internal/exchange/bitget/futures_account_test.go
git commit -m "feat: add bitget futures decode and private rest client"
```

## Task 3: Add raw websocket sources, bootstrap gap fill, and a real `marketd` collector

**Files:**
- Create: `internal/exchange/bitget/ws_source.go`
- Create: `internal/exchange/bitget/ws_source_test.go`
- Create: `internal/market/gap_fill.go`
- Create: `internal/market/gap_fill_test.go`
- Create: `internal/market/runtime.go`
- Create: `internal/market/runtime_test.go`
- Modify: `internal/market/public_feed.go`
- Modify: `internal/market/private_feed.go`
- Modify: `cmd/marketd/main.go`

- [ ] **Step 1: Write the failing collector tests**

```go
func TestRunCollectorWritesBootstrapAndStreamingEvents(t *testing.T) {
	store := newTestStore(t)
	cfg := testLiveConfig()
	publicSource := newFakeSource(
		[]byte(`{"arg":{"instId":"BTCUSDT","channel":"trade"},"data":[["1710000000000","62000","0.01","buy"]]}`),
		[]byte(`{"arg":{"instId":"BTCUSDT","channel":"candle1m"},"data":[["1710000060000","62000","62100","61980","62090","11"]]}`),
	)
	privateSource := newFakeSource(
		[]byte(`{"arg":{"channel":"orders","instId":"BTCUSDT"},"data":[{"clientOid":"ql-1","orderId":"123","status":"filled","size":"0.01","priceAvg":"62090","uTime":"1710000060000"}]}`),
	)
	bootstrap := []sqlite.LogEvent{
		market.BarClosedEvent{EventIDValue: "bootstrap-1", SymbolValue: "BTCUSDT", Interval: "1m", Ts: time.Unix(1709999940, 0)},
	}

	err := RunCollector(context.Background(), cfg, store, bootstrap, publicSource, privateSource)
	if err != nil {
		t.Fatalf("run collector: %v", err)
	}

	events, err := store.ListEventsAfter(context.Background(), 0, 10)
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	if len(events) < 4 {
		t.Fatalf("expected bootstrap + public + private events, got %+v", events)
	}
}

func TestBuildBootstrapEventsUsesFuturesCandles(t *testing.T) {
	client := &fakeBootstrapClient{bars: []core.Bar{{Time: time.Unix(1710000000, 0), Close: 62000}}}
	cfg := testLiveConfig()
	events, err := BuildBootstrapEvents(context.Background(), client, cfg)
	if err != nil {
		t.Fatalf("build bootstrap: %v", err)
	}
	if len(events) != 1 || events[0].Kind() != "bar_closed" {
		t.Fatalf("unexpected bootstrap events: %+v", events)
	}
}
```

- [ ] **Step 2: Run the market package tests to verify they fail**

Run: `PATH=/usr/local/go/bin:$PATH go test ./internal/market -run 'TestRunCollectorWritesBootstrapAndStreamingEvents|TestBuildBootstrapEventsUsesFuturesCandles' -v`

Expected: FAIL because `RunCollector`, websocket raw sources, and bootstrap builders do not exist.

- [ ] **Step 3: Implement websocket raw sources and bootstrap gap fill**

```go
type RawSource struct {
	url          string
	loginFrame   []byte
	subscribe    []byte
	dialer       websocket.Dialer
	pingInterval time.Duration
}

func (source *RawSource) Events(ctx context.Context) <-chan []byte {
	out := make(chan []byte)
	go func() {
		defer close(out)
		// Dial, optional login, subscribe, relay each raw frame, reconnect with backoff until ctx done.
	}()
	return out
}

func NewPublicSource(url, productType string, symbols []config.LiveSymbolConfig) *RawSource {
	// Build futures public subscriptions for `trade` and `candle1m`.
}

func MaybeNewPrivateSource(cfg config.Config) (*RawSource, error) {
	// Return `nil, nil` when `observe_only=true` and credentials are absent.
	// Otherwise build a login frame plus `orders`, `positions`, and `account` subscriptions.
}
```

```go
type BootstrapClient interface {
	FetchBars(ctx context.Context, spec config.DatasetConfig) ([]core.Bar, error)
	FetchFuturesPositions(ctx context.Context, creds bitget.PrivateCredentials, productType string) ([]market.PositionEvent, error)
	FetchFuturesAccount(ctx context.Context, creds bitget.PrivateCredentials, productType string) ([]market.AccountEvent, error)
}

func BuildBootstrapEvents(ctx context.Context, client BootstrapClient, cfg config.Config) ([]sqlite.LogEvent, error) {
	// Fetch recent futures bars for configured symbols.
	// If private credentials are present, append position and account snapshots too.
}

func NewBootstrapClient(client *bitget.Client, creds bitget.PrivateCredentials) BootstrapClient {
	// Return the concrete bootstrap adapter used by `marketd`.
}
```

- [ ] **Step 4: Implement the collector runtime and wire `cmd/marketd`**

```go
type PrivateStreamFeed struct {
	source  EventSource
	decoder PrivateFeed
}

func NewPrivateFeed(source EventSource, decoder PrivateFeed) *PrivateStreamFeed {
	return &PrivateStreamFeed{source: source, decoder: decoder}
}

func (feed *PrivateStreamFeed) Events(ctx context.Context) <-chan MarketEvent {
	// Same pattern as public feed, but without the micro-bar aggregator.
}

func RunCollector(ctx context.Context, cfg config.Config, store *sqlite.Store, bootstrap []sqlite.LogEvent, publicSource, privateSource market.EventSource) error {
	for _, evt := range bootstrap {
		if _, err := store.AppendEvent(ctx, "market.bootstrap", evt, nil); err != nil {
			return err
		}
	}

	publicFeed := NewPublicFeed(publicSource, PublicDecoder(bitget.DecodePublicEvents), NewMicroBarAggregator(time.Second))
	privateFeed := NewPrivateFeed(privateSource, PrivateDecoder(bitget.DecodePrivateEvents))

	for evt := range MergeEventChannels(ctx, publicFeed.Events(ctx), privateFeed.Events(ctx)) {
		source := "market.public"
		if isPrivateEvent(evt) {
			source = "market.private"
		}
		if _, err := store.AppendEvent(ctx, source, evt, mustJSON(evt)); err != nil {
			return err
		}
	}
	return ctx.Err()
}

func MergeEventChannels(ctx context.Context, chans ...<-chan market.MarketEvent) <-chan market.MarketEvent {
	// Fan in each channel, close when all inputs finish or ctx is done.
}

func isPrivateEvent(evt market.MarketEvent) bool {
	switch evt.(type) {
	case market.OrderEvent, market.PositionEvent, market.AccountEvent:
		return true
	default:
		return false
	}
}

func mustJSON(v any) []byte {
	body, _ := json.Marshal(v)
	return body
}
```

```go
func run(ctx context.Context, cfg config.Config) error {
	store, err := sqlitepkg.NewStore(cfg.Live.Runtime.StateDBPath)
	if err != nil {
		return err
	}
	client := bitget.NewClient(cfg.Live.Exchange.RESTBaseURL)
	creds := bitget.PrivateCredentials{
		Key:        os.Getenv(cfg.Live.Exchange.APIKeyEnv),
		Secret:     os.Getenv(cfg.Live.Exchange.APISecretEnv),
		Passphrase: os.Getenv(cfg.Live.Exchange.PassphraseEnv),
	}
	bootstrap, err := market.BuildBootstrapEvents(ctx, market.NewBootstrapClient(client, creds), cfg)
	if err != nil {
		return err
	}
	publicSource := bitget.NewPublicSource(cfg.Live.Exchange.PublicWSURL, cfg.Live.Exchange.ProductType, cfg.Live.Exchange.Symbols)
	privateSource, err := bitget.MaybeNewPrivateSource(cfg)
	if err != nil {
		return err
	}
	return market.RunCollector(ctx, cfg, store, bootstrap, publicSource, privateSource)
}
```

- [ ] **Step 5: Run the market and build verification**

Run:
- `PATH=/usr/local/go/bin:$PATH go test ./internal/market ./internal/exchange/bitget -v`
- `PATH=/usr/local/go/bin:$PATH go build ./cmd/marketd`

Expected: PASS.

- [ ] **Step 6: Run real `marketd` ingress verification against Bitget on ECS**

Run:
- `PATH=/usr/local/go/bin:$PATH RUN_BITGET_REAL=1 BITGET_API_KEY=$BITGET_API_KEY BITGET_API_SECRET=$BITGET_API_SECRET BITGET_PASSPHRASE=$BITGET_PASSPHRASE go test ./internal/market -run RealBitget -v`
- `sqlite3 <state_db_path> "SELECT source, event_kind, count(*) FROM event_log GROUP BY source, event_kind ORDER BY source, event_kind;"`

Expected:
- `event_log` contains real `market.public` rows for `trade_tick` and `bar_closed`.
- With valid private credentials plus passphrase, `event_log` also contains real `market.private` rows for `position_snapshot` and `account_snapshot`.
- `order_fill` may still be absent before Task 5 creates a real order lifecycle, but after Task 5 the same collector path must show that event kind too. If the collector cannot ingest private snapshots from real Bitget, Task 3 remains incomplete.

- [ ] **Step 7: Commit the real market ingress change**

```bash
git add internal/exchange/bitget/ws_source.go internal/exchange/bitget/ws_source_test.go internal/market/gap_fill.go internal/market/gap_fill_test.go internal/market/runtime.go internal/market/runtime_test.go internal/market/public_feed.go internal/market/private_feed.go cmd/marketd/main.go
git commit -m "feat: connect marketd to real bitget futures sources"
```

## Task 4: Strip execution side effects out of `Engine` and add a resumable trader runtime

**Files:**
- Create: `internal/trader/runtime.go`
- Create: `internal/trader/runtime_test.go`
- Create: `internal/trader/runtime_events.go`
- Create: `internal/trader/runtime_events_test.go`
- Modify: `internal/trader/state.go`
- Modify: `internal/trader/engine.go`
- Modify: `internal/trader/engine_test.go`
- Modify: `internal/trader/strategy_router.go`
- Modify: `internal/trader/position_policy.go`
- Modify: `cmd/traderd/main.go`

- [ ] **Step 1: Write the failing trader runtime tests**

```go
func TestRuntimeRestoresCheckpointAndCursor(t *testing.T) {
	store := newTestStore(t)
	if err := store.SaveCheckpoint(context.Background(), "traderd", EngineState{
		ArmingState: ArmingSafe,
		Symbols: map[string]SymbolState{
			"BTCUSDT": {Phase: PhaseWatching, Tranches: 1},
		},
	}); err != nil {
		t.Fatalf("save checkpoint: %v", err)
	}
	if err := store.SaveConsumerCursor(context.Background(), "traderd", 9); err != nil {
		t.Fatalf("save cursor: %v", err)
	}
	runtime, err := NewRuntime(RuntimeConfig{ConsumerKey: "traderd", Store: store})
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}
	if runtime.cursor != 9 || runtime.state.Symbols["BTCUSDT"].Tranches != 1 {
		t.Fatalf("runtime did not restore persisted state: %+v", runtime)
	}
}

func TestRuntimeWritesCandidateEnvelopeForBarClosed(t *testing.T) {
	store := newTestStore(t)
	runtime, err := NewRuntime(RuntimeConfig{
		Store:              store,
		ConsumerKey:        "traderd",
		InitialArmingState: ArmingSafe,
		ObserveOnly:        true,
		StrategyCfg:        config.StrategyConfig{FastSMA: 3, SlowSMA: 5, ATRWindow: 3, LevelLookback: 12, PivotWindow: 1},
		Strategy:           LegacyRuleProfile{StrategyCfg: config.StrategyConfig{FastSMA: 3, SlowSMA: 5, ATRWindow: 3, LevelLookback: 12, PivotWindow: 1}},
		Policy:             PolicyConfig{MaxTranches: 3},
	})
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}
	seq, err := store.AppendEvent(context.Background(), "market.public", market.BarClosedEvent{
		EventIDValue: "bar-1",
		SymbolValue:  "BTCUSDT",
		Interval:     "1m",
		Ts:           time.Unix(1710000000, 0),
		Close:        62000,
	}, []byte(`{"close":"62000"}`))
	if err != nil {
		t.Fatalf("append market event: %v", err)
	}
	if err := runtime.ProcessAvailable(context.Background()); err != nil {
		t.Fatalf("process available: %v", err)
	}
	events, err := store.ListEventsAfter(context.Background(), seq, 10, "trader")
	if err != nil {
		t.Fatalf("list trader events: %v", err)
	}
	if len(events) == 0 {
		t.Fatalf("expected trader output after bar close")
	}
}
```

- [ ] **Step 2: Run the trader tests to verify they fail**

Run: `PATH=/usr/local/go/bin:$PATH go test ./internal/trader -run 'TestRuntimeRestoresCheckpointAndCursor|TestRuntimeWritesCandidateEnvelopeForBarClosed' -v`

Expected: FAIL because no runtime exists and `Engine` still owns exchange side effects.

- [ ] **Step 3: Make `Engine` deterministic only and add trader runtime events**

```go
type Config struct {
	ArmingState ArmingState
	Strategy    Strategy
}

func (engine *Engine) Advance(evt market.MarketEvent) ([]Command, error) {
	switch event := evt.(type) {
	case market.BarClosedEvent:
		return engine.advanceBar(event.SymbolValue, event.Interval, core.Bar{Time: event.Ts, Open: event.Open, High: event.High, Low: event.Low, Close: event.Close, Volume: event.Volume})
	case market.MicroBarClosedEvent:
		return engine.advanceBar(event.SymbolValue, "1s", core.Bar{Time: event.ClosedAt, Open: event.Open, High: event.High, Low: event.Low, Close: event.Close, Volume: event.Volume})
	default:
		return nil, nil
	}
}
```

```go
type CandidateEvent struct {
	EventIDValue string
	SymbolValue  string
	Ts           time.Time
	KindValue    string
	Profile      string
	Phase        Phase
	Candidate    Candidate
}

func (event CandidateEvent) EventID() string      { return event.EventIDValue }
func (event CandidateEvent) Symbol() string       { return event.SymbolValue }
func (event CandidateEvent) EventTime() time.Time { return event.Ts }
func (event CandidateEvent) Kind() string         { return event.KindValue }

type RiskEvent struct {
	EventIDValue string
	SymbolValue  string
	Ts           time.Time
	KindValue    string
	Reason       string
	ArmingState  ArmingState
}

func (event RiskEvent) EventID() string      { return event.EventIDValue }
func (event RiskEvent) Symbol() string       { return event.SymbolValue }
func (event RiskEvent) EventTime() time.Time { return event.Ts }
func (event RiskEvent) Kind() string         { return event.KindValue }

type LiveExchange interface {
	PlaceOrder(ctx context.Context, req bitget.PlaceOrderRequest) error
}
```

```go
type Runtime struct {
	store       *sqlite.Store
	engine      *Engine
	router      *StrategyRouter
	policy      *PositionPolicy
	risk        *RiskEngine
	exchange    LiveExchange
	consumerKey string
	cursor      int64
	state       EngineState
	observeOnly bool
	strategyCfg config.StrategyConfig
	accountEquity float64
	runID       string
	symbolCaps  map[string]float64
}

type RuntimeConfig struct {
	Store              *sqlite.Store
	ConsumerKey        string
	InitialArmingState ArmingState
	ObserveOnly        bool
	StrategyCfg        config.StrategyConfig
	Strategy           Strategy
	Policy             PolicyConfig
	Risk               RiskConfig
	Exchange           LiveExchange
	RunID              string
	SymbolCaps         map[string]float64
}

func NewRuntime(cfg RuntimeConfig) (*Runtime, error) {
	state, err := cfg.Store.LoadCheckpoint(context.Background(), cfg.ConsumerKey)
	if err != nil {
		return nil, err
	}
	cursor, err := cfg.Store.LoadConsumerCursor(context.Background(), cfg.ConsumerKey)
	if err != nil {
		return nil, err
	}
	if state.ArmingState == "" {
		state.ArmingState = cfg.InitialArmingState
	}
	return &Runtime{
		store:       cfg.Store,
		engine:      NewEngine(Config{ArmingState: state.ArmingState, Strategy: cfg.Strategy}),
		router:      NewStrategyRouter(),
		policy:      NewPositionPolicy(cfg.Policy),
		risk:        NewRiskEngine(cfg.Risk),
		exchange:    cfg.Exchange,
		consumerKey: cfg.ConsumerKey,
		cursor:      cursor,
		state:       state,
		observeOnly: cfg.ObserveOnly,
		strategyCfg: cfg.StrategyCfg,
		runID:       cfg.RunID,
		symbolCaps:  cfg.SymbolCaps,
	}, nil
}
```

- [ ] **Step 4: Implement event processing, feature snapshots, and checkpoint writes**

```go
func (runtime *Runtime) ProcessAvailable(ctx context.Context) error {
	envelopes, err := runtime.store.ListEventsAfter(ctx, runtime.cursor, 256, "market.bootstrap", "market.public", "market.private", "replay")
	if err != nil {
		return err
	}
	for _, env := range envelopes {
		if err := runtime.handleEnvelope(ctx, env); err != nil {
			return err
		}
		runtime.cursor = env.Seq
		if err := runtime.store.SaveConsumerCursor(ctx, runtime.consumerKey, runtime.cursor); err != nil {
			return err
		}
		if err := runtime.store.SaveCheckpoint(ctx, runtime.consumerKey, runtime.state); err != nil {
			return err
		}
	}
	return nil
}

func (runtime *Runtime) Run(ctx context.Context) error {
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	for {
		if err := runtime.ProcessAvailable(ctx); err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (runtime *Runtime) Store() *sqlite.Store {
	return runtime.store
}

func (runtime *Runtime) handleBarClosed(ctx context.Context, event market.BarClosedEvent) error {
	features := core.ExtractFeatureSet(runtime.engine.bars[event.SymbolValue+":"+event.Interval], len(runtime.engine.bars[event.SymbolValue+":"+event.Interval])-1, runtime.strategyCfg)
	symbolState := runtime.state.Symbols[event.SymbolValue]
	symbolState.ContextOK = features.Regime.TrendUpFlag || features.Trigger.TriggerQualityScore > 0.8
	symbolState.RSI14 = features.Trigger.RSI14
	symbolState.NeedleDropPct = features.Trigger.NeedleDropPct
	symbolState.ReclaimPct = features.Trigger.ReclaimPct
	profile := runtime.router.Select(symbolState)
	symbolState.Phase = runtime.policy.DesiredPhase(profile)
	runtime.state.Symbols[event.SymbolValue] = symbolState

	commands, err := runtime.engine.Advance(event)
	if err != nil {
		return err
	}
	for _, cmd := range commands {
		candidate := cmd.(Candidate)
		body, _ := json.Marshal(candidate)
		_, err := runtime.store.AppendEvent(ctx, "trader", CandidateEvent{
			EventIDValue: fmt.Sprintf("candidate:%s:%d", candidate.Symbol, candidate.Ts.UnixMilli()),
			SymbolValue:  candidate.Symbol,
			Ts:           candidate.Ts,
			KindValue:    "candidate.created",
			Profile:      profile.Name(),
			Phase:        symbolState.Phase,
			Candidate:    candidate,
		}, body)
		if err != nil {
			return err
		}
		if err := runtime.maybeExecuteCandidate(ctx, CandidateEvent{
			EventIDValue: fmt.Sprintf("candidate:%s:%d", candidate.Symbol, candidate.Ts.UnixMilli()),
			SymbolValue:  candidate.Symbol,
			Ts:           candidate.Ts,
			KindValue:    "candidate.created",
			Profile:      profile.Name(),
			Phase:        symbolState.Phase,
			Candidate:    candidate,
		}); err != nil {
			return err
		}
	}
	return nil
}
```

- [ ] **Step 5: Run the trader package tests and build `traderd`**

Run:
- `PATH=/usr/local/go/bin:$PATH go test ./internal/trader -v`
- `PATH=/usr/local/go/bin:$PATH go build ./cmd/traderd`

Expected: PASS. `Engine` tests should now assert candidate emission only, and the new runtime tests should pass.

- [ ] **Step 6: Commit the deterministic runtime split**

```bash
git add internal/trader/runtime.go internal/trader/runtime_test.go internal/trader/runtime_events.go internal/trader/runtime_events_test.go internal/trader/state.go internal/trader/engine.go internal/trader/engine_test.go internal/trader/strategy_router.go internal/trader/position_policy.go cmd/traderd/main.go
git commit -m "refactor: move trader execution side effects out of engine"
```

## Task 5: Integrate reconciliation, risk gates, and exchange-backed execution into `traderd`

**Files:**
- Modify: `internal/trader/runtime.go`
- Modify: `internal/trader/risk_engine.go`
- Modify: `internal/trader/execution_coordinator.go`
- Modify: `internal/trader/reconciler.go`
- Modify: `internal/trader/reconciler_test.go`
- Modify: `internal/trader/execution_coordinator_test.go`
- Modify: `internal/exchange/bitget/futures_trade.go`
- Modify: `internal/exchange/bitget/futures_account.go`
- Modify: `cmd/traderd/main.go`

- [ ] **Step 1: Write the failing execution and reconciliation tests**

```go
func TestRuntimeDoesNotPlaceOrdersInSafeMode(t *testing.T) {
	store := newTestStore(t)
	exchange := &fakeExchange{}
	runtime, err := NewRuntime(RuntimeConfig{
		Store:              store,
		ConsumerKey:        "traderd",
		InitialArmingState: ArmingSafe,
		ObserveOnly:        false,
		Exchange:           exchange,
		Risk:               RiskConfig{MaxLeverage: 3},
	})
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}
	_, err = store.AppendEvent(context.Background(), "market.public", market.BarClosedEvent{
		EventIDValue: "bar-1",
		SymbolValue:  "BTCUSDT",
		Interval:     "1m",
		Ts:           time.Unix(1710000000, 0),
		Close:        62000,
	}, []byte(`{"close":"62000"}`))
	if err != nil {
		t.Fatalf("append bar: %v", err)
	}
	if err := runtime.ProcessAvailable(context.Background()); err != nil {
		t.Fatalf("process available: %v", err)
	}
	if exchange.placeCalls != 0 {
		t.Fatalf("safe mode must not place orders")
	}
}

func TestRuntimePlacesOrdersOnlyWhenArmedAndRiskPasses(t *testing.T) {
	store := newTestStore(t)
	exchange := &fakeExchange{}
	runtime, err := NewRuntime(RuntimeConfig{
		Store:              store,
		ConsumerKey:        "traderd",
		InitialArmingState: ArmingArmed,
		ObserveOnly:        false,
		Exchange:           exchange,
		Risk:               RiskConfig{MaxLeverage: 3},
	})
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}
	runtime.accountEquity = 1000
	_, err = store.AppendEvent(context.Background(), "market.public", market.BarClosedEvent{
		EventIDValue: "bar-1",
		SymbolValue:  "BTCUSDT",
		Interval:     "1m",
		Ts:           time.Unix(1710000000, 0),
		Close:        62000,
	}, []byte(`{"close":"62000"}`))
	if err != nil {
		t.Fatalf("append bar: %v", err)
	}
	if err := runtime.ProcessAvailable(context.Background()); err != nil {
		t.Fatalf("process available: %v", err)
	}
	if exchange.placeCalls != 1 {
		t.Fatalf("expected one order placement, got %d", exchange.placeCalls)
	}
}

func TestRuntimeDowngradesOnPositionMismatch(t *testing.T) {
	rec := NewReconciler()
	verdict := rec.Compare(PositionSnapshot{Symbol: "BTCUSDT", Qty: 0.01}, PositionSnapshot{Symbol: "BTCUSDT", Qty: 0.02})
	if verdict.NextArmingState != ArmingDegraded {
		t.Fatalf("unexpected verdict: %+v", verdict)
	}
}

type fakeExchange struct {
	placeCalls int
}

func (exchange *fakeExchange) PlaceOrder(_ context.Context, _ bitget.PlaceOrderRequest) error {
	exchange.placeCalls++
	return nil
}
```

- [ ] **Step 2: Run the trader tests to verify they fail**

Run: `PATH=/usr/local/go/bin:$PATH go test ./internal/trader -run 'TestRuntimeDoesNotPlaceOrdersInSafeMode|TestRuntimePlacesOrdersOnlyWhenArmedAndRiskPasses|TestRuntimeDowngradesOnPositionMismatch' -v`

Expected: FAIL because `Runtime` still does not execute candidate envelopes or react to position snapshots.

- [ ] **Step 3: Add entry execution, reduce-only exits, and downward-only arming transitions**

```go
func BuildEntryRequest(candidate Candidate, maxNotional float64, runID string) bitget.PlaceOrderRequest {
	return bitget.PlaceOrderRequest{
		Symbol:      candidate.Symbol,
		ProductType: "USDT-FUTURES",
		MarginMode:  "isolated",
		MarginCoin:  "USDT",
		Side:        "buy",
		OrderType:   "market",
		Size:        formatEntrySize(candidate.Entry, maxNotional),
		ClientOID:   BuildClientOID(runID, candidate.Symbol, 1, candidate.Ts.UnixMilli()),
	}
}

func formatEntrySize(entryPrice, maxNotional float64) string {
	if entryPrice <= 0 || maxNotional <= 0 {
		return "0"
	}
	return strconv.FormatFloat(maxNotional/entryPrice, 'f', 3, 64)
}

func (runtime *Runtime) maybeExecuteCandidate(ctx context.Context, event CandidateEvent) error {
	verdict := runtime.risk.Check(EntryIntent{Symbol: event.SymbolValue, Notional: event.Candidate.Entry, Equity: runtime.accountEquity})
	if !verdict.Allow {
		return runtime.appendRiskEvent(ctx, event.SymbolValue, "risk.execution_blocked", verdict.Reason)
	}
	if runtime.observeOnly || runtime.state.ArmingState != ArmingArmed || runtime.exchange == nil {
		return nil
	}
	return runtime.exchange.PlaceOrder(ctx, BuildEntryRequest(event.Candidate, runtime.symbolCaps[event.SymbolValue], runtime.runID))
}

func (runtime *Runtime) applyReconcileVerdict(ctx context.Context, verdict ReconcileVerdict) error {
	if runtime.state.ArmingState == ArmingHalted {
		return nil
	}
	if verdict.NextArmingState == ArmingArmed && runtime.state.ArmingState != ArmingArmed {
		return nil
	}
	runtime.state.ArmingState = verdict.NextArmingState
	return runtime.appendRiskEvent(ctx, "", "risk.state_changed", verdict.Reason)
}

func (runtime *Runtime) appendRiskEvent(ctx context.Context, symbol, kind, reason string) error {
	body, err := json.Marshal(JobRequest{Kind: kind, Subject: symbol, Body: reason})
	if err != nil {
		return err
	}
	_, err = runtime.store.AppendEvent(ctx, "trader", RiskEvent{
		EventIDValue: fmt.Sprintf("%s:%s:%d", kind, symbol, time.Now().UTC().UnixMilli()),
		SymbolValue:  symbol,
		Ts:           time.Now().UTC(),
		KindValue:    kind,
		Reason:       reason,
		ArmingState:  runtime.state.ArmingState,
	}, body)
	return err
}

type TradeClient struct {
	client *Client
	creds  PrivateCredentials
}

func NewTradeClient(exchangeCfg config.ExchangeConfig, key, secret, passphrase string) *TradeClient {
	return &TradeClient{
		client: NewClient(exchangeCfg.RESTBaseURL),
		creds:  PrivateCredentials{Key: key, Secret: secret, Passphrase: passphrase},
	}
}

func (client *TradeClient) PlaceOrder(ctx context.Context, req PlaceOrderRequest) error {
	// POST `/api/v2/mix/order/place-order` with signed private REST.
	return nil
}
```

- [ ] **Step 4: Wire `cmd/traderd` to create the real exchange client only when execution is allowed**

```go
func run(ctx context.Context, cfg config.Config) error {
	store, err := sqlitepkg.NewStore(cfg.Live.Runtime.StateDBPath)
	if err != nil {
		return err
	}
	runtimeCfg := trader.RuntimeConfig{
		Store:             store,
		ConsumerKey:       "traderd",
		InitialArmingState: trader.ArmingState(cfg.Live.Runtime.ArmingState),
		ObserveOnly:       cfg.Live.Runtime.ObserveOnly,
		Policy:            trader.PolicyConfig{MaxTranches: cfg.Live.Exchange.Symbols[0].MaxTranches},
		Risk:              trader.RiskConfig{MaxLeverage: cfg.Live.Risk.MaxLeverage},
	}
	if !cfg.Live.Runtime.ObserveOnly {
		runtimeCfg.Exchange = bitget.NewTradeClient(cfg.Live.Exchange, os.Getenv(cfg.Live.Exchange.APIKeyEnv), os.Getenv(cfg.Live.Exchange.APISecretEnv), os.Getenv(cfg.Live.Exchange.PassphraseEnv))
	}
	runtime, err := trader.NewRuntime(runtimeCfg)
	if err != nil {
		return err
	}
	return runtime.Run(ctx)
}
```

- [ ] **Step 5: Run package tests and build `traderd` again**

Run:
- `PATH=/usr/local/go/bin:$PATH go test ./internal/trader ./internal/exchange/bitget -v`
- `PATH=/usr/local/go/bin:$PATH go build ./cmd/traderd`

Expected: PASS.

- [ ] **Step 6: Run real Bitget order lifecycle verification on ECS**

Run:
- Start `marketd` and `traderd` on ECS with real Bitget credentials, `observe_only=false`, `arming_state=armed`, one symbol, and the smallest allowed `USDT-FUTURES` order size.
- Query the resulting runtime DB with `sqlite3 <state_db_path> "SELECT source, event_kind, count(*) FROM event_log WHERE source IN ('market.private','trader') GROUP BY source, event_kind ORDER BY source, event_kind;"`
- Query the exchange order state through the signed Bitget private REST path implemented in this task.

Expected:
- A real Bitget order is placed through `traderd`, can be queried back from Bitget, and yields matching `trader` plus `market.private` evidence in SQLite.
- The private stream shows `orders` updates and, when the order fills, an `order_fill` event. Position and account snapshots converge with the order result.
- If the order does not fill promptly, the task includes verifying cancel behavior. If the order opens a position, the task includes sending and verifying a reduce-only exit until the position returns flat.
- Any run that leaves unmatched exchange state, an open test position, or missing private stream evidence is a failed Task 5 verification.

- [ ] **Step 7: Commit the live execution and reconcile change**

```bash
git add internal/trader/runtime.go internal/trader/risk_engine.go internal/trader/execution_coordinator.go internal/trader/reconciler.go internal/trader/reconciler_test.go internal/trader/execution_coordinator_test.go internal/exchange/bitget/futures_trade.go internal/exchange/bitget/futures_account.go cmd/traderd/main.go
git commit -m "feat: add trader execution and reconciliation runtime"
```

## Task 6: Converge replay onto the same trader runtime and futures replay source

**Files:**
- Modify: `internal/adapters/data.go`
- Modify: `internal/replay/event_source.go`
- Modify: `internal/replay/harness.go`
- Modify: `internal/replay/harness_test.go`
- Modify: `cmd/lab/main.go`

- [ ] **Step 1: Write the failing replay parity tests**

```go
func TestBitgetEndpointUsesFuturesPathWhenProductTypePresent(t *testing.T) {
	got := bitgetEndpoint(config.DatasetConfig{
		Provider:    "bitget",
		Symbol:      "BTCUSDT",
		Interval:    "1m",
		Limit:       240,
		ProductType: "USDT-FUTURES",
	})
	if !strings.Contains(got, "/api/v2/mix/market/candles") {
		t.Fatalf("expected futures endpoint, got %s", got)
	}
}

func TestReplayHarnessUsesTraderRuntimeOutput(t *testing.T) {
	store, err := sqlite.NewStore(filepath.Join(t.TempDir(), "replay.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	runtime, err := trader.NewRuntime(trader.RuntimeConfig{
		Store:              store,
		ConsumerKey:        "replay",
		InitialArmingState: trader.ArmingSafe,
		ObserveOnly:        true,
		StrategyCfg:        config.StrategyConfig{FastSMA: 3, SlowSMA: 5, ATRWindow: 3, LevelLookback: 12, PivotWindow: 1},
		Strategy:           trader.LegacyRuleProfile{StrategyCfg: config.StrategyConfig{FastSMA: 3, SlowSMA: 5, ATRWindow: 3, LevelLookback: 12, PivotWindow: 1}},
		Policy:             trader.PolicyConfig{MaxTranches: 3},
	})
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}
	harness := NewHarness(runtime, NewLongOnlySimulator())
	report, err := harness.Run([]market.MarketEvent{
		market.BarClosedEvent{SymbolValue: "BTCUSDT", Interval: "1m", Ts: time.Unix(1710000000, 0), Close: 62000},
	})
	if err != nil {
		t.Fatalf("run replay: %v", err)
	}
	if report.CandidateCount == 0 {
		t.Fatalf("expected replay runtime to emit at least one candidate")
	}
}
```

- [ ] **Step 2: Run the replay-related tests to verify they fail**

Run: `PATH=/usr/local/go/bin:$PATH go test ./internal/adapters ./internal/replay -run 'TestBitgetEndpointUsesFuturesPathWhenProductTypePresent|TestReplayHarnessUsesTraderRuntimeOutput' -v`

Expected: FAIL because Bitget still uses the spot endpoint and replay still drives `LegacyRuleProfile` directly.

- [ ] **Step 3: Switch Bitget replay fetches to futures and reuse trader runtime from replay**

```go
func bitgetEndpoint(spec config.DatasetConfig) string {
	limit := spec.Limit
	if limit <= 0 {
		limit = 1000
	}
	basePath := "/api/v2/spot/market/candles"
	if strings.EqualFold(spec.ProductType, "USDT-FUTURES") {
		basePath = "/api/v2/mix/market/candles"
	}
	return fmt.Sprintf("https://api.bitget.com%s?symbol=%s&granularity=%s&productType=%s&limit=%d", basePath, url.QueryEscape(strings.ToUpper(spec.Symbol)), url.QueryEscape(bitgetGranularity(spec.Interval)), url.QueryEscape(spec.ProductType), limit)
}

func (client *Client) fetchBitget(ctx context.Context, spec config.DatasetConfig) ([]core.Bar, error) {
	endpoint := bitgetEndpoint(spec)
	// Decode as before.
}
```

```go
type RuntimeHarness struct {
	runtime   *trader.Runtime
	simulator Simulator
}

func (runtime *trader.Runtime) Inject(ctx context.Context, evt market.MarketEvent) error {
	body, err := json.Marshal(evt)
	if err != nil {
		return err
	}
	if _, err := runtime.Store().AppendEvent(ctx, "replay", evt, body); err != nil {
		return err
	}
	return runtime.ProcessAvailable(ctx)
}

func (runtime *trader.Runtime) Report() Report {
	// Read the replay sink summary and candidate counters accumulated during injection.
}

func (h *Harness) Run(events []market.MarketEvent) (Report, error) {
	for _, evt := range events {
		if err := h.runtime.Inject(context.Background(), evt); err != nil {
			return Report{}, err
		}
	}
	return h.runtime.Report(), nil
}
```

```go
func runReplay(args []string) error {
	// Load config and events as before.
	store, err := sqlitepkg.NewStore(filepath.Join(cfg.ArtifactDir, "replay-state.db"))
	if err != nil {
		return err
	}
	policy := trader.PolicyConfig{MaxTranches: 1}
	if len(cfg.Live.Exchange.Symbols) > 0 {
		policy.MaxTranches = cfg.Live.Exchange.Symbols[0].MaxTranches
	}
	runtime, err := trader.NewRuntime(trader.RuntimeConfig{
		Store:              store,
		ConsumerKey:        "replay",
		InitialArmingState: trader.ArmingSafe,
		ObserveOnly:        true,
		StrategyCfg:        cfg.Strategy,
		Strategy:           trader.LegacyRuleProfile{StrategyCfg: cfg.Strategy},
		Policy:             policy,
	})
	if err != nil {
		return err
	}
	harness := replay.NewHarness(runtime, replay.NewLongOnlySimulator())
	report, err := harness.RunAndWrite(events, cfg.ArtifactDir)
	if err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(report)
}
```

- [ ] **Step 4: Run replay verification**

Run:
- `PATH=/usr/local/go/bin:$PATH go test ./internal/adapters ./internal/replay -v`
- `PATH=/usr/local/go/bin:$PATH go run ./cmd/lab replay -config configs/demo-bitget.yaml`

Expected: PASS. `artifacts/demo-bitget/replay-report.json` should now come from the same runtime used by `traderd`.

- [ ] **Step 5: Commit replay/live convergence**

```bash
git add internal/adapters/data.go internal/replay/event_source.go internal/replay/harness.go internal/replay/harness_test.go cmd/lab/main.go
git commit -m "feat: converge replay onto trader runtime"
```

## Task 7: Turn `agentd` into a read-only consumer of trader advisory and risk events

**Files:**
- Create: `internal/agent/runtime.go`
- Create: `internal/agent/runtime_test.go`
- Modify: `internal/agent/service.go`
- Modify: `internal/agent/jobs.go`
- Modify: `internal/agent/mcp.go`
- Modify: `internal/agent/service_test.go`
- Modify: `cmd/agentd/main.go`

- [ ] **Step 1: Write the failing agent runtime tests**

```go
func TestRuntimeReviewsCandidateEvents(t *testing.T) {
	store := newTestStore(t)
	client := &fakeResponsesClient{createResponse: CreateResponse{ID: "resp-1", Status: "completed"}}
	service := NewService(client, Config{Model: "gpt-5.4", Store: true})
	runtime := NewRuntime(RuntimeConfig{Store: store, ConsumerKey: "agentd", Service: service})

	_, err := store.AppendEvent(context.Background(), "trader", trader.CandidateEvent{
		EventIDValue: "cand-1",
		SymbolValue:  "BTCUSDT",
		Ts:           time.Unix(1710000000, 0),
		KindValue:    "candidate.created",
		Profile:      "needle_capture",
		Candidate:    trader.Candidate{Symbol: "BTCUSDT", Score: 4.2},
	}, []byte(`{"symbol":"BTCUSDT","suggested_action":"probe_long","max_leverage":3,"reasons":["needle_capture"]}`))
	if err != nil {
		t.Fatalf("append trader event: %v", err)
	}
	if err := runtime.ProcessAvailable(context.Background()); err != nil {
		t.Fatalf("process available: %v", err)
	}
	if client.createCalls != 1 {
		t.Fatalf("expected one review request, got %d", client.createCalls)
	}
}

func TestRuntimeExplainsRiskStateChanges(t *testing.T) {
	store := newTestStore(t)
	client := &fakeResponsesClient{createResponse: CreateResponse{ID: "resp-risk", Status: "completed"}}
	service := NewService(client, Config{Model: "gpt-5.4", Store: true})
	runtime := NewRuntime(RuntimeConfig{Store: store, ConsumerKey: "agentd", Service: service})

	_, err := store.AppendEvent(context.Background(), "trader", trader.RiskEvent{
		EventIDValue: "risk-1",
		SymbolValue:  "BTCUSDT",
		Ts:           time.Unix(1710000060, 0),
		KindValue:    "risk.state_changed",
		Reason:       "position_mismatch",
		ArmingState:  trader.ArmingDegraded,
	}, []byte(`{"kind":"risk.state_changed","subject":"BTCUSDT","body":"Explain degraded state caused by position mismatch."}`))
	if err != nil {
		t.Fatalf("append trader event: %v", err)
	}
	if err := runtime.ProcessAvailable(context.Background()); err != nil {
		t.Fatalf("process available: %v", err)
	}
	if client.createCalls != 1 {
		t.Fatalf("expected one risk explanation request, got %d", client.createCalls)
	}
}
```

- [ ] **Step 2: Run the agent tests to verify they fail**

Run: `PATH=/usr/local/go/bin:$PATH go test ./internal/agent -run 'TestRuntimeReviewsCandidateEvents|TestRuntimeExplainsRiskStateChanges' -v`

Expected: FAIL because `agentd` does not tail the event log yet.

- [ ] **Step 3: Implement the advisory runtime**

```go
type Runtime struct {
	store       *sqlite.Store
	service     *Service
	consumerKey string
	cursor      int64
}

type RuntimeConfig struct {
	Store       *sqlite.Store
	Service     *Service
	ConsumerKey string
}

func NewRuntime(cfg RuntimeConfig) *Runtime {
	return &Runtime{store: cfg.Store, service: cfg.Service, consumerKey: cfg.ConsumerKey}
}

func (runtime *Runtime) ProcessAvailable(ctx context.Context) error {
	envelopes, err := runtime.store.ListEventsAfter(ctx, runtime.cursor, 256, "trader")
	if err != nil {
		return err
	}
	for _, env := range envelopes {
		switch env.Kind {
		case "candidate.created":
			var pkt CandidatePacket
			if err := json.Unmarshal(env.Payload, &pkt); err != nil {
				return err
			}
			if _, err := runtime.service.ReviewCandidate(ctx, pkt); err != nil {
				return err
			}
		case "risk.state_changed":
			var job JobRequest
			if err := json.Unmarshal(env.Payload, &job); err != nil {
				return err
			}
			if _, err := runtime.service.StartBackgroundJob(ctx, job); err != nil {
				return err
			}
		}
		runtime.cursor = env.Seq
		if err := runtime.store.SaveConsumerCursor(ctx, runtime.consumerKey, runtime.cursor); err != nil {
			return err
		}
	}
	return nil
}

func (runtime *Runtime) Run(ctx context.Context) error {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		if err := runtime.ProcessAvailable(ctx); err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
```

```go
func run(ctx context.Context, cfg config.Config) error {
	if !cfg.Live.Agent.Enabled {
		<-ctx.Done()
		return ctx.Err()
	}
	service, err := newService(cfg)
	if err != nil {
		return err
	}
	store, err := sqlitepkg.NewStore(cfg.Live.Runtime.StateDBPath)
	if err != nil {
		return err
	}
	runtime := agent.NewRuntime(agent.RuntimeConfig{Store: store, ConsumerKey: "agentd", Service: service})
	return runtime.Run(ctx)
}
```

- [ ] **Step 4: Run the agent package tests and build `agentd`**

Run:
- `PATH=/usr/local/go/bin:$PATH go test ./internal/agent -v`
- `PATH=/usr/local/go/bin:$PATH go build ./cmd/agentd`

Expected: PASS. The old advisory-only prompt tests must still pass.

- [ ] **Step 5: Commit the advisory runtime**

```bash
git add internal/agent/runtime.go internal/agent/runtime_test.go internal/agent/service.go internal/agent/jobs.go internal/agent/mcp.go internal/agent/service_test.go cmd/agentd/main.go
git commit -m "feat: add advisory event consumer for agentd"
```

## Task 8: Update verification flow and rollout gates for the real runtime

**Files:**
- Modify: `scripts/measure.sh`
- Modify: `configs/demo-bitget.yaml`
- Modify: `configs/live-bitget.yaml`
- Modify: `plan/rollout-checklist.md`

- [ ] **Step 1: Run the current demo measure command and record the undesired behavior**

Run: `PATH=/usr/local/go/bin:$PATH SMOKE_SECONDS=3 ./scripts/measure.sh configs/demo-bitget.yaml`

Expected: This should become too strong after Tasks 3-7 because `marketd` and `traderd` will require real event sources, so the script needs an explicit live-smoke gate instead of always starting daemons.

- [ ] **Step 2: Add a dedicated `RUN_LIVE_SMOKE` gate to `measure.sh`**

```bash
RUN_LIVE_SMOKE=${RUN_LIVE_SMOKE:-0}

if [[ "$is_live_runtime" == "1" ]]; then
	"$LAB_BIN" replay -config "$CHECK_CONFIG_PATH" > "$REPLAY_JSON"
	if [[ "$RUN_LIVE_SMOKE" == "1" ]]; then
		run_smoke marketd "$MARKETD_BIN" "$CHECK_CONFIG_PATH" "$MARKETD_LOG"
		run_smoke traderd "$TRADERD_BIN" "$CHECK_CONFIG_PATH" "$TRADERD_LOG"
	fi
else
	"$LAB_BIN" replay -config "$BASELINE_CONFIG_PATH" > "$REPLAY_JSON"
	run_smoke stream "$STREAM_BIN" "$BASELINE_CONFIG_PATH" "$STREAM_LOG"
fi
```

- [ ] **Step 3: Tighten config and rollout docs around manual arm and observe-only**

```yaml
live:
  runtime:
    arming_state: safe
    observe_only: true
  agent:
    enabled: false
    advisory_only: true
```

```md
- `RUN_LIVE_SMOKE=0` keeps `measure.sh` CI-safe and replay-only for demo/live configs.
- `RUN_LIVE_SMOKE=1` is required before promotion and assumes valid Bitget credentials plus reachable websocket endpoints.
- Real Bitget verification is a separate completion gate: public and private reads, private streams, order query, and real order lifecycle all have to pass on ECS before rollout can advance.
- `BITGET_API_KEY`, `BITGET_API_SECRET`, and `BITGET_PASSPHRASE` remain env-only inputs. Config files keep only env var names.
- Promotion from `safe` to `armed` remains a config-and-restart action after reconciliation passes; the runtime may downgrade itself but never auto-promote itself.
```

- [ ] **Step 4: Run the full verification matrix**

Run:
- `PATH=/usr/local/go/bin:$PATH go test ./...`
- `PATH=/usr/local/go/bin:$PATH go build ./cmd/lab ./cmd/marketd ./cmd/traderd ./cmd/agentd`
- `PATH=/usr/local/go/bin:$PATH ./scripts/measure.sh configs/baseline.yaml`
- `PATH=/usr/local/go/bin:$PATH SMOKE_SECONDS=3 ./scripts/measure.sh configs/demo-bitget.yaml`
- `PATH=/usr/local/go/bin:$PATH RUN_BITGET_REAL=1 BITGET_API_KEY=$BITGET_API_KEY BITGET_API_SECRET=$BITGET_API_SECRET BITGET_PASSPHRASE=$BITGET_PASSPHRASE go test ./internal/exchange/bitget -run RealBitget -v`
- `PATH=/usr/local/go/bin:$PATH RUN_BITGET_REAL=1 BITGET_API_KEY=$BITGET_API_KEY BITGET_API_SECRET=$BITGET_API_SECRET BITGET_PASSPHRASE=$BITGET_PASSPHRASE go test ./internal/market -run RealBitget -v`
- `PATH=/usr/local/go/bin:$PATH RUN_LIVE_SMOKE=1 SMOKE_SECONDS=15 ./scripts/measure.sh configs/demo-bitget.yaml`

Expected:
- First four commands PASS without requiring live exchange credentials.
- The two `RUN_BITGET_REAL=1` commands PASS only when ECS has valid real Bitget credentials and the implementation can talk to the actual Bitget public/private surfaces.
- The last command PASSes only in a real Bitget environment with valid credentials, reachable websocket endpoints, and no leftover manual cleanup.
- Missing `BITGET_PASSPHRASE` means the private read/write checks cannot be counted as complete even if the public checks pass.

- [ ] **Step 5: Commit the verification and rollout update**

```bash
git add scripts/measure.sh configs/demo-bitget.yaml configs/live-bitget.yaml plan/rollout-checklist.md
git commit -m "docs: tighten runtime verification and rollout gates"
```

## Verification Matrix

Prerequisites:

- Export real Bitget credentials on ECS as environment variables and map config env names to them: `BITGET_API_KEY`, `BITGET_API_SECRET`, `BITGET_PASSPHRASE`.
- Do not write those secret values into tracked YAML, tracked Markdown, or committed helper scripts.

Run these after all tasks are complete:

```bash
PATH=/usr/local/go/bin:/usr/bin:/bin go test ./...
PATH=/usr/local/go/bin:/usr/bin:/bin go build ./cmd/lab ./cmd/marketd ./cmd/traderd ./cmd/agentd
PATH=/usr/local/go/bin:/usr/bin:/bin ./scripts/measure.sh configs/baseline.yaml
PATH=/usr/local/go/bin:/usr/bin:/bin SMOKE_SECONDS=3 ./scripts/measure.sh configs/demo-bitget.yaml
PATH=/usr/local/go/bin:/usr/bin:/bin RUN_BITGET_REAL=1 BITGET_API_KEY=$BITGET_API_KEY BITGET_API_SECRET=$BITGET_API_SECRET BITGET_PASSPHRASE=$BITGET_PASSPHRASE go test ./internal/exchange/bitget -run RealBitget -v
PATH=/usr/local/go/bin:/usr/bin:/bin RUN_BITGET_REAL=1 BITGET_API_KEY=$BITGET_API_KEY BITGET_API_SECRET=$BITGET_API_SECRET BITGET_PASSPHRASE=$BITGET_PASSPHRASE go test ./internal/market -run RealBitget -v
PATH=/usr/local/go/bin:/usr/bin:/bin RUN_LIVE_SMOKE=1 SMOKE_SECONDS=15 ./scripts/measure.sh configs/demo-bitget.yaml
PATH=/usr/local/go/bin:/usr/bin:/bin timeout --preserve-status --signal=INT --kill-after=2s 900s ./marketd -config configs/demo-bitget.yaml
PATH=/usr/local/go/bin:/usr/bin:/bin timeout --preserve-status --signal=INT --kill-after=2s 900s ./traderd -config configs/demo-bitget.yaml
PATH=/usr/local/go/bin:/usr/bin:/bin timeout --preserve-status --signal=INT --kill-after=2s 300s ./agentd -config configs/demo-bitget.yaml
```

Success criteria:

- `marketd` writes ordered `market.bootstrap`, `market.public`, and `market.private` rows into `event_log`.
- Real Bitget public verification proves futures market reads plus WebSocket `trade_tick` and `bar_closed` ingestion work against the actual exchange.
- Real Bitget private verification proves account, positions, and order-query calls succeed and private WebSocket events land in SQLite.
- At least one real order lifecycle is verified end to end: place order, query order state, observe `orders` or `order_fill`, and if exposure is created, flatten it with a verified reduce-only exit.
- `traderd` restores checkpoint and cursor, emits `candidate.created` and `risk.state_changed` rows under source `trader`, and never auto-promotes arming state.
- `lab replay` writes `artifacts/demo-bitget/replay-report.json` using the same trader runtime semantics as `traderd`.
- `agentd` tails only `trader` events, calls the Responses API in advisory-only mode, and never touches exchange credentials.
- `measure.sh` remains safe for CI by default and requires `RUN_LIVE_SMOKE=1` for real daemon smoke.
- `mock pass != done`: completion requires both the package-level real Bitget checks and the daemon-level real Bitget smoke to pass on ECS.

## Spec Coverage Check

- Design item `marketd real Bitget futures public/private source + SQLite event log` is implemented by Tasks 1, 2, and 3.
- Design item `traderd real consumer + checkpoint + router + policy + risk + reconciliation + execution` is implemented by Tasks 4 and 5.
- Design item `replay/live convergence onto one deterministic trader runtime` is implemented by Task 6.
- Design item `agentd advisory-only event consumer` is implemented by Task 7.
- Design item `real rollout gates and demo/live promotion checks` is implemented by Task 8.

## Plan Self-Review

- No dual-write, migration guard, or second state store was introduced. SQLite/WAL remains the only runtime persistence layer.
- `Engine` is intentionally simplified into a deterministic candidate generator; exchange side effects move to `Runtime`, which removes the current mixed-responsibility design in `internal/trader/engine.go`.
- Manual arm stays explicit and operator-driven; automatic transitions are downward only. This keeps the first-phase control surface small and auditable.
- `agentd` remains read-only. The plan never routes exchange credentials or order placement through the AI plane.
- Exchange-facing tasks now have explicit dual verification layers. Fake-server and unit coverage speed up development, but real Bitget verification on ECS is the hard completion gate.
- Secret handling stays out of git. The plan requires env-only credential injection and treats missing `BITGET_PASSPHRASE` as an incomplete private/order verification state rather than something to hand-wave past.
