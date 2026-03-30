# Operator Quickstart

## What Is Ready

当前已经真实可用的 operator 面有四条：

1. `platformd`
   读取现有 SQLite runtime state，并提供 `health / status / positions / orders / events / bars / features / strategies / backtests / promotions / live flatten` HTTP API。
2. `platformctl`
   作为 operator CLI，读取 `platformd` API，并提供 `status`、`strategy versions`、`historical sync`、`aggregate`、`export parquet`、`backtest run`、`promotion *`、`live flatten`、`notify test`、`warehouse health` 能力。
3. `notifierd`
   消费 `event_log` 中的 `order_fill`、`risk.state_changed`、`promotion.*` 事件，并通过 Telegram / DingTalk 发通知。
4. `execd`
   消费 `trader` source 中的 `entry.intent.created`，独占 Bitget 写路径，并支持 `flatten-symbol` 收尾。

## Remote Paths

- repo:
  `/root/.config/superpowers/worktrees/quant-lab/autoresearch-20260328-all-plan`
- runtime SQLite:
  `var/mstr-e2e-state.db`
- binaries:
  `bin/platformd`
  `bin/platformctl`
  `bin/notifierd`
  `bin/execd`

## Required Env

### Telegram

- `TELEGRAM_BOT_TOKEN`
- `TELEGRAM_ALERT_CHAT_ID`
  如果未设置，`platformctl notify test` 与 `notifierd` 会回退到 `TELEGRAM_COMMAND_CHAT_ID`
- `TELEGRAM_COMMAND_CHAT_ID`

### DingTalk

- `DINGTALK_WEBHOOK`
- `DINGTALK_SECRET`

### Bitget

- `BITGET_API_KEY`
- `BITGET_API_SECRET`
- `BITGET_PASSPHRASE`

## Build

```bash
cd /root/.config/superpowers/worktrees/quant-lab/autoresearch-20260328-all-plan
PATH=/usr/local/go/bin:/usr/bin:/bin go build -o ./bin/platformd ./cmd/platformd
PATH=/usr/local/go/bin:/usr/bin:/bin go build -o ./bin/platformctl ./cmd/platformctl
PATH=/usr/local/go/bin:/usr/bin:/bin go build -o ./bin/notifierd ./cmd/notifierd
PATH=/usr/local/go/bin:/usr/bin:/bin go build -o ./bin/execd ./cmd/execd
```

## Start platformd

默认配置来自 `configs/demo-mstr-e2e.yaml`，其中 state db 是 `var/mstr-e2e-state.db`。当前这台 ECS 的 `127.0.0.1:8080` 被 SearXNG 占用，operator 入口固定使用 `127.0.0.1:18080`。

```bash
cd /root/.config/superpowers/worktrees/quant-lab/autoresearch-20260328-all-plan
./bin/platformd -config configs/demo-mstr-e2e.yaml -execd-path ./bin/execd -listen 127.0.0.1:18080
```

Smoke:

```bash
curl -fsS http://127.0.0.1:18080/health
./bin/platformctl status -addr http://127.0.0.1:18080
./bin/platformctl strategy versions -addr http://127.0.0.1:18080 -strategy mstr-wave-fib
./bin/platformctl positions -addr http://127.0.0.1:18080
./bin/platformctl orders -addr http://127.0.0.1:18080 --limit 20
./bin/platformctl events -addr http://127.0.0.1:18080 --limit 20
curl -fsS "http://127.0.0.1:18080/api/bars?config_path=configs/demo-mstr-e2e.yaml&dataset=mstrusdt_demo_replay"
curl -fsS "http://127.0.0.1:18080/api/features?config_path=configs/demo-mstr-e2e.yaml&dataset=mstrusdt_demo_replay&offset=0"
./bin/platformctl backtest run -addr http://127.0.0.1:18080 -config configs/demo-mstr-bundle.yaml
./bin/platformctl promotion request -addr http://127.0.0.1:18080 -strategy mstr-wave-fib -version v0.1.2 -config configs/demo-mstr-bundle.yaml
```

## Promotion Flow

基于同一个 `platformd`，当前可以走通的 promotion 状态机命令是：

