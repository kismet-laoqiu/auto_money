# Agentic Runtime MSTR 实盘闭环实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 用一套真实 Bitget API 凭证，在同一个 `MSTRUSDT` `USDT-FUTURES` 合约上，把系统一开始规划的核心能力全部真实跑一遍：公共行情读取、私有账户与仓位读取、事件落 SQLite、3x long 最小单位开仓、订单查询、成交回流、reduce-only 平仓、trader gate、replay 对齐、agent advisory，并留下可审计证据。

**Architecture:** 本计划不再以 `marketd` / `traderd` / `agentd` 的服务边界作为主叙事，而改成“同一交易对、同一最小实盘练兵路径”的链路叙事。实现上仍保持四进程拆分与一个 SQLite/WAL `event_log`，但验收必须围绕一条唯一实践主线展开：`MSTRUSDT`、`isolated`、`3x`、`long`、最小下单单位、真实 Bitget、真实回补平仓、最终回到空仓。

**Tech Stack:** Go, SQLite/WAL, Bitget futures REST/WebSocket, Gorilla WebSocket, YAML, OpenAI Responses API, `MSTRUSDT` `USDT-FUTURES` contract, ECS runtime.

---

## 1. 文档身份与真实基线

| 项目 | 内容 |
| --- | --- |
| 中文 plan 路径 | `plan/2026-03-28-agentic-runtime-mstr-e2e-plan-cn.md` |
| ECS worktree | `/root/.config/superpowers/worktrees/quant-lab/autoresearch-20260328-all-plan` |
| 审查目的 | 明确“我们支持哪些核心链路”和“这些链路如何全部用真实 Bitget 跑一遍” |
| 唯一实践标的 | `MSTRUSDT` |
| Product Type | `USDT-FUTURES` |
| 交易方向 | `long` |
| 杠杆要求 | `3x` |
| 保证金模式 | `isolated` |
| 最小下单单位 | `0.01` |

### 1.1 已验证的 Bitget 合约事实

以下事实来自 2026-03-28 在 ECS 上对 Bitget 公共接口 `GET /api/v2/mix/market/contracts?productType=USDT-FUTURES` 的真实查询：

| 字段 | 实际值 |
| --- | --- |
| `symbol` | `MSTRUSDT` |
| `baseCoin` | `MSTR` |
| `quoteCoin` | `USDT` |
| `symbolStatus` | `normal` |
| `minTradeNum` | `0.01` |
| `sizeMultiplier` | `0.01` |
| `minTradeUSDT` | `5` |
| `minLever` | `1` |
| `maxLever` | `25` |
| `volumePlace` | `2` |
| `pricePlace` | `2` |
| `maxMarketOrderQty` | `200` |

从这些真实字段可以直接得出三件事：

1. `MSTRUSDT` futures 当前确实存在，且状态为 `normal`。
2. “最小单位 + 3x long”的实践主线成立，因为 `minTradeNum=0.01` 且 `3` 位于 `1..25` 杠杆范围内。
3. 所有核心能力都可以围绕同一个 symbol 验证，不需要在 plan 里切换第二个交易对。

### 1.2 凭证事实边界

本计划将用户口中的“Bitget API key 实跑”统一解释为“一套 Bitget API 凭证三件套实跑”：

- `BITGET_API_KEY`
- `BITGET_API_SECRET`
- `BITGET_PASSPHRASE`

原因不是文风问题，而是接口事实问题。Bitget 私有 REST 与私有 WebSocket 认证头都要求：

- `ACCESS-KEY`
- `ACCESS-SIGN`
- `ACCESS-TIMESTAMP`
- `ACCESS-PASSPHRASE`

所以：

- 公共行情链路可以不带凭证跑。
- 私有账户、仓位、订单、下单、撤单、查单、平仓链路不能只靠 API key；必须是完整凭证三件套。
- plan 中不能再写成“只用 key 就能把核心私有链路跑通”，那是错误要求。

## 2. 本轮必须真实跑通的核心链路

