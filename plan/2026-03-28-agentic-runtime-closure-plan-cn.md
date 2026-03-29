# Agentic Runtime Closure 实施计划（中文审查版）

> 面向审查的中文主文档。本文对应英文原稿 `2026-03-28-agentic-runtime-closure-plan.md`，保留同一套目标、任务拆解、验收标准与 rollout gate。代码路径、命令、环境变量、接口名保持英文原样，避免术语漂移。

## 文档身份

| 项目 | 内容 |
| --- | --- |
| 中文审查稿路径 | `plan/2026-03-28-agentic-runtime-closure-plan-cn.md` |
| 英文原稿路径 | `plan/2026-03-28-agentic-runtime-closure-plan.md` |
| ECS worktree | `/root/.config/superpowers/worktrees/quant-lab/autoresearch-20260328-all-plan` |
| 本机镜像目录 | `/Users/qiukeming/Documents/projects/ob/obsidian/ecs/plan` |
| 英文原稿行数 | `1676` |
| 英文原稿提交 | `3be70798bfef1489dce62f59fe60d8989a39f828` |

**目标：** 在现有 scaffold 基础上补齐一套可验证的 Bitget `USDT-FUTURES` 运行时：`marketd` 把真实 futures public/private 事件写入 SQLite/WAL，`traderd` 消费事件日志并执行 deterministic 规则与 execution gate，`lab replay` 与 live 共用同一 trader runtime，`agentd` 仅消费 advisory 与 risk 事件且不获得执行权限。所有 exchange-facing 工作只有在 ECS 上通过真实 Bitget 验证后才算完成，mock 或 fake server 通过不构成完成。

**架构：** 保持现有 modular monolith 代码形态和四进程运行时拆分。用通用顺序 `event_log` 与 `consumer_cursor` 取代当前 ad hoc `market_event_log`；`marketd` 是唯一 exchange ingress writer，`traderd` 是唯一 deterministic decision/execution writer，`agentd` 只消费 `trader` 侧 advisory/risk 事件；`lab replay` 不再单独走 `LegacyRuleProfile`，而是复用同一套 trader runtime 语义。

**技术栈：** Go、SQLite/WAL、Bitget futures REST/WebSocket、Gorilla WebSocket、YAML、OpenAI Responses API、现有 `quantlab/internal/core` 兼容路径。

---

## 当前代码事实

- `cmd/marketd/main.go` 虽然打开了 SQLite store，但传给 `market.NewPublicFeed` 的 source 仍是 `nil`，当前没有真实 market source 接入。
- `cmd/traderd/main.go` 只初始化 store 与 `trader.NewEngine`，没有真正消费事件。
- `cmd/agentd/main.go` 当前只检查 `live.agent.advisory_only` 与 `OPENAI_API_KEY`，随后阻塞。
- `cmd/lab/main.go` 的 replay 仍然直接构造 `LegacyRuleProfile{StrategyCfg: cfg.Strategy}`，没有与 live runtime 收敛。
- `internal/exchange/bitget/public_ws.go` 只解 `trade`，`ticker` 与 candle 仍返回 `nil`。
- `internal/exchange/bitget/private_ws.go` 只解 `orders`，`positions` 与 `account` 仍返回 `nil`。
- `internal/adapters/data.go` 仍然从 `/api/v2/spot/market/candles` 抓 Bitget K 线，这与 futures replay parity 不一致。
- `internal/store/sqlite/schema.sql` 还没有单调递增事件序列，也没有 consumer cursor 表。

## 方案选择

1. 推荐方案：继续保持 `marketd`、`traderd`、`agentd`、`lab` 四个 runtime，统一落到一个顺序化 SQLite/WAL `event_log`，并让 live/replay 复用同一 deterministic trader runtime。
2. 不采用方案：把所有逻辑收回单进程。这个方案虽然短期更快，但会破坏重启隔离，并违背已经批准的运行时拆分。
3. 不采用方案：在 I/O 还没闭环前先整块重写 trader kernel。当前代码里已有 `StrategyRouter`、`PositionPolicy`、`RiskEngine`、`Reconciler`、`ExecutionCoordinator` 等可复用组件，先闭环再收敛更小、更稳。

