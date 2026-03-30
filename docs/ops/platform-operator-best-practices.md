# Quant Platform Operator Best Practices

## 实例概览

| 项目 | 值 |
| --- | --- |
| ECS | `47.250.138.143` |
| 代码仓库 | `/root/.config/superpowers/worktrees/quant-lab/autoresearch-20260328-all-plan` |
| 当前分支 | `autoresearch/20260328-all-plan` |
| 平台 API | `http://127.0.0.1:8080` |
| 通知 Health | `http://127.0.0.1:18081/health` |
| MCP bridge | `127.0.0.1:18082` |
| 运行时 SQLite | `var/mstr-e2e-state.db` |
| Telegram operator chat | `6959476905` |
| 真实验证策略 | `mstr-wave-fib` |
| 真实验证版本 | `v0.1.2` |
| Telegram command cursor | `140596841` |
| 回测证据 | `artifacts/mstr-bundle-v0.1.2/backtest-result.json` |

本实例的真实结果如下：

- `platform.target` 已是 `active`，`quantlab-marketd/traderd/execd/platformd/mcpd/notifierd` 均为 `active/running`。
- `@ottomoneyfree_bot` 的 Telegram inbound ownership 已固定到 `notifierd`，`OpenClaw` 不再持有同一 bot token 的 `getUpdates`。
- 真实 direct chat 已完成 `/status`、`/positions`、`/backtest mstr-wave-fib v0.1.2` 三条命令验证，`consumer_cursor.telegram.command=140596841`，`getUpdates=[]`。
- `artifacts/mstr-bundle-v0.1.2/backtest-result.json` 的修改时间是 `2026-03-30 10:48:50 CST`，与真实 `/backtest` 验证时刻对齐。

## 操作主线

### 第一步：先确认边界，再碰系统

执行命令：

```bash
cd /root/.config/superpowers/worktrees/quant-lab/autoresearch-20260328-all-plan
git branch --show-current
git status --short --branch
systemctl show platform.target --property=ActiveState,SubState --no-pager
systemctl show quantlab-marketd quantlab-traderd quantlab-execd quantlab-platformd quantlab-mcpd quantlab-notifierd --property=Id,ActiveState,SubState --no-pager
```

从这一步可以得知：

- 所有真实操作都必须在 ECS 远端仓库中执行，不能把本机目录当成执行面。
- operator 先确认当前分支和工作区状态，再决定是否继续修改、验证或推送。
- `platform.target` 代表整个平台编排状态，单个 service 只是子视角。
- 若 `git status` 仍有未提交改动，后续验证与 GitHub 推送必须基于当前现场，不要假设仓库是干净的。

### 第二步：按控制面入口工作，不要跨边界

执行命令：

```bash
curl -fsS http://127.0.0.1:8080/health
curl -fsS http://127.0.0.1:8080/api/status | jq '.'
curl -fsS http://127.0.0.1:8080/api/dashboard | jq '.generated_at, (.symbols | length), (.alerts | length)'
./bin/platformctl status -addr http://127.0.0.1:8080
./bin/platformctl positions -addr http://127.0.0.1:8080
./bin/platformctl events -addr http://127.0.0.1:8080 --limit 20
```

从这一步可以得知：

- `platformd` 是 operator 的统一 HTTP 控制面，不需要直接翻 SQLite 才能看状态。
- 浏览器入口 `/` 与 JSON 入口 `/api/dashboard` 现在也是 control-plane 的一部分，watchlist、最新价格、daily features、active alerts 都应先从这里观察。
- `platformctl` 是 operator CLI，只应通过 `platformd` 暴露出来的控制面动作工作。
- `marketd -> traderd -> execd -> notifierd` 的职责边界是固定的，任何读写穿透都属于反模式。
- `execd` 仍是唯一 Bitget 写路径，`platformd` 和 AI 面都不能绕过它直接写交易所。

### 第三步：把 Telegram 当成 operator command 面，不要再让 OpenClaw 抢 owner

执行命令：