| 核心链路 | 必须使用的真实条件 | 预期落盘证据 | 完成定义 |
| --- | --- | --- | --- |
| 合约元数据链路 | `MSTRUSDT` `USDT-FUTURES` | contract 查询结果、最小单位与杠杆字段 | 证明目标合约存在且支持 `0.01`、`3x` |
| Public bootstrap 链路 | 真实 Bitget public REST | `event_log` 中 bootstrap K 线 | `market.bootstrap` 可恢复 trader 初始特征 |
| Public stream 链路 | 真实 Bitget public WS | `trade_tick`、`bar_closed` | `market.public` 持续写入 SQLite |
| Private snapshot 链路 | 真实私有 REST | `account_snapshot`、`position_snapshot`、`order_snapshot` | `market.private` 可恢复账户与仓位真相 |
| Private stream 链路 | 真实私有 WS | `orders`、`positions`、`account` 更新 | 下单后状态变化能自动回流 |
| 杠杆设置链路 | `MSTRUSDT` `isolated` `3x` `long` | leverage 设置请求与查询结果 | 开仓前确认 3x long 配置已真实生效 |
| 开多链路 | `MSTRUSDT` `0.01` `3x` `long` | place-order 请求、SQLite、order detail | 最小单位 long 开仓成功 |
| 订单查询链路 | 真实 private REST | `orderId` / `clientOid` 查询结果 | 下单后可以独立查单 |
| 成交与持仓回流链路 | 真实私有推送 | `order_fill`、position/account 更新 | 成交后本地与 Bitget 状态一致 |
| Reduce-only 平仓链路 | 真实 private REST | reduce-only close order、最终空仓 | 验证结束后账户回到空仓 |
| Trader gate 链路 | `safe/degraded/halted` | `risk.state_changed`、`candidate.created` | `traderd` gate 行为真实生效 |
| Replay parity 链路 | 同一个 `MSTRUSDT` | replay report 与 live 事件语义 | replay/live 共用同一 deterministic kernel |
| Agent advisory 链路 | 消费真实 `trader` 事件 | advisory artifact、Responses 调用记录 | `agentd` 对真实事件解释，但无执行权 |

## 3. 本轮对外声称“支持”的功能清单

只有下表中的功能全部在真实 Bitget 上跑过一遍，才允许对外声称“系统已支持”。

| 功能 | 是否必须真实跑 | 实践约束 |
| --- | --- | --- |
| 读取 `MSTRUSDT` 合约配置 | 是 | 真实 public REST |
| 拉取 `MSTRUSDT` futures candles | 是 | 真实 public REST |
| 订阅 `MSTRUSDT` trade stream | 是 | 真实 public WS |
| 订阅 `MSTRUSDT` candle stream | 是 | 真实 public WS |
| 读取账户资产 | 是 | 真实 private REST |
| 读取当前持仓 | 是 | 真实 private REST |
| 读取单笔订单详情 | 是 | 真实 private REST |
| 订阅 orders / positions / account | 是 | 真实 private WS |
| 设置 `MSTRUSDT` `isolated` `3x` long | 是 | 真实 private REST |
| 开最小单位 long | 是 | `size=0.01`，真实 private REST |
| 查询刚刚下出的订单 | 是 | 必须用 `orderId` 或 `clientOid` 查回 |
| 接收成交回报并写 SQLite | 是 | 必须看到 `order_fill` 或等价成交更新 |
| reduce-only 平仓 | 是 | 若形成仓位，必须真实平掉 |
| 空仓收尾校验 | 是 | 真实 position/account 再次查询 |
| safe/degraded/halted gate | 是 | 真实 runtime 行为 |
| replay 与 live 同核 | 是 | 同 symbol，同 runtime 语义 |
| agent 对真实事件做解释 | 是 | 只读，不接触交易权限 |

## 4. 本轮不支持声称完成的范围

以下内容即使顺手做了，也不构成这份 plan 的主交付，不得替代 `MSTRUSDT` 实盘主线：

