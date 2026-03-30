# 2026-03-29 Platform Baseline Checklist

## Scope

本文件冻结 `2026-03-29` 当前 ECS 现场，为 one-shot plan 的 `P01-06` 与 `P01-07` 提供锚点。后续任何 `execd / bundle / promotion / Telegram ingress` 改造，都必须先对照这里的基线，而不是靠会话记忆。

## Remote Baseline Facts

| Item | Value |
| --- | --- |
| ECS | `47.250.138.143` / `root` |
| Repo | `/root/.config/superpowers/worktrees/quant-lab/autoresearch-20260328-all-plan` |
| Branch | `autoresearch/20260328-all-plan` |
| Commit | `c409e465d4948cb5289caf92b7194be911579274` |
| Global test gate | `PATH=/usr/local/go/bin:/usr/bin:/bin go test ./...` passes |
| Baseline score command | `PATH=/usr/local/go/bin:/usr/bin:/bin ./scripts/measure.sh configs/baseline.yaml` |
| Real Bitget smoke | `./scripts/run_mstr_e2e.sh` already verified with final flat cleanup |

## Runtime SQLite Contract

当前 runtime edge log 仍然只有一个 SQLite/WAL state db，结构来自 `internal/store/sqlite/schema.sql`：

| Table | Fields | Role |
| --- | --- | --- |
| `event_log` | `seq, source, event_id, symbol, event_kind, exchange_ts, received_ts, payload_json` | 顺序化事件主日志 |
| `engine_checkpoint` | `shard_key, state_json, updated_at` | `marketd / traderd` checkpoint |
| `consumer_cursor` | `consumer_key, last_seq, updated_at` | 事件消费者 offset |

补充事实：

- `payload_json` schema 仍声明为 `BLOB`，但 `internal/store/sqlite/live_store.go` 已兼容从 `BLOB` 或 `TEXT` 读取历史行。
- 当前 control plane `platformd` 与 `notifierd` 都直接读取这个 SQLite，而不是新的 warehouse / registry / ops db。

## Current Runtime Topology

### `marketd`

- 仍是唯一 exchange ingress。
- 负责把 `market.bootstrap`、`market.public`、`market.private` 事件写入 `event_log`。

### `traderd`

- 当前既是 deterministic evaluator，也是临时 live writer。
- `internal/trader/runtime.go` 在收到 `Candidate` 后，仍会在 `ObserveOnly=false` 且 `ArmingState=armed` 时直接调用 `Exchange.PlaceOrder`。
- `internal/trader/execution_coordinator.go` 仍保留 `BuildEntryRequest / BuildExitRequest`。
- live size 仍硬编码 `defaultEntryQtyValue = 0.01` / `defaultEntryQtyText = "0.01"`，与真实 Bitget `MSTRUSDT >= 5 USDT` 最小有效 size `0.04` 冲突。

### `agentd`

- 仍是 advisory-only consumer。
- `cmd/agentd/main.go` 强制 `live.agent.advisory_only=true`，并且只会消费 trader 事件、调用 Responses API 生成解释性 artifact。
- 结论：当前 `agentd` 不是交易执行面，也不是 research orchestration 面。

### `platformd / platformctl / notifierd`

- `platformd` 已真实提供 `health / status / positions / orders / events`。
- `platformctl` 当前是 operator CLI，只做状态读取与 `notify test`。
- `notifierd` 当前消费 `event_log` 并发送 Telegram / DingTalk 通知。

### Legacy `stream`

- `cmd/stream` 仍直接调用 `internal/adapters/stream.go`。
- 该链路直接订阅 Bitget public WS、在内存里维护 bar cache、调用 `core.EvaluateSignal` 并直接发通知。
- 它不写 `event_log`、不经 `platformd`、不参与 promotion governance。
- 结论：`stream` 只能视为 legacy observer，不是平台完成信号。

## Current Scoring Baseline

来自 `./scripts/measure.sh configs/baseline.yaml` 的最新输出：

