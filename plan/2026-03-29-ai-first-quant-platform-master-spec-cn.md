# AI-First Quant Platform Master Design Spec

## 文档身份

| 项目 | 内容 |
| --- | --- |
| 文档定位 | 平台总设计文档，不是单个 feature 的局部方案 |
| 目标仓库 | `/root/.config/superpowers/worktrees/quant-lab/autoresearch-20260328-all-plan` |
| 文档日期 | `2026-03-29` |
| 目标环境 | `47.250.138.143` ECS |
| 实盘交易所 | Bitget `USDT-FUTURES` |
| AI 运行面 | `codex`, `claude`, `openclaw` 均已安装在 ECS |
| 人机交互主面 | Telegram first, DingTalk push, OpenClaw conversational gateway |

## 1. 目标定义

这个项目的目标不是再做一个“能跑 demo 的量化样板”，而是建设一套 **AI-FIRST 的稳定策略交易平台**，满足四个同时成立的条件：

1. 平台本身是独立稳定的交易系统，哪怕没有 AI 也能完成数据获取、特征计算、回测、通知、实盘交易。
2. 平台要向 `codex`、`claude`、`openclaw` 暴露清晰、强约束、可审计的操作面，让 AI 可以基于平台内的数据、理论文档和历史结果开展研究、修改策略、发起回测、比较结果、准备上线。
3. 研究、回测、上线、实盘执行必须共享同一套策略定义和风险约束，不能出现“AI 研究一套、平台执行另一套”的双轨漂移。
4. 所有关键路径都必须能在 ECS 上做真实验证，尤其是 Bitget 真实 public/private read、真实 websocket、真实最小单位下单、真实 reduce-only 平仓、最终空仓收尾。

## 2. 从用户需求抽出来的硬要求

用户真正要的是一套“策略工厂”，不是一个会说话的 agent，也不是一个只能回测的脚本集。平台必须一次性覆盖以下完整能力：

### 2.1 平台基础能力

- 多资产历史 K 线采集、校验、补洞、聚合、导出。
- 多时间粒度统一管理，例如 `1m/5m/15m/1h/4h/1d`。
- 波浪理论、斐波那契、量价、撑压线、价格行为、regime 等特征的稳定计算。
- 实盘 public/private websocket 持续接入。
- 订单、仓位、账户、风控、通知、回测、上线流程全部平台化。

### 2.2 AI-FIRST 能力

- AI 能主动读取平台数据，不靠手工复制粘贴。
- AI 能学习理论文档、历史样本、特征快照、历史回测结果。
- AI 能修改策略定义，而不是只能改一些脆弱脚本参数。
- AI 能发起回测、读取评分、比较版本、生成上线候选。
- AI 能在严格权限边界内发起 shadow / canary / live promotion 流程。

### 2.3 交互与运维能力

- 用户需要日常交互面。
- 前端不是必须项，Telegram 是第一优先级。
- DingTalk 机器人必须集成。
- OpenClaw 必须纳入体系，因为 ECS 上已经可用，而且它天然提供多 channel、agent、cron、dashboard、skills、gateway 能力。

## 3. 当前代码事实

当前仓库已经有一批很重要的基础，不需要推倒重来，但也远远还不是平台：

### 3.1 已有的真实基础

- Bitget futures public/private REST 与 websocket 已经打通，并且有真实测试与真实最小下单生命周期验证。
- `marketd` 已经能把 bootstrap/public/private 事件写入顺序化 SQLite/WAL `event_log`。
- `traderd` 已经能消费统一事件并产出 deterministic candidate/risk event。
- `agentd` 已经能消费 trader 事件并调用 Responses API 生成 advisory artifact。
- `scripts/run_mstr_e2e.sh` 已经能在真实 Bitget 上根据最新价格与合约规则计算 `>= 5 USDT` 的最小有效 size，并做 smoke/preflight。
- 当前特征层并不空白，`wave/levels/fib/trigger/volume/regime` 已经有 `core.FeatureSet` 和 snapshot tests。

### 3.2 当前的根本缺口

- `internal/adapters/data.go` 仍然主要是“抓数据到 CSV 再回测”的研究脚手架，不是平台级市场数据仓。
- `cmd/stream` + `internal/adapters/stream.go` 仍然是旧式只读报警器，不是统一交互面。
- `agentd` 只是 advisory-only 的 LLM 调用器，不是 AI 研究/改策略/上线的平台操作面。
- 还没有策略注册、版本管理、promotion、shadow、canary、rollback 的控制平面。
- 还没有给 `codex`/`claude`/`openclaw` 用的统一 MCP / skill / command surface。
- 还没有 Telegram 交互层。
- 回测和 live 虽已共享大部分 signal 入口，但还没有“平台级统一策略定义格式”。