## 真实 Bitget 验证合同

- 所有 exchange-facing 任务都必须同时满足两层验证：本地 package test 或 fake server 用于快速迭代，ECS 上的真实 Bitget 验证用于完成判定。
- 真实 Bitget 验证必须覆盖：
  - public REST：market metadata、candles、bootstrap read
  - public WebSocket：`trade_tick`、`bar_closed`
  - private REST：account、positions、order query
  - private WebSocket：orders、positions、account
  - write path：真实下单、真实查单、必要时撤单、若形成持仓则 reduce-only 平仓
- 凭证只允许通过 ECS 环境变量注入：`BITGET_API_KEY`、`BITGET_API_SECRET`、`BITGET_PASSPHRASE`。`live.exchange.api_key_env`、`api_secret_env`、`passphrase_env` 只存环境变量名，不存敏感值。
- `BITGET_PASSPHRASE` 是 private REST、private WebSocket、下单、查单、撤单、reduce-only exit 的硬前置条件。没有它，任何 private/order 任务都不能宣称完成。
- `go test ./...`、replay、fake-server coverage 只是开发信号，不是完成信号。`mock pass != done`，真实 Bitget 验证通过才算 done。

## 前提与约束

- 不假设存在已经发布的生产 runtime state，因此 SQLite schema 可以直接重塑，不做 dual write、不做 migration guard。
- `arming_state` 在 phase 1 继续保持人工设置加重启生效。runtime 允许自动下调到 `degraded` 或 `halted`，但禁止自动回升到 `armed`。
- `agentd` 全程保持 advisory-only。它可以读取 `traderd` 事件、写 explanatory artifact，但不能持有 exchange credential，也不能拥有 order route。
- 真实验证都必须在 ECS 上执行。public 验证可以先只依赖公共接口，private/account/order 验证必须依赖 ECS 环境中的 key、secret、passphrase。
- 真实下单验证使用 Bitget `USDT-FUTURES` 允许的最小下单规模、isolated margin，并强制包含 reduce-only 清仓收尾。若验证过程留下未关闭测试持仓，视为验证失败。

## 文件结构总览

### 需要修改的既有文件

- `internal/store/sqlite/schema.sql`
- `internal/store/sqlite/live_store.go`
- `internal/store/sqlite/live_store_test.go`
- `internal/market/event.go`
- `internal/market/public_feed.go`
- `internal/market/private_feed.go`
- `internal/exchange/bitget/rest_client.go`
- `internal/exchange/bitget/models.go`
- `internal/exchange/bitget/futures_market.go`
- `internal/exchange/bitget/futures_trade.go`
- `internal/exchange/bitget/public_ws.go`
- `internal/exchange/bitget/private_ws.go`
- `internal/exchange/bitget/public_ws_test.go`
- `internal/exchange/bitget/private_ws_test.go`
- `internal/adapters/data.go`
- `internal/trader/state.go`
- `internal/trader/engine.go`
- `internal/trader/engine_test.go`
- `internal/trader/strategy_router.go`
- `internal/trader/position_policy.go`
- `internal/trader/risk_engine.go`
- `internal/trader/execution_coordinator.go`
- `internal/trader/reconciler.go`
- `internal/trader/reconciler_test.go`
- `internal/trader/execution_coordinator_test.go`
- `internal/replay/event_source.go`
- `internal/replay/harness.go`
- `internal/replay/harness_test.go`
- `internal/agent/service.go`
- `internal/agent/jobs.go`
- `internal/agent/mcp.go`
- `internal/agent/service_test.go`
- `cmd/lab/main.go`
- `cmd/marketd/main.go`
- `cmd/traderd/main.go`
- `cmd/agentd/main.go`
- `scripts/measure.sh`
- `configs/demo-bitget.yaml`
- `configs/live-bitget.yaml`
- `plan/rollout-checklist.md`

### 需要新增的文件

- `internal/exchange/bitget/futures_account.go`
- `internal/exchange/bitget/futures_account_test.go`
- `internal/exchange/bitget/ws_source.go`
- `internal/exchange/bitget/ws_source_test.go`
- `internal/market/gap_fill.go`
- `internal/market/gap_fill_test.go`
- `internal/market/runtime.go`
- `internal/market/runtime_test.go`
- `internal/trader/runtime.go`
- `internal/trader/runtime_test.go`
- `internal/trader/runtime_events.go`
- `internal/trader/runtime_events_test.go`
- `internal/agent/runtime.go`
- `internal/agent/runtime_test.go`