- 多 symbol 并行验证
- short 方向验证
- cross margin 验证
- AI 自动下单
- 未经真实 Bitget 跑通的“接口已支持”声明
- 只基于 unit test / fake server / demo replay 的完成声明

## 5. 文件结构与落点

### 5.1 既有文件

- `internal/store/sqlite/schema.sql`
- `internal/store/sqlite/live_store.go`
- `internal/store/sqlite/live_store_test.go`
- `internal/exchange/bitget/rest_client.go`
- `internal/exchange/bitget/models.go`
- `internal/exchange/bitget/futures_market.go`
- `internal/exchange/bitget/futures_trade.go`
- `internal/exchange/bitget/public_ws.go`
- `internal/exchange/bitget/private_ws.go`
- `internal/market/public_feed.go`
- `internal/market/private_feed.go`
- `internal/market/event.go`
- `internal/trader/runtime.go`
- `internal/trader/engine.go`
- `internal/trader/risk_engine.go`
- `internal/trader/reconciler.go`
- `internal/trader/execution_coordinator.go`
- `internal/replay/event_source.go`
- `internal/replay/harness.go`
- `internal/agent/service.go`
- `internal/agent/jobs.go`
- `internal/agent/mcp.go`
- `cmd/marketd/main.go`
- `cmd/traderd/main.go`
- `cmd/lab/main.go`
- `cmd/agentd/main.go`
- `scripts/measure.sh`
- `plan/rollout-checklist.md`

### 5.2 新增文件

- `configs/demo-mstr-e2e.yaml`
- `scripts/run_mstr_e2e.sh`
- `scripts/run_mstr_e2e_test.sh`
- `internal/exchange/bitget/futures_account.go`
- `internal/exchange/bitget/ws_source.go`
- `internal/market/runtime.go`
- `internal/trader/runtime_events.go`
- `internal/agent/runtime.go`

## 6. 实施任务

## Task 1: 固化 `MSTRUSDT` 唯一实践主线与实盘配置

**Files:**
- Create: `configs/demo-mstr-e2e.yaml`
- Create: `scripts/run_mstr_e2e.sh`
- Modify: `configs/demo-bitget.yaml`
- Modify: `plan/rollout-checklist.md`

- [ ] **Step 1: 固化单一验证场景**
  把唯一实践场景写进配置与脚本：`symbol=MSTRUSDT`、`productType=USDT-FUTURES`、`marginMode=isolated`、`leverage=3`、`direction=long`、`size=0.01`。

- [ ] **Step 2: 固化凭证与预检**
  `scripts/run_mstr_e2e.sh` 必须先检查 `BITGET_API_KEY`、`BITGET_API_SECRET`、`BITGET_PASSPHRASE`，然后查询 `MSTRUSDT` 合约元数据并断言 `minTradeNum=0.01`、`maxLever>=3`。

- [ ] **Step 3: 运行配置预检**
  运行 `go run ./cmd/marketd -config configs/demo-mstr-e2e.yaml` 与 `go run ./cmd/traderd -config configs/demo-mstr-e2e.yaml` 的配置加载路径，确认配置能被真实加载。

- [ ] **Step 4: 提交 MSTR 实盘配置基线**

## Task 2: 落地顺序 `event_log` 与 consumer cursor

**Files:**
- Modify: `internal/store/sqlite/schema.sql`
- Modify: `internal/store/sqlite/live_store.go`
- Modify: `internal/store/sqlite/live_store_test.go`

- [ ] **Step 1: 补失败测试**
  覆盖 `AppendEvent` 返回 `seq`、`ListEventsAfter`、`SaveConsumerCursor/LoadConsumerCursor`。

- [ ] **Step 2: 落地 `event_log`**
  增加统一事件序列、`source`、`event_kind`、`payload_json`、`received_ts`，作为全系统唯一消费底座。

- [ ] **Step 3: 落地 cursor API**
  让 `marketd`、`traderd`、`agentd`、`replay` 都使用同一种 cursor 语义。

- [ ] **Step 4: 运行 sqlite package tests**
  `go test ./internal/store/sqlite -v`

- [ ] **Step 5: 提交存储层改动**