## 4. 终态方案比较

### 方案 A：沿着当前仓库继续加脚本

做法：

- 保持现有 `lab/marketd/traderd/agentd` 结构不变。
- 再加一批 shell script、YAML config、手工规则和 prompt。
- Telegram、上线、回测、研究都通过脚本粘起来。

优点：

- 短期最快。
- 对现有代码侵入最小。

缺点：

- 最终会得到一个“会跑但不会长”的脚本森林。
- AI 工作流会极度脆弱，复现性差，权限边界模糊。
- 策略版本、回测结果、promotion、live state 会继续散落。
- 平台不是平台，还是脚本集。

结论：

- 不推荐。

### 方案 B：先做完整 Web SaaS 控制台

做法：

- 先建统一数据库、HTTP API、前端 dashboard。
- 再把数据、回测、交易、AI 全部挂进去。

优点：

- 产品形态完整。
- 面向人类操作更直观。

缺点：

- 当前真正最缺的是平台 substrate，不是好看的 dashboard。
- 在 substrate 没稳定之前先做 UI，会把精力打散。
- 你明确说 Telegram 更简单，而且 OpenClaw 已经在现场。

结论：

- 不作为第一优先级。

### 方案 C：AI-FIRST Strategy Factory on Stable Trading Substrate

做法：

- 保留并强化已经证明可用的 Go runtime 主干。
- 把交易与通知做成独立稳定的内部平台能力。
- 在其上建设统一市场数据仓、统一策略版本格式、统一回测评分面、统一 promotion 面。
- 再通过 Telegram、OpenClaw、MCP、skills 暴露给 `codex` 和 `claude`。

优点：

- 同时满足“稳定交易系统”和“AI-first research platform”。
- 最大化复用当前已验证的 Bitget runtime 与 feature 基础。
- 最适合单 ECS 起步并逐步增强。
- 人和 AI 都面向同一份平台事实，不会有两套系统。

缺点：

- 需要明确切分控制面、数据面、执行面。
- 需要一次性建立策略版本与 promotion 纪律。

结论：

- **推荐方案。**

## 5. 推荐架构总览

推荐终态是一个 **三层平台**：

1. **Trading Substrate**
   平台内部稳定运行的交易/通知/风控底座，不依赖 agent 才能成立。
2. **Research & Control Plane**
   数据仓、回测、评分、策略注册、promotion、审计都在这里。
3. **AI & Human Interaction Plane**
   Telegram、OpenClaw、Codex、Claude、MCP、skills 都只通过受控接口操作平台。

## 6. 核心设计原则

### 6.1 AI 不是执行真相源

AI 负责研究、修改策略、解释结果、准备 promotion。

平台负责：

- 数据真相
- 回测真相
- live state 真相
- 风控真相
- 下单真相

### 6.2 实盘 side effect 必须单点收口

只有一个服务可以向 Bitget 发写请求：

- `execd`

这样可以彻底避免：

- `traderd` 直接写单
- AI runtime 直接下单
- Telegram handler 直接调交易所
- 多个服务竞争订单状态

### 6.3 策略必须版本化、可回放、可晋升

策略不能再只是 config 参数。

策略必须是平台的一等对象：

- 有 ID
- 有 version
- 有 owner
- 有 bundle
- 有 backtest report
- 有 promotion status
- 有 rollback pointer

### 6.4 研究与 live 必须共享同一份策略语义

为避免“研究一套、live 一套”，策略定义采用 **Strategy Bundle**：

- `strategy.yaml`
- `score.cel`
- `gates.cel`
- `risk.yaml`
- `universe.yaml`
- `fixtures/`
- `README.md`

研究侧、回测侧、live trader 都消费同一份 bundle。

## 7. 运行时服务设计

### 7.1 `marketd`

职责：

- Bitget public/private ingress 唯一入口
- 实时 trade/candle/account/position/order/fill 采集
- bootstrap + gap fill
- 写入共享 SQLite/WAL `event_log`
- 同步归档到中央 warehouse

### 7.2 `traderd`

职责：

- 只做 deterministic strategy evaluation
- 消费市场事件
- 读取 active `StrategyBundle`
- 产出 `signal.created`、`risk.state_changed`、`entry.intent.created`

禁止：

- 直接下单
- 持有交易所写权限

### 7.3 `execd`

职责：

- 全平台唯一 Bitget 写通道
- 读取 `entry.intent.created`
- 下单、查单、reconcile、reduce-only close、flatten
- 写 `order.submitted`、`order.filled`、`position.reconciled`