## Task 1：把 `market_event_log` 替换成通用顺序事件日志与 consumer cursor

**涉及文件：**
- `internal/store/sqlite/schema.sql`
- `internal/store/sqlite/live_store.go`
- `internal/store/sqlite/live_store_test.go`

**执行步骤：**
1. 编写失败的 storage tests，覆盖单调递增 `seq`、`ListEventsAfter`、`SaveConsumerCursor/LoadConsumerCursor`。
2. 运行 `go test ./internal/store/sqlite -run 'TestAppendEventReturnsMonotonicSeqAndListEventsAfter|TestConsumerCursorRoundTrip' -v`，确认现有 API 无法满足 sequence/cursor 合同。
3. 实现 `event_log` 表与 `consumer_cursor` 表；补齐 `AppendEvent`、`ListEventsAfter`、`SaveConsumerCursor`、`LoadConsumerCursor`。
4. 运行 `go test ./internal/store/sqlite -v`，要求整个 sqlite package 通过。
5. 提交存储层改动。

**核心实现点：**
- 事件日志必须使用单调递增 `seq` 作为统一消费游标，而不是依赖 event id。
- 事件 envelope 要保留 `source`、`event_id`、`symbol`、`event_kind`、`exchange_ts`、`received_ts`、`payload_json`。
- consumer cursor 必须与 checkpoint 解耦；cursor 管消费位置，checkpoint 管 engine state。

**完成判据：**
- `AppendEvent` 返回 `seq`。
- `ListEventsAfter` 支持按 `seq` 与可选 `source` 查询。
- `consumer_cursor` round-trip 通过。
- commit message：`feat: add sequenced runtime event log`

## Task 2：补齐 futures 事件模型与 Bitget futures decode / client surface

**涉及文件：**
- `internal/market/event.go`
- `internal/exchange/bitget/rest_client.go`
- `internal/exchange/bitget/models.go`
- `internal/exchange/bitget/futures_market.go`
- `internal/exchange/bitget/futures_trade.go`
- `internal/exchange/bitget/public_ws.go`
- `internal/exchange/bitget/private_ws.go`
- `internal/exchange/bitget/public_ws_test.go`
- `internal/exchange/bitget/private_ws_test.go`
- `internal/exchange/bitget/futures_account.go`
- `internal/exchange/bitget/futures_account_test.go`

**执行步骤：**
1. 编写失败的 decode 与 private REST tests，覆盖 public candle、private position、private account、签名 header。
2. 运行 `go test ./internal/exchange/bitget -run 'TestDecodePublicCandleFrameIntoBarClosedEvent|TestDecodePrivatePositionEvent|TestDecodePrivateAccountEvent|TestDoPrivateSetsBitgetHeaders' -v`，确认现有实现失败。
3. 增加 futures `PositionEvent`、`AccountEvent`，补齐 `doPrivate`、public/private decoder、`FetchFuturesPositions`、`FetchFuturesAccount`。
4. 运行 `go test ./internal/exchange/bitget -v`。
5. 在 ECS 上执行真实 Bitget read-side 验证：

```bash
PATH=/usr/local/go/bin:$PATH RUN_BITGET_REAL=1 BITGET_API_KEY=$BITGET_API_KEY BITGET_API_SECRET=$BITGET_API_SECRET BITGET_PASSPHRASE=$BITGET_PASSPHRASE go test ./internal/exchange/bitget -run RealBitget -v
```

6. 提交 exchange surface 改动。

**核心实现点：**
- public feed 必须能解 `trade`、`ticker`、`candle*`；私有 feed 必须能解 `orders`、`positions`、`account`。
- `doPrivate` 必须补齐 `ACCESS-KEY`、`ACCESS-PASSPHRASE`、`ACCESS-TIMESTAMP`、`ACCESS-SIGN`。
- 位置、账户、订单查询全部收敛到 futures REST path，而不是沿用 spot path。