```bash
curl -fsS http://127.0.0.1:18081/health | jq '.'
sqlite3 -header -column var/mstr-e2e-state.db "select * from consumer_cursor where consumer_key='telegram.command';"
curl -fsS "https://api.telegram.org/bot$TELEGRAM_BOT_TOKEN/getUpdates?timeout=1" | jq '.'
```

从这一步可以得知：

- `telegram_command.enabled=true` 是平台 bot command 面存活的必要条件。
- `consumer_cursor.telegram.command` 是 Telegram inbound command 的真实消费凭证。
- `getUpdates=[]` 才表示 pending update 已清空，命令没有卡在 reply path。
- `OpenClaw` 只保留 DingTalk / job gateway；平台 bot 的 Telegram inbound owner 固定是 `notifierd`。

### 第四步：真实 `/backtest` 只回摘要，不回整块 JSON

执行命令：

```bash
./bin/platformctl backtest run -addr http://127.0.0.1:18080 -config configs/demo-mstr-bundle-v0.1.2.yaml
stat artifacts/mstr-bundle-v0.1.2/backtest-result.json
```

从这一步可以得知：

- 平台 API `POST /api/backtests/run` 会生成完整 JSON 报告与 Markdown 报告，文件证据在 `artifacts/`。
- Telegram reply 只应该回摘要字段，例如 `final_score`、`objective_score`、`avg_oos_return`、`avg_trade_count`，而不是整块 `28 KB` JSON。
- 真实 `/backtest` 的 operator 验证要同时看 Telegram reply 和 `artifacts/mstr-bundle-v0.1.2/backtest-result.json` 的修改时间。
- 若 Telegram reply 再次出现超长问题，先复查 `internal/platform/notifier/telegram_command.go` 的 summary path，而不是怀疑 platform API 本体。

### 第五步：真实 Bitget 写路径必须围绕同一条主线

执行命令：

```bash
env RUN_BITGET_REAL=1 \
BITGET_API_KEY="$BITGET_API_KEY" \
BITGET_API_SECRET="$BITGET_API_SECRET" \
BITGET_PASSPHRASE="$BITGET_PASSPHRASE" \
PATH=/usr/local/go/bin:/usr/bin:/bin \
go test ./internal/execution -run TestRealExecutionRuntimeIntentLifecycle -count=1 -v
```

从这一步可以得知：

- 真实 Bitget 主线固定是 `MSTRUSDT / USDT-FUTURES / isolated / leverage=3 / long`。
- 验证必须以空仓开场，以空仓收尾。
- 最小有效 size 不是 `0.01`，当前主线下的真实最小有效 size 是 `0.04`。
- 任何 live 写路径问题，优先复查 `execd` 和 `internal/execution`，不要把写权限回塞到 `traderd`。

### 第六步：OpenClaw 只做 conversational gateway，不做平台 bot owner

执行命令：

```bash
sudo -u admin \
  HOME=/home/admin \
  XDG_CONFIG_HOME=/home/admin/.config \
  XDG_STATE_HOME=/home/admin/.local/state \
  XDG_DATA_HOME=/home/admin/.local/share \
  PATH=/usr/local/bin:/usr/bin:/bin \
  /home/admin/.local/share/pnpm/openclaw agent --local --to +15555550123 --message "platform status" --json
```

从这一步可以得知：

- OpenClaw 读取平台状态时，应通过 `platformd` 或 `platformctl`，而不是自己变成状态真相源。
- OpenClaw 可以把 job 结果发回 Telegram，但不能重新拥有同一个 bot token 的 inbound command ownership。
- OpenClaw 不持有 Bitget 写密钥，这个边界必须长期固定。
- 若以后要恢复 OpenClaw Telegram outbound，先确认 `channels.telegram.enabled` 不会重新抢占 `getUpdates`。

### 第七步：推 GitHub 只推源码与安全文档，不推敏感现场

执行命令：

```bash
git status --short --branch
git diff --cached --name-only
git ls-files --others --exclude-standard
```

从这一步可以得知：