## Task 3: 打通 `MSTRUSDT` Public market 链路

**Files:**
- Modify: `internal/exchange/bitget/futures_market.go`
- Modify: `internal/exchange/bitget/public_ws.go`
- Modify: `internal/market/public_feed.go`
- Modify: `internal/market/runtime.go`
- Modify: `cmd/marketd/main.go`

- [ ] **Step 1: 打通 public REST bootstrap**
  `marketd` 启动时必须拉取 `MSTRUSDT` futures candles，落 `market.bootstrap`。

- [ ] **Step 2: 打通 public WS**
  订阅并解码 `trade` 与 `candle1m`，产出 `trade_tick` 与 `bar_closed`。

- [ ] **Step 3: 写入 SQLite**
  真实 public 事件必须按 `market.public` source 落入 `event_log`。

- [ ] **Step 4: 运行真实验证**

```bash
PATH=/usr/local/go/bin:$PATH RUN_BITGET_REAL=1 go test ./internal/exchange/bitget ./internal/market -run RealBitget -v
sqlite3 <state_db_path> "SELECT source, event_kind, count(*) FROM event_log WHERE source IN ('market.bootstrap','market.public') GROUP BY source, event_kind ORDER BY source, event_kind;"
```

- [ ] **Step 5: 提交 public market 链路**

## Task 4: 打通 `MSTRUSDT` Private state 链路

**Files:**
- Modify: `internal/exchange/bitget/rest_client.go`
- Modify: `internal/exchange/bitget/private_ws.go`
- Create: `internal/exchange/bitget/futures_account.go`
- Modify: `internal/market/private_feed.go`
- Modify: `cmd/marketd/main.go`

- [ ] **Step 1: 打通 private REST**
  支持账户资产、当前持仓、单笔订单查询，全部对准 `MSTRUSDT` `USDT-FUTURES`。

- [ ] **Step 2: 打通 private WS**
  订阅 `orders`、`positions`、`account`，并解码为统一事件。

- [ ] **Step 3: 写入 SQLite**
  私有事件统一落到 `market.private`，至少包含 `account_snapshot`、`position_snapshot`、`order_snapshot` 或等价订单状态事件。

- [ ] **Step 4: 运行真实验证**

```bash
PATH=/usr/local/go/bin:$PATH RUN_BITGET_REAL=1 BITGET_API_KEY=$BITGET_API_KEY BITGET_API_SECRET=$BITGET_API_SECRET BITGET_PASSPHRASE=$BITGET_PASSPHRASE go test ./internal/exchange/bitget ./internal/market -run RealBitget -v
sqlite3 <state_db_path> "SELECT source, event_kind, count(*) FROM event_log WHERE source='market.private' GROUP BY source, event_kind ORDER BY event_kind;"
```

- [ ] **Step 5: 提交 private state 链路**

## Task 5: 打通 `MSTRUSDT` `isolated` `3x` `long` 最小单位执行链路

**Files:**
- Modify: `internal/exchange/bitget/futures_trade.go`
- Modify: `internal/trader/execution_coordinator.go`
- Modify: `internal/trader/runtime.go`
- Modify: `cmd/traderd/main.go`

- [ ] **Step 1: 补齐 trade client 能力**
  至少实现：设置杠杆、下单、查单、撤单、reduce-only 平仓。

- [ ] **Step 2: 固化唯一订单模板**
  开仓单固定为 `MSTRUSDT`、`USDT-FUTURES`、`isolated`、`marginCoin=USDT`、`leverage=3`、`side=buy`、`tradeSide=open`、`size=0.01`。

- [ ] **Step 3: 补齐平仓模板**
  若形成持仓，则必须发送 `reduceOnly` 的最小可行平仓指令，直到仓位回到零。

- [ ] **Step 4: 跑真实订单生命周期**
  用 `scripts/run_mstr_e2e.sh` 顺序执行：设置 3x → 开 0.01 long → 查单 → 等成交或撤单 → 若开仓成功则 reduce-only 平仓 → 再次查持仓与账户。