**完成判据：**
- futures candle 解码能产出 `bar_closed`。
- private `positions` 与 `account` 解码能产出 `position_snapshot`、`account_snapshot`。
- 真实 Bitget market read 与 private read 在 ECS 上通过。
- 缺少或错误的 `BITGET_PASSPHRASE` 会让 private read 验证失败或显式 skip，Task 2 不能算完成。
- commit message：`feat: add bitget futures decode and private rest client`

## Task 3：增加 raw websocket source、bootstrap gap fill，以及真实 `marketd` collector

**涉及文件：**
- `internal/exchange/bitget/ws_source.go`
- `internal/exchange/bitget/ws_source_test.go`
- `internal/market/gap_fill.go`
- `internal/market/gap_fill_test.go`
- `internal/market/runtime.go`
- `internal/market/runtime_test.go`
- `internal/market/public_feed.go`
- `internal/market/private_feed.go`
- `cmd/marketd/main.go`

**执行步骤：**
1. 编写失败的 collector tests，覆盖 bootstrap + public stream + private stream 一起写入 `event_log`。
2. 运行 `go test ./internal/market -run 'TestRunCollectorWritesBootstrapAndStreamingEvents|TestBuildBootstrapEventsUsesFuturesCandles' -v`，确认 collector 与 bootstrap builder 都不存在。
3. 实现 `RawSource`、public/private subscribe、bootstrap gap fill、`BuildBootstrapEvents`。
4. 实现 `RunCollector` 与 `cmd/marketd` wiring，把 bootstrap、public、private 事件统一落到 `event_log`。
5. 运行：

```bash
PATH=/usr/local/go/bin:$PATH go test ./internal/market ./internal/exchange/bitget -v
PATH=/usr/local/go/bin:$PATH go build ./cmd/marketd
```

6. 在 ECS 上执行真实 `marketd` ingress 验证：

```bash
PATH=/usr/local/go/bin:$PATH RUN_BITGET_REAL=1 BITGET_API_KEY=$BITGET_API_KEY BITGET_API_SECRET=$BITGET_API_SECRET BITGET_PASSPHRASE=$BITGET_PASSPHRASE go test ./internal/market -run RealBitget -v
sqlite3 <state_db_path> "SELECT source, event_kind, count(*) FROM event_log GROUP BY source, event_kind ORDER BY source, event_kind;"
```

7. 提交 `marketd` 闭环改动。

**核心实现点：**
- public source 订阅 futures `trade` 与 `candle1m`。
- private source 在凭证存在时订阅 `orders`、`positions`、`account`，并带 login frame。
- bootstrap 先补 K 线，若凭证存在则补 positions/account snapshot。
- `marketd` 是唯一 exchange ingress writer，source 只允许写成 `market.bootstrap`、`market.public`、`market.private`。

**完成判据：**
- `event_log` 中能看到真实 `market.public` 的 `trade_tick` 与 `bar_closed`。
- `BITGET_PASSPHRASE` 正确时，`event_log` 中能看到真实 `market.private` 的 `position_snapshot` 与 `account_snapshot`。
- Task 5 完成后，这条 collector 路径还必须能看到 `order_fill`。
- commit message：`feat: connect marketd to real bitget futures sources`

## Task 4：从 `Engine` 中剥离执行副作用，增加可恢复的 trader runtime

**涉及文件：**
- `internal/trader/runtime.go`
- `internal/trader/runtime_test.go`
- `internal/trader/runtime_events.go`
- `internal/trader/runtime_events_test.go`
- `internal/trader/state.go`
- `internal/trader/engine.go`
- `internal/trader/engine_test.go`
- `internal/trader/strategy_router.go`
- `internal/trader/position_policy.go`
- `cmd/traderd/main.go`

**执行步骤：**
1. 编写失败的 trader runtime tests，覆盖 checkpoint/cursor 恢复，以及 `bar_closed` 触发 `candidate.created`。
2. 运行 `go test ./internal/trader -run 'TestRuntimeRestoresCheckpointAndCursor|TestRuntimeWritesCandidateEnvelopeForBarClosed' -v`。
3. 把 `Engine` 收敛成 deterministic `Advance(event)`；增加 `CandidateEvent`、`RiskEvent`、`LiveExchange`、`RuntimeConfig`、`Runtime`。
4. 实现 `ProcessAvailable`、`handleEnvelope`、feature snapshot 更新、checkpoint/cursor 持久化。
5. 运行：