| Metric | Value |
| --- | --- |
| `OBJECTIVE_SCORE` | `0.560497091678` |
| `FINAL_SCORE` | `0.863671625267` |
| `MEDIAN_OBJECTIVE_SCORE` | `1.228197587246` |
| `MIN_BUCKET_MEAN` | `0.130429586905` |
| `POSITIVE_OOS_RETURN_RATIO` | `0.777777777778` |
| `REPLAY_EVENT_COUNT` | `4929` |
| `REPLAY_COMMAND_COUNT` | `10` |
| `REPLAY_CANDIDATE_COUNT` | `10` |
| `BUCKET_COMMODITY` | `1.670822906521` |
| `BUCKET_CRYPTO` | `0.130429586905` |
| `BUCKET_US_EQUITY` | `0.518798624129` |

当前 `FINAL_SCORE` 仍由 `cmd/lab/main.go` 调 `core.BuildRobustScore(reports)` 生成，不存在 bundle / CEL / registry 版本边界。

## Bundle / Registry Gap Snapshot

当前 repo 还没有以下平台级对象：

- `internal/strategybundle/`
- `strategies/<strategy_id>/versions/<version>/`
- `score.cel`
- `gates.cel`
- strategy registry store
- promotion state machine

当前策略与风控配置仍然散落在：

- `configs/*.yaml` 的 `strategy` / `live.exchange.symbols` / `live.risk`
- `internal/trader/legacy_strategy.go`
- `internal/trader/execution_coordinator.go`
- `cmd/lab/main.go` 中的固定 `LegacyRuleProfile`

结论：现在还没有“可被 AI 修改、可回测、可评分、可上线”的 bundle 形态，只有 hardcoded baseline runtime。

## Notification Baseline

当前已真实验证的通知能力：

| Event / Action | Status |
| --- | --- |
| `platformctl notify test --channel telegram` | real success |
| `platformctl notify test --channel dingtalk` | real success |
| `notifierd` handles `order_fill` | implemented |
| `notifierd` handles `risk.state_changed` (`degraded|halted`) | implemented |
| `notifierd` handles `promotion.approved` | implemented |
| `notifierd` handles `promotion.canary_degraded` | implemented |

仍然缺失：

- Telegram inbound command ingress
- 中文模板 snapshot 固化
- retry / dead-letter
- Asia/Shanghai daily summary 调度
- notifier health metrics

## Rollout Blockers Frozen At P01

1. `traderd` 仍直接持有 Bitget write path，`execd` 尚未落地。
2. live size 仍写死 `0.01`，与真实最小有效 size `0.04` 冲突。
3. `stream` 仍是 legacy observer，不能被当作平台完成度证据。
4. `platformctl` 还没有 `flatten / backtest / promotion / live ops`。
5. Telegram 当前仍是 outbound-only，OpenClaw 也还没有 Telegram channel。

## Verification Commands

```bash
cd /root/.config/superpowers/worktrees/quant-lab/autoresearch-20260328-all-plan
PATH=/usr/local/go/bin:/usr/bin:/bin go test ./...
PATH=/usr/local/go/bin:/usr/bin:/bin ./scripts/measure.sh configs/baseline.yaml
PATH=/usr/local/go/bin:/usr/bin:/bin go build ./cmd/lab ./cmd/marketd ./cmd/traderd ./cmd/agentd ./cmd/platformd ./cmd/platformctl ./cmd/notifierd
```

真实 Bitget 基线：

```bash
env RUN_BITGET_REAL=1 RUN_BITGET_CLEANUP=1 \
BITGET_API_KEY=... \
BITGET_API_SECRET=... \
BITGET_PASSPHRASE=... \
PATH=/usr/local/go/bin:/usr/bin:/bin \
go test ./internal/exchange/bitget -run 'TestRealBitget(FetchFuturesCandles|PrivateReadSurface|OrderLifecycleAndPrivateStream|EnsureFlatPosition)' -v

PATH=/usr/local/go/bin:/usr/bin:/bin ./scripts/run_mstr_e2e.sh
```
