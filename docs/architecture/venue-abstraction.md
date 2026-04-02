# Venue Abstraction

## Purpose

本文固定 Phase 1 的统一 venue 语义，避免后续 Bitget / Hyperliquid / leader-follow / experiments 各自发明一套字段名和 symbol 口径。

## 当前范围

Phase 1 当前只完成三件事：

- 定义统一 `venue/common` 对象模型
- 把 Bitget 与 Hyperliquid 的只读 payload 适配到统一模型
- 把当前 live runtime 事件与 execution intent 补上 `venue` / `market_type` / `execution_venue`

Phase 1 当前没有完成：

- Hyperliquid runtime recorder
- leader_follow bundle
- 第二条真实交易所写路径
- warehouse schema 迁移

## Unified Objects

当前统一对象固定落在 `internal/venue/common`：

- `Instrument`
- `Candle`
- `PriceSnapshot`
- `FundingRate`
- `OpenInterest`
- `ContractSpec`
- `LeaderFill`
- `LeaderStateSnapshot`
- `OrderUpdate`

这些对象统一共享下面这组语义：

- `venue`
  - 数据来自哪个交易所或 venue
- `market_type`
  - 统一归一为 `spot` / `perp`
- `canonical_symbol`
  - 平台内部主符号，例如 `BTC`
- `venue_symbol`
  - venue 自己的原始符号，例如 Bitget `BTCUSDT`、Hyperliquid `BTC`
- `timestamp`
  - 统一用 UTC 时间语义

## Instrument Mapper

`internal/venue/common/mapper` 负责把 venue-specific 标识映射成 canonical instrument。

当前 fixture 固定覆盖：

- Bitget futures core symbols
- Hyperliquid perp core symbols
- spot/perp 两类 market type

Mapper 的职责只有两件事：

- 统一符号口径
- 拒绝未知 mapping

Mapper 当前不负责：

- 自动发现新合约
- 动态刷新远端 symbol registry
- fallback 猜测

这一步故意保持静态，是为了先把语义钉死，而不是提前引入漂移源。

## Adapter Boundaries

### Bitget

`internal/venue/bitget` 当前适配：

- candles
- ticker / price snapshot
- funding / open interest
- contract spec
- private fill / order / position 读面

当前 live producer 仍然直接产出 `market.*` 事件，但这些事件现在会带：

- `venue=bitget`
- `market_type=perp` 或 `spot`

### Hyperliquid

`internal/venue/hyperliquid` 当前适配：

- `candleSnapshot`
- `userFillsByTime`
- `clearinghouseState`
- `orderStatus`

当前这层仍是只读 adapter，不是 runtime recorder。

## Runtime Impact

Phase 1 对当前 live runtime 的影响刻意收窄在两个点：

### `marketd`

- `trade_tick / bar_closed / micro_bar_closed`
- `position_snapshot / position_update / order_update / order_fill / account_snapshot`

这些 payload 现在都显式带 `venue` 与 `market_type`。

这里没有改：

- SQLite event log schema
- warehouse ingest table shape
- current Bitget marketd wiring

原因是当前 event log 本来就是 JSON payload append-only，补字段不会要求 schema migration。

### `traderd -> execd`

`entry.intent.created` 与 `execution.reconciled` 现在显式带 `execution_venue`。

当前约束：

- 默认 `execution_venue=bitget`
- `execd` 当前只接受 `bitget`
- 非 `bitget` intent 会被 runtime guard 拒绝

这一步只是在事件契约上承认“未来 execution venue 会分裂”，并没有提前开放第二条交易写路径。

## Compatibility Rule

Phase 1 的硬约束是：

- 当前 Bitget 主线零行为回归
- `marketd / traderd / execd / platformd / mcpd` build 仍然通过
- `go test ./...` 仍然通过
- Bitget real read / real execution safe line 仍然通过

如果后续 phase 想引入：

- Hyperliquid recorder
- warehouse venue-first schema
- real dual-venue execution

都必须建立在本文件已经固定的语义之上，而不是重写字段含义。

## Real Verification On 2026-04-02

- `go test ./internal/venue/common ./internal/venue/bitget ./internal/venue/hyperliquid ./internal/exchange/bitget ./internal/trader ./internal/execution ./internal/market -count=1`
- `go test ./cmd/marketd ./internal/warehouse/ingest -count=1`
- `go test ./... -count=1`
- `go build ./cmd/lab ./cmd/marketd ./cmd/traderd ./cmd/agentd ./cmd/platformd ./cmd/mcpd`
- `RUN_BITGET_REAL=1 go test ./internal/exchange/bitget -run TestRealBitgetPrivateReadSurface -count=1 -v`
- `RUN_BITGET_REAL=1 go test ./internal/execution -run TestRealExecutionRuntimeIntentLifecycle -count=1 -v`
- Hyperliquid public read artifact:
  - `artifacts/hyperliquid-phase1/20260402T142345Z/candleSnapshot_BTC_15m.json`
  - `artifacts/hyperliquid-phase1/20260402T142345Z/metaAndAssetCtxs.json`
  - `artifacts/hyperliquid-phase1/20260402T142345Z/ws_candle_BTC_15m.json`