```bash
PATH=/usr/local/go/bin:$PATH go test ./internal/trader -v
PATH=/usr/local/go/bin:$PATH go build ./cmd/traderd
```

6. 提交 deterministic runtime split。

**核心实现点：**
- `Engine` 只负责 deterministic candidate generation，不再直接操作 exchange。
- `Runtime` 负责 tail `event_log`、维护 symbol state、保存 cursor/checkpoint。
- `CandidateEvent` 与 `RiskEvent` 变成 live/replay 共用事件模型。

**完成判据：**
- `Runtime` 能从 checkpoint 与 cursor 恢复。
- `candidate.created` 由 `Runtime` 统一写入 `trader` source。
- `Engine` tests 改为只断言 candidate emission，不再断言 side effect。
- commit message：`refactor: move trader execution side effects out of engine`

## Task 5：把 reconciliation、risk gate 与真实 exchange execution 接进 `traderd`

**涉及文件：**
- `internal/trader/runtime.go`
- `internal/trader/risk_engine.go`
- `internal/trader/execution_coordinator.go`
- `internal/trader/reconciler.go`
- `internal/trader/reconciler_test.go`
- `internal/trader/execution_coordinator_test.go`
- `internal/exchange/bitget/futures_trade.go`
- `internal/exchange/bitget/futures_account.go`
- `cmd/traderd/main.go`

**执行步骤：**
1. 编写失败的执行与对账测试，覆盖 `safe` 模式不下单、`armed` 且 risk pass 时可下单、position mismatch 时降级。
2. 运行 `go test ./internal/trader -run 'TestRuntimeDoesNotPlaceOrdersInSafeMode|TestRuntimePlacesOrdersOnlyWhenArmedAndRiskPasses|TestRuntimeDowngradesOnPositionMismatch' -v`。
3. 增加入场下单、reduce-only exit、向下单向切换的 arming transition，以及真实 `TradeClient`。
4. 在 `cmd/traderd` 中仅当 execution 允许时创建真实 exchange client。
5. 重新运行：

```bash
PATH=/usr/local/go/bin:$PATH go test ./internal/trader ./internal/exchange/bitget -v
PATH=/usr/local/go/bin:$PATH go build ./cmd/traderd
```

6. 在 ECS 上执行真实 Bitget 订单生命周期验证：
- 启动 `marketd`、`traderd`
- 设置 `observe_only=false`
- 设置 `arming_state=armed`
- 只保留一个 symbol
- 使用 Bitget 允许的最小 `USDT-FUTURES` 下单量
- 用 signed private REST 查询订单状态
- 若形成持仓，则验证 reduce-only 清仓

7. 提交 execution/reconcile 改动。

**核心实现点：**
- `safe`、`degraded`、`halted` 必须严格生效，任何手工绕过都不允许成为默认路径。
- `maybeExecuteCandidate` 必须先过 risk gate，再检查 `observeOnly`、`arming_state`、`exchange`。
- `applyReconcileVerdict` 只允许向下切换；不允许自动回升到 `armed`。
- `TradeClient` 必须走 Bitget signed private REST path。

**完成判据：**
- 真实 Bitget order 被 `traderd` 下出后，可以被查回。
- SQLite 中的 `trader` 与 `market.private` 证据与 Bitget order state 一致。
- private stream 能回流 `orders`，订单成交时能看到 `order_fill`。
- 若订单未及时成交，必须验证 cancel；若形成持仓，必须验证 reduce-only flat。
- 验证结束后不允许残留未对齐 exchange state 或未平测试仓位。
- commit message：`feat: add trader execution and reconciliation runtime`

## Task 6：让 replay 与 live 收敛到同一套 trader runtime 与 futures replay source

**涉及文件：**
- `internal/adapters/data.go`
- `internal/replay/event_source.go`
- `internal/replay/harness.go`
- `internal/replay/harness_test.go`
- `cmd/lab/main.go`