- [ ] **Step 5: 落证据**
  SQLite 中必须能看到 `candidate.created`、订单状态事件、`order_fill`、最终空仓相关 `position_snapshot`。

- [ ] **Step 6: 提交 execution 链路**

## Task 6: 严格落地 `safe/degraded/halted` 与对账逻辑

**Files:**
- Modify: `internal/trader/risk_engine.go`
- Modify: `internal/trader/reconciler.go`
- Modify: `internal/trader/runtime.go`
- Modify: `internal/trader/state.go`

- [ ] **Step 1: 让 `safe` 绝不下单**
  即使 market/public/private 事件都到齐，只要不是 `armed`，真实 Bitget 不得发出开仓单。

- [ ] **Step 2: 让 `degraded/halted` 有真实证据**
  position mismatch、order mismatch、account mismatch 必须能产出 `risk.state_changed`。

- [ ] **Step 3: 只允许向下自动切换**
  runtime 允许自动降到 `degraded` 或 `halted`，但不得自动回到 `armed`。

- [ ] **Step 4: 运行真实 gate 验证**
  在同一 `MSTRUSDT` 主线上分别验证 `safe`、`armed`、手工 `halt` 的实际行为。

- [ ] **Step 5: 提交 gate 与对账逻辑**

## Task 7: 让 replay 与 live 在 `MSTRUSDT` 上共用同一内核

**Files:**
- Modify: `internal/adapters/data.go`
- Modify: `internal/replay/event_source.go`
- Modify: `internal/replay/harness.go`
- Modify: `cmd/lab/main.go`

- [ ] **Step 1: 把 replay 数据源改成 futures**
  `MSTRUSDT` 历史数据必须来自 futures path，而不是 spot path。

- [ ] **Step 2: replay 注入同一 runtime**
  `lab replay` 不再单独调用旧旁路逻辑，而是把同 symbol 历史事件送进 `trader.Runtime`。

- [ ] **Step 3: 运行 parity 验证**

```bash
PATH=/usr/local/go/bin:$PATH go test ./internal/adapters ./internal/replay -v
PATH=/usr/local/go/bin:$PATH go run ./cmd/lab replay -config configs/demo-mstr-e2e.yaml
```

- [ ] **Step 4: 提交 replay/live convergence**

## Task 8: 让 `agentd` 只读消费真实 `MSTRUSDT` trader 事件

**Files:**
- Create: `internal/agent/runtime.go`
- Modify: `internal/agent/service.go`
- Modify: `internal/agent/jobs.go`
- Modify: `internal/agent/mcp.go`
- Modify: `cmd/agentd/main.go`

- [ ] **Step 1: 让 `agentd` 只消费 `trader` 事件**
  输入只允许是真实 `MSTRUSDT` 的 `candidate.created` 与 `risk.state_changed`。

- [ ] **Step 2: 调用 Responses API 生成解释**
  输出是 advisory artifact，不是执行指令。

- [ ] **Step 3: 验证权限边界**
  `agentd` 不能读取 Bitget credential，不能直接调用 order API。

- [ ] **Step 4: 提交 advisory plane**

## Task 9: 产出一份一次性 `MSTRUSDT` 实盘证据包

**Files:**
- Create: `artifacts/mstr-e2e/<run_id>/`
- Modify: `scripts/run_mstr_e2e.sh`
- Modify: `scripts/measure.sh`

- [ ] **Step 1: 统一收集证据**
  每次真实 run 都生成 `<run_id>` 目录，收集 contract metadata、配置快照、SQLite 查询结果、下单与查单响应、最终空仓校验结果。

- [ ] **Step 2: 固化最小证据清单**
  至少包括：
  - `contract.json`
  - `preflight.txt`
  - `event-log-summary.txt`
  - `order-open.json`
  - `order-detail.json`
  - `position-after-open.json`
  - `order-close.json`
  - `position-final.json`
  - `account-final.json`
  - `replay-report.json`
  - `agent-advisory.json`