### 7.4 `notifierd`

职责：

- 消费 event_log 与 promotion event
- 直接发送 Telegram Bot API 和 DingTalk 消息
- 发送 live alert、risk alert、promotion alert、daily summary

### 7.5 `platformd`

职责：

- HTTP API
- MCP server
- backtest orchestration
- scoring
- strategy registry
- promotion workflow
- artifact index
- AI write operations guard

### 7.6 `researchd`

职责：

- 管理长任务，例如批量回测、理论研究、AI summary、nightly report
- 调用 OpenAI Responses background mode 或本地 CLI workers
- 不持有交易所写权限

### 7.7 `openclaw`

定位：

- conversational gateway
- Telegram / Discord / Slack 等多 channel agent ingress
- cron / dashboard / skill routing

用途：

- 人通过 Telegram 触发研究任务
- AI 研究结果回送到 Telegram
- 定时日报与 job 状态播报

限制：

- 不作为实盘下单主通道
- 不作为交易真相源

## 8. 数据平面设计

### 8.1 边缘实时事件存储

继续保留当前已证明可用的共享 SQLite/WAL `event_log` 作为 **runtime edge log**。

原因：

- 已经实测通过
- 单 ECS 上简单、快、可恢复
- 对 `marketd/traderd/execd/notifierd` 非常合适
- 交易执行不需要依赖中心数据库写入成功

### 8.2 中央数据仓

引入 `Postgres 17 + TimescaleDB` 作为平台 warehouse。

承载：

- raw bars
- normalized bars
- feature snapshots
- strategy registry
- backtest runs
- promotion requests
- execution journal
- alert ledger
- AI research runs

### 8.3 Parquet / DuckDB 导出层

引入按 symbol / interval / date 分区的 Parquet 导出。

作用：

- 给 `codex` / `claude` / `openclaw` 提供快速可读 dataset
- 给离线分析与批量比较提供高吞吐读取
- 让 AI 可以在不接触 runtime SQLite 的情况下分析长期历史

### 8.4 K 线时间粒度策略

设计：

- `1m` 为 crypto intraday 的基础粒度
- `1h` 为中频基础粒度
- `1d` 为跨资产基础粒度
- 其余粒度由平台聚合生成或按 provider 拉取后归一

第一版硬要求：

- `1m/5m/15m/1h/4h/1d`

## 9. 特征与理论层设计

平台内建且版本化的特征组：

- `wave_structure`
- `level_cluster`
- `fib_confluence`
- `price_action_trigger`
- `volume_confirmation`
- `regime_tags`

要求：

- 每个特征组独立版本化
- 每个特征组有 snapshot fixtures
- 回测与 live 使用相同特征实现
- 特征 materialization 可离线批量回放

## 10. 策略定义设计

### 10.1 Strategy Bundle

每个策略版本目录：

```text
strategies/<strategy_id>/versions/<version>/
  strategy.yaml
  universe.yaml
  score.cel
  gates.cel
  risk.yaml
  fixtures/
  analysis/
  README.md
```

### 10.2 为什么不用随意的 Python live strategy

因为那会让回测和实盘分叉。

推荐方案是：

- 允许 AI 用任何语言做研究
- 但最终上线必须归约成平台可执行的 Strategy Bundle
- live trader 只认 bundle

### 10.3 策略生命周期

```text
draft
-> backtest_passed
-> shadow_running
-> canary_live
-> live_active
-> deprecated / rolled_back
```

## 11. AI 操作面设计

### 11.1 统一 MCP surface

平台提供两个 MCP 面：

#### `quant-read-mcp`

只读工具：

- `market.search_bars`
- `market.fetch_bar_slice`
- `feature.fetch_snapshot`
- `feature.compare_versions`
- `strategy.list_versions`
- `backtest.get_run`
- `docs.fetch_theory`
- `ops.get_positions`
- `ops.get_orders`
- `ops.get_health`

#### `quant-write-mcp`

受控写工具：

- `strategy.clone_version`
- `strategy.update_bundle`
- `backtest.run`
- `promotion.create_request`
- `promotion.start_shadow`
- `promotion.start_canary`
- `ops.halt_strategy`
- `ops.flatten_symbol`

### 11.2 Skill pack

提供一个 umbrella skill：

- `quant-platform-operator`

下挂子技能：

- `quant-theory-study`
- `quant-dataset-scout`
- `quant-strategy-edit`
- `quant-backtest-compare`
- `quant-promotion`
- `quant-live-ops`

### 11.3 AI 权限边界

