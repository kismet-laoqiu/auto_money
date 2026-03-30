# 2026-03-29 Platform E2E Report

## 结论

截至 `2026-03-30 02:18 CST`，`quant-lab` 在 ECS `47.250.138.143` 上已经完成 plan 中规划的平台代码、systemd / compose、warehouse、historical ingest、aggregation/export、feature materialization、strategy bundle、backtest/scoring/promotion、notification plane、platform API/MCP/OpenClaw、researchd、Codex/Claude skill、以及真实 Bitget write path 的实现与 fresh 验证。

本轮 fresh 验证全部通过的证据根目录是：

- `artifacts/platform/final-verify/20260329T175723Z`

本轮 fresh smoke 产物是：

- `artifacts/platform/e2e/20260329T175752Z`
- `artifacts/platform/skill-smoke/20260329T180906Z`

全部 134 个任务现已完成并完成 fresh verification；`P12-08 Telegram inbound command` 已在 `2026-03-30 10:48 CST` 完成真实用户侧 `/status`、`/positions`、`/backtest mstr-wave-fib v0.1.2` 入站验证。

## Fresh Verification

### Build / Test

- `PATH=/usr/local/go/bin:/usr/bin:/bin go test ./... -count=1`
  - 结果：`RC=0`
  - 证据：`artifacts/platform/final-verify/20260329T175723Z/go-test.log`
- `PATH=/usr/local/go/bin:/usr/bin:/bin go build ./cmd/lab ./cmd/marketd ./cmd/traderd ./cmd/agentd ./cmd/execd ./cmd/notifierd ./cmd/platformctl ./cmd/platformd ./cmd/mcpd ./cmd/researchd`
  - 结果：`RC=0`
  - 证据：`artifacts/platform/final-verify/20260329T175723Z/go-build.log`

### Warehouse / Historical / Export

- `docker compose -f deploy/docker-compose.platform.yml ps`
  - 结果：`quantlab-warehouse-db` healthy，映射 `127.0.0.1:55432->5432`
  - 证据：`artifacts/platform/final-verify/20260329T175723Z/docker-compose-platform-ps.log`
- `./bin/platformctl warehouse health -config configs/platform/warehouse.yaml`
  - 结果：`ok=true`，`database=quantlab`，`version=17.7`
  - 证据：`artifacts/platform/final-verify/20260329T175723Z/warehouse-health-fresh.json`
- `platformctl historical sync` 对全 `live_symbol_allowlist` 的 15 个 symbol 做 `1h` 首轮同步
  - 结果：15 个 dataset 全部成功，summary 中列出 `BTCUSDT ... DOGEUSDT`
  - 证据：`artifacts/platform/historical-sync/20260329T181600Z/20260329T181818Z/summary.json`
  - 摘要：`artifacts/platform/final-verify/20260329T175723Z/historical-allowlist-summary.json`
- 同一窗口立即重跑一次历史同步
  - 结果：所有 dataset `inserted=0`，证明幂等成立
  - 证据：`artifacts/platform/final-verify/20260329T175723Z/historical-allowlist-rerun.log`
- `platformctl export parquet` 对 `BTCUSDT/ETHUSDT/SOLUSDT/MSTRUSDT` 做首轮导出
  - 结果：`total_files=12`，`total_rows=216`，四个 symbol 各 3 个 parquet 分区文件
  - 证据：`artifacts/platform/export/20260329T181600Z/20260329T181820Z/summary.json`
  - 补充计数：`artifacts/platform/final-verify/20260329T175723Z/export-core4-counts.json`
- DuckDB manifest 已生成
  - 证据：`artifacts/platform/export/20260329T181600Z/20260329T181820Z/manifest.sql`
  - 说明：当前 ECS 上无 `duckdb` CLI，也无 Python `duckdb` module；本轮未追加本机 DuckDB count，但导出 summary / manifest / parquet 路径都已完整落地。

### Platform E2E / Skill Smoke

- `./scripts/run_platform_e2e.sh`
  - 结果：`RC=0`
  - 产物：`artifacts/platform/e2e/20260329T175752Z`
  - 摘要：`final_score=-109.52527002520578`，`objective_score=-136.90658753150723`，promotion request `state=backtest_passed`，version=`v0.1.2`
- `./scripts/run_platform_skill_smoke.sh`
  - 结果：`RC=0`
  - 产物：`artifacts/platform/skill-smoke/20260329T180906Z`
  - 内容：`codex.txt`、`claude.txt`、`openclaw.json`、`openclaw-deliver.json`
  - 说明：这条 smoke 不是“卡死”，而是耗时长。`Codex` 与 `OpenClaw` 都会消耗明显的 wall clock 时间，整条链路完成时间接近 5 分钟。

### Systemd / Health / Ports

- `systemctl daemon-reload && systemctl start platform.target && systemctl start quantlab-researchd@default`
  - 结果：`RC=0`
  - `platform.target`：`ActiveState=active`，`SubState=active`
  - `quantlab-marketd/traderd/execd/platformd/mcpd/notifierd`：全部 `active/running`
  - `quantlab-researchd@default`：`Result=success`，`ExecMainStatus=0`，oneshot 完成后 `inactive/dead`
  - 证据：`artifacts/platform/final-verify/20260329T175723Z/systemd.log`
- 监听端口
  - `127.0.0.1:18080 -> platformd`
  - `127.0.0.1:18081 -> notifierd`
  - `127.0.0.1:18082 -> socat(mcpd bridge)`
  - 证据：`artifacts/platform/final-verify/20260329T175723Z/ss-health-ports.log`

### Notifier / Researchd