- 应提交：源码、配置、skill、架构文档、ops 文档、脚本、测试。
- 不应提交：`bin/`、`var/`、运行产物、状态库、日志、一次性 artifact。
- 不应提交：带明文密钥的敏感现场文档，例如 `workspace/core.md` 一类 durable memory。
- 任何 GitHub 推送前都必须先做 build/test 复核，再看 staged file list 是否与请求范围一致。

## 服务与 ownership 速查表

| 对象 | 唯一职责 | 不允许做的事 |
| --- | --- | --- |
| `marketd` | 交易所 ingress，写入 `market.*` | 直接做策略决策 |
| `traderd` | deterministic evaluation，产出 `entry.intent.created` | 直接下单、查单、flatten |
| `execd` | 唯一 Bitget 写路径 | 被 AI 或 OpenClaw 绕过 |
| `platformd` | operator HTTP API | 直接持有交易所写权限 |
| `mcpd` | AI tool control plane | 直接拿 exchange creds |
| `notifierd` | 事件通知、Telegram inbound command、insights DingTalk alert delivery | 抢占 OpenClaw job gateway 语义 |
| `OpenClaw` | conversational / delivery gateway | 变成平台 Telegram bot inbound owner |

## 每日操作顺序

| 顺序 | 操作 | 验证 |
| --- | --- | --- |
| 1 | 进入远端 repo，确认 branch/status | `git branch --show-current` `git status --short --branch` |
| 2 | 确认 systemd 与 health | `systemctl show ...` `curl /health` |
| 3 | 确认 Telegram command 面 | `consumer_cursor.telegram.command` `getUpdates=[]` |
| 4 | 做 control-plane 读操作 | `platformctl status/positions/events` |
| 5 | 做 backtest / promotion | `platformctl backtest run` `platformctl promotion ...` |
| 6 | 做真实 Bitget 写链路前先确认空仓 | `TestRealExecutionRuntimeIntentLifecycle` 或等价 flatten 检查 |
| 7 | 推 GitHub 前先全量验证 | `go test ./... -count=1` `go build ./cmd/...` |

## 故障排查速查表

| 现象 | 先看哪里 | 根因方向 |
| --- | --- | --- |
| Telegram command 不消费 | `http://127.0.0.1:18081/health` `consumer_cursor.telegram.command` `getUpdates` | owner 冲突、reply path 卡住、chat allowlist 不匹配 |
| `/backtest` 执行了但 bot 不回 | `artifacts/*/backtest-result.json` `getUpdates` | Telegram reply 超长、cursor 未提交 |
| live 写路径异常 | `go test ./internal/execution -run TestRealExecutionRuntimeIntentLifecycle -v` | `execd`、Bitget contract、最小有效 size |
| OpenClaw 行为异常 | `openclaw-gateway` journal `openclaw.json` | channel 配置漂移、plugin allowlist、ownership 误回流 |
| GitHub push 失败 | `git remote -v` `git status` `git diff --cached --name-only` | 选错 key、staged 集合不对、non-fast-forward |

## 代码与文档入口

| 入口 | 路径 |
| --- | --- |
| Telegram command runtime | `internal/platform/notifier/telegram_command.go` |
| Telegram backtest summary 回归测试 | `internal/platform/notifier/telegram_command_test.go` |
| control plane 边界 | `docs/architecture/control-plane-boundaries.md` |
| runtime 事件契约 | `docs/architecture/runtime-event-contract.md` |
| operator quickstart | `docs/ops/operator-quickstart.md` |
| OpenClaw setup | `docs/ops/openclaw-setup.md` |
| promotion governance | `docs/ops/promotion-governance.md` |
| 最终 E2E 报告 | `plan/2026-03-29-platform-e2e-report-cn.md` |

## 最后规则

- 先在 ECS 远端读、改、测，再谈完成。
- 先看 `platformd/notifierd/execd` 的边界，再写任何新功能。
- Telegram bot 只能有一个 inbound owner。
- 真实 Bitget 验证必须最终空仓。
- GitHub 推送只推源码与安全文档，不推明文密钥与运行现场。