AI 永远不能：

- 读取交易所 secret 原文
- 直接调用 Bitget 写接口
- 绕过 promotion gate 直接切 live

AI 只能：

- 调研究接口
- 读平台事实
- 改策略 bundle
- 发起受控 promotion request

## 12. Telegram / DingTalk / OpenClaw 设计

### 12.1 Telegram

Telegram 是第一优先级交互入口。

第一版命令集：

- `/status`
- `/positions`
- `/orders`
- `/strategy active`
- `/backtest <strategy_id> <version>`
- `/promote <strategy_id> <version> shadow`
- `/approve <promotion_id>`
- `/halt <strategy_id>`
- `/flatten <symbol>`
- `/research <prompt>`

### 12.2 DingTalk

DingTalk 保持推送渠道：

- risk alerts
- fill alerts
- daily summary
- promotion state changes
- rollback alerts

### 12.3 OpenClaw

OpenClaw 用于：

- Telegram conversational agent ingress
- AI jobs dispatch
- cron-based research / summary
- dashboard / session trace

不用于：

- 交易执行主逻辑
- exchange credentials holder

## 13. 安全与风控设计

### 13.1 密钥分层

- `marketd` 只持有 public keyless + private read creds
- `execd` 持有唯一交易所写权限
- `platformd/researchd/openclaw` 不持有交易所写权限
- AI tools 只看到平台 API，不看到 secret

### 13.2 Promotion gate

任何 live promotion 都必须满足：

- backtest gate 通过
- shadow gate 通过
- canary gate 通过
- 当前账户无异常持仓
- 人工批准通过

### 13.3 Kill switch

全平台必须有：

- symbol flatten
- strategy halt
- exchange write freeze
- notification fanout

## 14. 测试与验证合同

### 14.1 单元与集成

- feature tests
- bundle parser tests
- scoring tests
- promotion state machine tests
- API contract tests
- MCP tool tests

### 14.2 平台级回放

- 用历史 K 线做 deterministic replay
- 用固定 fixture 比较 candidate / order intent / risk state

### 14.3 Bitget 真实验证

必须真实验证：

- public candles
- public trade/candle websocket
- private account/positions/orders
- leverage set
- open market order
- order query
- fill stream
- reduce-only close
- empty position on exit

### 14.4 交互验证

必须真实验证：

- Telegram 入站命令
- Telegram 出站通知
- DingTalk 推送
- OpenClaw agent message delivery
- AI background job callback / polling

## 15. 需要保留与需要淘汰的现有部分

### 15.1 保留并强化

- `marketd`
- `traderd`
- `internal/exchange/bitget/*`
- `internal/core/features.go`
- `internal/store/sqlite/*`
- `scripts/run_mstr_e2e.sh`
- 真实 Bitget tests

### 15.2 重构

- `internal/adapters/data.go`
- `cmd/lab`
- `internal/core/EvaluateSignal` 周围的硬编码策略组合
- `agentd` 的职责边界

### 15.3 退役

- `cmd/stream`
- `internal/adapters/stream.go`

因为它们属于旧的报警脚手架，不再代表平台终态。

## 16. 第一版完整平台的 Definition of Done

当且仅当以下条件全部成立，才算完成：

1. 平台内可以管理历史 K 线与多时间粒度数据。
2. 平台内可以计算并存储版本化特征快照。
3. 平台内可以注册策略 bundle、回测、评分、比较版本。
4. `traderd` 与 `execd` 已经分离，交易 side effect 单点收口。
5. Telegram 与 DingTalk 都能真实交互。
6. OpenClaw 已接入并能触发研究任务。
7. Codex 与 Claude Code 都有可执行的 skill / MCP workflow。
8. Bitget 实盘验证链路全部真实通过，并且以空仓收尾。
9. promotion / shadow / canary / rollback 全流程可操作。
10. AI 不持有交易所写权限，但可以完整驱动研究与 promotion 流程。

## 17. 非目标

第一版不做：

- 图形化 Web 大屏作为必需依赖
- 高频 order book alpha
- 多交易所同步执行
- 自动无人值守 full live promotion

这些都可以后续追加，但不属于第一版必须条件。

## 18. 推荐落地顺序

这是一套完整平台，但实现必须分阶段：

1. 先稳定 substrate。
2. 再统一数据与策略定义。
3. 再补 AI 与 Telegram/OpenClaw 面。
4. 最后做 promotion/canary/live governance。

这个顺序不是保守，而是为了让你得到一个真正可上线、可扩展、可被 AI 驾驭的平台，而不是一个看起来很大、实际全靠人工兜底的半成品。
