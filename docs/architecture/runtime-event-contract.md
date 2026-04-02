# Runtime Event Contract

## Purpose

本文固定 `marketd -> traderd -> execd -> notifierd` 的运行时事件边界，避免 AI worker 或后续代码把交易所写路径重新塞回 `traderd`。

## Source Of Truth

- exchange ingress 只能由 `marketd` 写入 `event_log`
- deterministic evaluation 只能由 `traderd` 产出
- Bitget write side effect 只能由 `execd` 执行
- operator notification 由 `notifierd` 消费事件后外发

## Event Flow

### `marketd`

写入 source:

- `market.bootstrap`
- `market.public`
- `market.private`

核心 kind:

- `bar_closed`
- `trade_tick`
- `position_snapshot`
- `position_update`
- `order_update`
- `order_fill`
- `account_snapshot`

当前 Phase 1 统一约束：

- `market.public`
  - `trade_tick / bar_closed / micro_bar_closed` payload 显式携带 `venue` 与 `market_type`
- `market.private`
  - `position_snapshot / position_update / order_update / order_fill / account_snapshot` payload 同样显式携带 `venue` 与 `market_type`
- 当前 live producer 仍然只有 `bitget`
  - 因此真实值当前固定表现为 `venue=bitget`
  - `market_type` 依据 `productType / instType` 归一后表现为 `perp` 或 `spot`

### `traderd`

消费 source:

- `market.bootstrap`
- `market.public`
- `market.private`

写入 source:

- `trader`

核心 kind:

- `candidate.created`
  deterministic signal output，只表达候选交易，不触发写单
- `entry.intent.created`
  交给 `execd` 的执行意图，当前 payload 固定包含：
  `symbol / interval / ts / side / score / entry / stop / target / execution_venue / product_type / margin_mode / margin_coin / size / leverage / client_oid`
- `risk.state_changed`
  reconciliation 或风控降级事件

明确禁止：

- `traderd` 不得直接调用 `SetLeverage`
- `traderd` 不得直接调用 `PlaceOrder`
- `traderd` 不得直接调用 `GetOrderDetail`
- `traderd` 不得直接做 flatten / close write

### `execd`

消费 source:

- `trader`

只处理 kind:

- `entry.intent.created`

写入 source:

- `execution`

核心 kind:

- `execution.reconciled`
  记录 `intent_event_id / client_oid / order_id / status / size / price_avg / reduce_only`

Bitget write surface:

- `SetLeverage`
- `PlaceOrder`
- `GetOrderDetail`
- `FetchSinglePosition`

flatten 规则:

- one-way mode: `reduceOnly=YES`
- hedge mode: `tradeSide=close`

Phase 1 execution venue 约束：

- 当前 `execution_venue` 已进入 `entry.intent.created` 与 `execution.reconciled`
- 当前 formal runtime 只接受 `bitget`
- 非 `bitget` intent 会在进入 `SetLeverage / PlaceOrder` 之前被直接拒绝
- 这一步的目标是先把 execution venue 语义钉进事件契约，而不是提前引入第二条真实交易所写路径

### `notifierd`

消费 source:

- `market.private`
- `trader`
- `platform`

当前外发关注：

- `order_fill`
- `risk.state_changed`
- `promotion.approved`
- `promotion.canary_degraded`

## Real Verification On 2026-03-29

- `PATH=/usr/local/go/bin:/usr/bin:/bin go build ./cmd/traderd ./cmd/execd` 通过
- `PATH=/usr/local/go/bin:/usr/bin:/bin go test ./internal/trader ./internal/execution ./internal/exchange/bitget -v` 通过
- `PATH=/usr/local/go/bin:/usr/bin:/bin go test ./...` 通过
- `rg -n "PlaceOrder|SubmitOrder|ReduceOnlyClose|QueryOrder|SetLeverage" internal/trader cmd/traderd` 返回空
- `RUN_BITGET_REAL=1 ... go test ./internal/execution -run TestRealExecutionRuntimeIntentLifecycle -v` 真实通过
  - 实测 `MSTRUSDT` `last_price=126.2200`
  - 实测最小有效 size `0.04`
  - 真实开仓、查单、成交、flatten、最终空仓全部成功