```bash
cd /root/.config/superpowers/worktrees/quant-lab/autoresearch-20260328-all-plan

promo_id=$(./bin/platformctl promotion request \
  -addr http://127.0.0.1:18080 \
  -strategy mstr-wave-fib \
  -version v0.1.2 \
  -config configs/demo-mstr-bundle.yaml | jq -r '.id')

./bin/platformctl promotion start-shadow -addr http://127.0.0.1:18080 -id "$promo_id"
./bin/platformctl promotion pass-shadow -addr http://127.0.0.1:18080 -id "$promo_id"
./bin/platformctl promotion start-canary -addr http://127.0.0.1:18080 -id "$promo_id"
./bin/platformctl promotion approve -addr http://127.0.0.1:18080 -id "$promo_id"
./bin/platformctl promotion get -addr http://127.0.0.1:18080 -id "$promo_id"
```

## Send Test Notifications

```bash
cd /root/.config/superpowers/worktrees/quant-lab/autoresearch-20260328-all-plan

TELEGRAM_BOT_TOKEN=... \
TELEGRAM_ALERT_CHAT_ID=... \
./bin/platformctl notify test --channel telegram --message "platformctl telegram test"

DINGTALK_WEBHOOK=... \
DINGTALK_SECRET=... \
./bin/platformctl notify test --channel dingtalk --message "platformctl dingtalk test"

TELEGRAM_BOT_TOKEN=... \
TELEGRAM_ALERT_CHAT_ID=... \
DINGTALK_WEBHOOK=... \
DINGTALK_SECRET=... \
./bin/platformctl notify test --channel all --message "platformctl all-channel test"
```

## Run notifierd

持续运行：

```bash
cd /root/.config/superpowers/worktrees/quant-lab/autoresearch-20260328-all-plan
TELEGRAM_BOT_TOKEN=...
TELEGRAM_ALERT_CHAT_ID=...
DINGTALK_WEBHOOK=...
DINGTALK_SECRET=...
./bin/notifierd -config configs/demo-mstr-e2e.yaml
```

一次性消费现有事件并退出：

```bash
cd /root/.config/superpowers/worktrees/quant-lab/autoresearch-20260328-all-plan
TELEGRAM_BOT_TOKEN=...
TELEGRAM_ALERT_CHAT_ID=...
DINGTALK_WEBHOOK=...
DINGTALK_SECRET=...
./bin/notifierd -state-db /tmp/notifier-smoke.db -once
```

## Run execd

持续消费 `entry.intent.created`：

```bash
cd /root/.config/superpowers/worktrees/quant-lab/autoresearch-20260328-all-plan
BITGET_API_KEY=...
BITGET_API_SECRET=...
BITGET_PASSPHRASE=...
./bin/execd -config configs/demo-mstr-e2e.yaml
```

一次性消费当前 intent 并退出：

```bash
cd /root/.config/superpowers/worktrees/quant-lab/autoresearch-20260328-all-plan
BITGET_API_KEY=...
BITGET_API_SECRET=...
BITGET_PASSPHRASE=...
./bin/execd -config configs/demo-mstr-e2e.yaml -once
```

主动平仓收尾：

```bash
cd /root/.config/superpowers/worktrees/quant-lab/autoresearch-20260328-all-plan
BITGET_API_KEY=...
BITGET_API_SECRET=...
BITGET_PASSPHRASE=...
./bin/execd -config configs/demo-mstr-e2e.yaml -flatten-symbol MSTRUSDT
```

## Event Kinds notifierd Currently Handles

- `order_fill`
  生成 `成交回报`
- `risk.state_changed`
  当 `to=degraded|halted` 时生成 `风控停止`
- `promotion.approved`
  生成 `上线审批通过`
- `promotion.canary_degraded`
  生成 `canary 降级`

## Real Verification Completed On 2026-03-29

- `platformd` 在 `127.0.0.1:18080` 启动成功
- `curl /health` 返回 `{"ok":true}`
- `platformctl status` 返回 `last_seq=624`
- `platformctl positions` 返回真实仓位快照：
  `BABAUSDT=12`
  `CRCLUSDT=20`
  `HOODUSDT=2`
- `platformctl events --limit 2` 返回真实 `MSTRUSDT trade_tick`
- `platformd` 在 `127.0.0.1:18081` 上真实跑通 `backtest run`
  并返回 `MSTRUSDT` 720 根 `1m` bars 的报告与评分
- `platformd` 在 `127.0.0.1:18082` 上真实跑通 promotion 主链：
  `promotion.requested -> promotion.shadow_started -> promotion.shadow_passed -> promotion.canary_started -> promotion.approved`