- [ ] **Step 3: 跑完整 `MSTRUSDT` E2E**
  要求一条 run 同时覆盖 public read、private read、order open、order query、order fill / cancel、reduce-only close、final flat。

- [ ] **Step 4: 提交证据收集链路**

## 7. 最终验证矩阵

### 7.1 必跑命令

```bash
PATH=/usr/local/go/bin:/usr/bin:/bin go test ./...
PATH=/usr/local/go/bin:/usr/bin:/bin go build ./cmd/lab ./cmd/marketd ./cmd/traderd ./cmd/agentd
PATH=/usr/local/go/bin:/usr/bin:/bin RUN_BITGET_REAL=1 go test ./internal/exchange/bitget ./internal/market -run RealBitget -v
PATH=/usr/local/go/bin:/usr/bin:/bin ./scripts/run_mstr_e2e.sh
PATH=/usr/local/go/bin:/usr/bin:/bin go run ./cmd/lab replay -config configs/demo-mstr-e2e.yaml
PATH=/usr/local/go/bin:/usr/bin:/bin go build ./cmd/agentd && timeout --preserve-status --signal=INT --kill-after=2s 120s ./agentd -config configs/demo-mstr-e2e.yaml
```

### 7.2 必须同时满足的成功标准

- `MSTRUSDT` 合约真实存在，且预检确认 `minTradeNum=0.01`、`maxLever>=3`
- `event_log` 中有 `market.bootstrap`、`market.public`、`market.private` 三类真实事件
- 真实 `trade_tick` 与 `bar_closed` 已写入 SQLite
- 真实 `account_snapshot`、`position_snapshot` 已写入 SQLite
- 杠杆设置真实生效为 `3x`
- 成功发出一笔 `MSTRUSDT` `0.01` `long` 开仓单
- 可以通过 `orderId` 或 `clientOid` 查回该订单
- 订单状态变化和成交回流进入 SQLite
- 若形成持仓，则 reduce-only 平仓成功，最终持仓回到零
- `traderd` 在 `safe/degraded/halted` 三种状态下行为正确
- replay 与 live 在同一 `MSTRUSDT` 上使用同一内核语义
- `agentd` 对真实 trader 事件给出 advisory，但没有交易权限

### 7.3 明确的失败定义

以下任一条出现，都视为本轮未完成：

- 只跑通 public 链路，private/order 链路没跑
- 只跑通 fake server，没有在真实 Bitget 跑
- 没有真实 `MSTRUSDT` 最小单位开多记录
- 下单后没做查单
- 开仓后没有 reduce-only 平仓收尾
- 结束时账户仍有测试持仓
- 事件没有落入 SQLite
- `agentd` 获得了任何 exchange 执行权限
- 使用第二个交易对替代 `MSTRUSDT`

## 8. 与最初规划的对应关系

这份重写 plan 对应的不是“又多写了一层文档”，而是把一开始的核心能力收敛成一条可真实审计的单链路：

- `marketd`：负责把 `MSTRUSDT` 的公共与私有事实落盘
- `traderd`：负责在同一 symbol 上执行 deterministic gate 与实盘下单
- `lab replay`：负责用同一 symbol 的历史事件复用 live kernel
- `agentd`：负责解释同一 symbol 上的真实候选与风险事件

换句话说，本轮不是抽象支持“任意 futures 交易对”，而是先把 `MSTRUSDT` 这一个最小真实练兵路径完整打透。

## 9. Plan Self-Review

- 这份 plan 已从“按服务拆任务”改成“按一条真实 `MSTRUSDT` 实盘链路拆任务”，更适合审查。
- 文档明确区分了“支持的核心链路”和“支持的功能”，不再把两者混在一起。
- 文档明确纠正了“只靠 API key 即可实跑私有链路”的错误说法，改成真实凭证三件套事实。
- 文档把“最小单位、3x、long、真实开仓、真实查单、真实平仓、最终空仓”写成完成门槛，而不是可选验证。
- 文档没有把完成标准停留在 `go test ./...`，而是要求真实 Bitget `MSTRUSDT` E2E 证据包。