**执行步骤：**
1. 编写失败的 replay parity tests，覆盖 futures endpoint 路由与 replay 使用 trader runtime 输出。
2. 运行 `go test ./internal/adapters ./internal/replay -run 'TestBitgetEndpointUsesFuturesPathWhenProductTypePresent|TestReplayHarnessUsesTraderRuntimeOutput' -v`。
3. 把 Bitget replay fetch 改成 futures endpoint，并让 replay 复用 `trader.Runtime`。
4. 运行 replay 验证：

```bash
PATH=/usr/local/go/bin:$PATH go test ./internal/adapters ./internal/replay -v
PATH=/usr/local/go/bin:$PATH go run ./cmd/lab replay -config configs/demo-bitget.yaml
```

5. 提交 replay/live convergence 改动。

**核心实现点：**
- `bitgetEndpoint` 必须在 `productType=USDT-FUTURES` 时切到 `/api/v2/mix/market/candles`。
- replay 不再直接驱动 `LegacyRuleProfile`，而是把 historical event 注入与 live 相同的 runtime。
- `artifacts/demo-bitget/replay-report.json` 必须反映 runtime 输出，而不是旧回测旁路。

**完成判据：**
- replay 和 live 使用同一 deterministic kernel。
- futures replay 数据源与 live exchange product type 对齐。
- commit message：`feat: converge replay and live trader runtime`

## Task 7：把 `agentd` 推进为只读的 trader advisory / risk 事件消费者

**涉及文件：**
- `internal/agent/runtime.go`
- `internal/agent/runtime_test.go`
- `internal/agent/service.go`
- `internal/agent/jobs.go`
- `internal/agent/mcp.go`
- `internal/agent/service_test.go`
- `cmd/agentd/main.go`

**执行步骤：**
1. 编写失败的 agent runtime tests，覆盖 `candidate.created` 解释与 `risk.state_changed` 解释。
2. 运行 `go test ./internal/agent -run 'TestRuntimeReviewsCandidateEvents|TestRuntimeExplainsRiskStateChanges' -v`。
3. 实现 advisory runtime，使 `agentd` 从 `trader` source 读取事件并调用 Responses API。
4. 运行：

```bash
PATH=/usr/local/go/bin:$PATH go test ./internal/agent -v
PATH=/usr/local/go/bin:$PATH go build ./cmd/agentd
```

5. 提交 advisory runtime。

**核心实现点：**
- `agentd` 只消费 `trader` 事件，不直接读 exchange，不持有 exchange credential。
- 输入只允许是 `candidate.created` 与 `risk.state_changed` 这类 advisory plane 事件。
- 输出是解释、研究 artifact、后台任务结果，不是 execution intent。

**完成判据：**
- `agentd` 可以对 trader 事件触发 Responses API 调用。
- `agentd` 保持 advisory-only，权限边界不被突破。
- commit message：`feat: add advisory event consumer for agentd`

## Task 8：更新 verification flow 与 rollout gate，使其符合真实运行时

**涉及文件：**
- `scripts/measure.sh`
- `configs/demo-bitget.yaml`
- `configs/live-bitget.yaml`
- `plan/rollout-checklist.md`

**执行步骤：**
1. 先运行当前 demo measure 命令，确认它在 Tasks 3-7 完成后会过强。
2. 在 `measure.sh` 中加入 `RUN_LIVE_SMOKE` gate，让 CI 默认只做 replay-safe 验证。
3. 收紧 config 与 rollout doc，明确 manual arm、observe-only 与 env-only credential contract。
4. 执行完整验证矩阵：

```bash
PATH=/usr/local/go/bin:$PATH go test ./...
PATH=/usr/local/go/bin:$PATH go build ./cmd/lab ./cmd/marketd ./cmd/traderd ./cmd/agentd
PATH=/usr/local/go/bin:$PATH ./scripts/measure.sh configs/baseline.yaml
PATH=/usr/local/go/bin:$PATH SMOKE_SECONDS=3 ./scripts/measure.sh configs/demo-bitget.yaml
PATH=/usr/local/go/bin:$PATH RUN_BITGET_REAL=1 BITGET_API_KEY=$BITGET_API_KEY BITGET_API_SECRET=$BITGET_API_SECRET BITGET_PASSPHRASE=$BITGET_PASSPHRASE go test ./internal/exchange/bitget -run RealBitget -v
PATH=/usr/local/go/bin:$PATH RUN_BITGET_REAL=1 BITGET_API_KEY=$BITGET_API_KEY BITGET_API_SECRET=$BITGET_API_SECRET BITGET_PASSPHRASE=$BITGET_PASSPHRASE go test ./internal/market -run RealBitget -v
PATH=/usr/local/go/bin:$PATH RUN_LIVE_SMOKE=1 SMOKE_SECONDS=15 ./scripts/measure.sh configs/demo-bitget.yaml
```