- 同一 promotion smoke db 中，`sqlite3 /tmp/promotion-smoke.db "select consumer_key, last_seq from consumer_cursor"` 返回 `notifierd|5`
  证明 `notifierd -once` 已消费 promotion 事件并完成真实外发
- `platformd` 在 `127.0.0.1:18083` 上真实返回
  `/api/bars` = `mstrusdt_demo_replay / MSTRUSDT / 1m / 720 / last_close=126.06`
  `/api/features` = `MSTRUSDT / index=719 / bar_time=2026-03-29T06:35:00Z`
- `platformctl notify test --channel telegram` 返回成功
- `platformctl notify test --channel dingtalk` 返回成功
- `notifierd -once` 基于临时 smoke db 消费 `risk.state_changed`
  并成功把 `consumer_cursor` 写成 `notifierd|1`
- `execd` 已真实通过 `TestRealExecutionRuntimeIntentLifecycle`
  - `MSTRUSDT last_price=126.2200`
  - 最小有效 size=`0.04`
  - 真实开仓 -> 查单 -> `execution.reconciled` -> flatten -> 最终空仓

## Current Notes

- Telegram command ingress 已在 `2026-03-30 10:48 CST` 完成真实验证：`/status`、`/positions`、`/backtest mstr-wave-fib v0.1.2` 都已由 direct chat 触发，`consumer_cursor.telegram.command=140596841`，`getUpdates=[]`。
- 当前 ownership 已固定：平台 bot 的 Telegram inbound 由 `notifierd` 独占，`/home/admin/.openclaw/openclaw.json` 已切到 `channels.telegram.enabled=false`；OpenClaw 保留 DingTalk / job gateway。
- `/backtest` Telegram reply 已改成 Telegram-safe summary，避免 `Bad Request: message is too long`。回归测试：`go test ./internal/platform/notifier -run TestCommandRuntimeRepliesToBacktestWithTelegramSafeSummary -count=1 -v`。
- `platformctl` 已提供 `strategy versions`、`historical sync`、`aggregate`、`export parquet`、`live flatten`、`warehouse health` 等 control-plane 命令；`platformd` 也已提供 `/api/strategies/versions` 与 `/api/live/flatten`。
- Codex / Claude / OpenClaw skill smoke 已可运行；`scripts/run_platform_skill_smoke.sh` 当前会额外保存 OpenClaw raw log，并提取结构化 JSON artifact。
- 当前运行态仍存在一个 operator 风险：`platformctl status` 持续显示 `execd` cursor 停在 `2059`，`trader.arming_state=degraded`，这属于运行态排查项，不是功能缺项。

## 2026-03-30 Supplement

- `notifierd` 的 systemd `ExecStart` 已固定 `-health-addr 127.0.0.1:18081`；fresh `curl -fsS http://127.0.0.1:18081/health` 返回 `ok=true`，并包含 `telegram_command.enabled=true` 与 `daily_summary.location_name=Asia/Shanghai`。
- Telegram poll 的 transient `409 / timeout` 已在 runtime 中改为 retry + backoff，不再把 poll error 当 fatal error 直接退出进程。fresh systemd 状态为 `ActiveState=active`、`SubState=running`。
- `2026-03-30 10:48 CST` 真实 Telegram direct chat `/status`、`/positions`、`/backtest mstr-wave-fib v0.1.2` 已通过；复核命令：`sqlite3 -header -column var/mstr-e2e-state.db "select * from consumer_cursor where consumer_key='telegram.command';"` 与 `curl -fsS "https://api.telegram.org/bot$TELEGRAM_BOT_TOKEN/getUpdates?timeout=1" | jq '.'`。
- `researchd` 的 systemd oneshot 入口已经实装，启动前必须准备 `/etc/quantlab/researchd/default.env`，当前已验证的内容是：
  - `RESEARCHD_ARGS=--kind nightly_report --strategy mstr-wave-fib --config-path configs/demo-mstr-bundle.yaml --datasets MSTRUSDT:1h,BTCUSDT:1h`
- 当前固定启动顺序：
  1. `systemctl daemon-reload`
  2. `systemctl start platform.target`
  3. `systemctl start quantlab-researchd@default`
  4. `curl -fsS http://127.0.0.1:18080/health`
  5. `curl -fsS http://127.0.0.1:18081/health`
- fresh 证据路径：
  - `artifacts/platform/final-verify/20260329T175723Z`
  - `artifacts/platform/e2e/20260329T175752Z`
  - `artifacts/platform/skill-smoke/20260329T180906Z`