- `notifierd` 的 Telegram poll transient error retry 已修复
  - 行为：`telegram status=409 / timeout` 不再作为 fatal error 触发进程退出，而是 backoff 后继续 poll
  - 代码：`internal/platform/notifier/telegram_command.go`
  - 测试：`internal/platform/notifier/telegram_command_test.go`
- 当前 `quantlab-notifierd` systemd 状态
  - `NRestarts=15`，`ActiveState=active`，`SubState=running`
  - `curl -fsS http://127.0.0.1:18081/health` 返回 `ok=true`
  - `telegram_command.enabled=true`
  - `daily_summary.location_name=Asia/Shanghai`
- `researchd@default`
  - 必需 env：`/etc/quantlab/researchd/default.env`
  - 当前内容：`RESEARCHD_ARGS=--kind nightly_report --strategy mstr-wave-fib --config-path configs/demo-mstr-bundle.yaml --datasets MSTRUSDT:1h,BTCUSDT:1h`
  - oneshot 启动成功后自然退出，符合设计

### AI Bundle Edit Results

- Codex 真实链路
  - candidate version：`v0.1.3`
  - 变更：`cooldown_bars 3 -> 5`
  - 结果：`final_score=-103.10085898959917`，相对 `v0.1.2` 变差
  - 证据：`artifacts/platform/ai-edits/codex-v0.1.3.txt`
- Claude 真实链路
  - candidate version：`v0.1.4`
  - 变更：`signal_threshold 3.00 -> 3.50`
  - 结果：`final_score -100.10 -> -99.32`，`objective_score -125.12 -> -124.15`，`avg_trade_count 16 -> 12`
  - 证据：`artifacts/platform/ai-edits/claude-v0.1.4.txt`

### Real Bitget Write Path

- `env RUN_BITGET_REAL=1 RUN_BITGET_CLEANUP=1 go test ./internal/exchange/bitget -run 'TestRealBitget(EnsureFlatPosition|OrderLifecycleAndPrivateStream)' -count=1 -v`
  - 结果：`RC=0`
  - 关键输出：`MSTRUSDT last_price=125.7900 size=0.04`
  - 证据：`artifacts/platform/final-verify/20260329T175723Z/bitget-real.log`
- `env RUN_BITGET_REAL=1 RUN_BITGET_CLEANUP=1 go test ./internal/execution -run TestRealExecutionRuntimeIntentLifecycle -count=1 -v`
  - 结果：`RC=0`
  - 关键输出：`real execd preflight: symbol=MSTRUSDT last_price=125.7900 size=0.04`
  - 证据：`artifacts/platform/final-verify/20260329T175723Z/execution-real.log`
- 结论：真实 open order -> query -> private stream / reconcile -> reduce-only close 链路 fresh 通过，且以空仓收尾。

## P12-08 Telegram Inbound Command Completed

`2026-03-30 10:48 CST` 已完成真实 Telegram inbound command 闭环验证。

真实证据：

- ownership 已统一到平台侧：`/home/admin/.openclaw/openclaw.json` 现为 `channels.telegram.enabled=false`，`quantlab-notifierd` 成为同一 `TELEGRAM_BOT_TOKEN` 的唯一 `getUpdates` owner
- `curl -fsS http://127.0.0.1:18081/health` 返回 `telegram_command.enabled=true`、`allowed_chat_id=6959476905`、`platform_base_url=http://127.0.0.1:18080`
- 真实 direct chat 已依次发送 `/status`、`/positions`、`/backtest mstr-wave-fib v0.1.2`
- `sqlite3 -header -column var/mstr-e2e-state.db "select * from consumer_cursor where consumer_key='telegram.command';"` 返回 `last_seq=140596841`
- `curl -fsS "https://api.telegram.org/bot$TELEGRAM_BOT_TOKEN/getUpdates?timeout=1" | jq '.'` 返回空数组，说明 pending update 已清空
- `artifacts/mstr-bundle-v0.1.2/backtest-result.json` 修改时间为 `2026-03-30 10:48:50 CST`，与真实 `/backtest` 验证时刻对齐

根因与修复：

- 之前的真实阻断一是 OpenClaw 与 `notifierd` 双轮询同一 bot token，二是 `/api/backtests/run` 的回复体约 `28 KB`，直接回 Telegram 会触发 `Bad Request: message is too long`
- 已完成修复：
  - 平台 bot inbound ownership 固定到 `notifierd`
  - `internal/platform/notifier/telegram_command.go` 把 `/backtest` reply 改为 Telegram-safe summary，并对 command reply 增加统一长度兜底
  - 回归测试 `TestCommandRuntimeRepliesToBacktestWithTelegramSafeSummary` 已加入并通过

## Operator SOP

最终 operator 启动顺序固定为：

1. `systemctl daemon-reload`
2. `systemctl start platform.target`
3. `systemctl start quantlab-researchd@default`
4. `curl -fsS http://127.0.0.1:18080/health`
5. `curl -fsS http://127.0.0.1:18081/health`
6. `./scripts/run_platform_e2e.sh`
7. `./scripts/run_platform_skill_smoke.sh`
8. 需要真实交易闭环时，再运行 `bitget-real.log` 与 `execution-real.log` 对应测试

## Final Verdict

- 代码实现：完成
- Fresh build/test：完成
- Warehouse / historical / export：完成
- Bundle / backtest / scoring / promotion：完成
- Telegram / DingTalk / notifier：完成
- Platform API / MCP / OpenClaw：完成
- Researchd / Codex / Claude：完成
- Systemd / orchestration / health ports：完成
- 真实 Bitget write path：完成
- Telegram inbound command：完成