5. 提交 verification 与 rollout 改动。

**核心实现点：**
- `RUN_LIVE_SMOKE=0` 必须保持 CI-safe。
- `RUN_LIVE_SMOKE=1` 只在真实 Bitget 环境里执行 daemon smoke。
- promotion 仍然是 config + restart 的人工动作，不允许 runtime 自行升回 `armed`。

**完成判据：**
- 前四个命令不依赖 live credential。
- 两个 `RUN_BITGET_REAL=1` 命令必须在 ECS 上通过真实 Bitget public/private 验证。
- `RUN_LIVE_SMOKE=1` 必须在真实环境通过，且不留下脏状态。
- 缺少 `BITGET_PASSPHRASE` 时，不允许把 private read/write 检查算作通过。
- commit message：`docs: tighten runtime verification and rollout gates`

## 最终验证矩阵

### 前置条件

- 在 ECS 环境变量中提供真实 Bitget 凭证：`BITGET_API_KEY`、`BITGET_API_SECRET`、`BITGET_PASSPHRASE`
- `configs/demo-bitget.yaml` 与 `configs/live-bitget.yaml` 中只允许保留环境变量名，不允许存放敏感值
- `demo` 与 `live` 必须使用不同的 `state_db_path`

### 统一验证命令

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

### 统一成功标准

- `marketd` 把 `market.bootstrap`、`market.public`、`market.private` 有序写入 `event_log`
- 真实 Bitget public 验证证明 futures market read、`trade_tick`、`bar_closed` ingestion 已打通
- 真实 Bitget private 验证证明 account、positions、order-query 与 private WebSocket 已打通
- 至少一条真实 order lifecycle 闭环通过：place order → query order → observe `orders` 或 `order_fill` → 如有仓位则 reduce-only flatten
- `traderd` 能恢复 checkpoint 与 cursor，并写出 `candidate.created`、`risk.state_changed`
- `lab replay` 输出 `artifacts/demo-bitget/replay-report.json`，且语义与 `traderd` 一致
- `agentd` 只 tail `trader` 事件，调用 Responses API，但从不接触 exchange credential
- `measure.sh` 默认保持 CI-safe；真实 smoke 必须显式开启
- `mock pass != done`：完成定义必须同时满足 package-level real Bitget checks 与 daemon-level real Bitget smoke

## 与设计文档的覆盖关系

- 设计项 `marketd real Bitget futures public/private source + SQLite event log` 由 Tasks 1、2、3 覆盖
- 设计项 `traderd real consumer + checkpoint + router + policy + risk + reconciliation + execution` 由 Tasks 4、5 覆盖
- 设计项 `replay/live convergence onto one deterministic trader runtime` 由 Task 6 覆盖
- 设计项 `agentd advisory-only event consumer` 由 Task 7 覆盖
- 设计项 `real rollout gates and demo/live promotion checks` 由 Task 8 覆盖

## 审查结论与自检

- 本计划没有引入 dual write、migration guard、第二套 state store。SQLite/WAL 仍是唯一 runtime persistence layer。
- `Engine` 被刻意收敛为 deterministic candidate generator，exchange side effect 被推到 `Runtime`，职责边界更清晰。
- manual arm 继续保持人工显式控制；automatic transition 只允许 downward。
- `agentd` 保持 read-only；AI plane 不获得 execution authority。
- exchange-facing 任务现在都有双层验证，但真实 Bitget 验证才是硬完成门槛。
- secret 处理完全留在 env-only contract；缺少 `BITGET_PASSPHRASE` 时，private/order 任务直接判定为未完成。

