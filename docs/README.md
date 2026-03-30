# Quant Platform Docs

## 功能说明

这个仓库当前交付的是一个可在 ECS 上真实运行的 AI-first quant platform，固定边界如下：

- `marketd` 负责交易所行情 ingress。
- `traderd` 只做 deterministic evaluation。
- `execd` 是唯一 Bitget write path。
- `platformd` 提供 operator HTTP control plane。
- `platformctl` 提供 operator CLI。
- `notifierd` 负责 Telegram / DingTalk 通知与 Telegram inbound command。
- `mcpd` 为 AI 暴露 MCP control plane。
- `researchd` 为 Codex / Claude 执行长任务 research workflow。
- `OpenClaw` 只做 conversational / delivery gateway，不拥有交易所写权限。

## 使用手册

推荐按下面顺序阅读并执行：

1. `docs/ops/operator-quickstart.md`
   先完成 build、启动、health check、status、strategy versions、backtest、promotion、notify、flatten 等标准操作。
2. `docs/ops/warehouse-setup.md`
   初始化 PostgreSQL / TimescaleDB、migration、health check。
3. `docs/ops/historical-sync.md`
   运行 historical sync、aggregate、parquet export。
4. `docs/ops/strategy-bundle-authoring.md`
   编写或修改 `strategies/<strategy_id>/versions/<version>` bundle。
5. `docs/ops/promotion-governance.md`
   推进 `request -> shadow -> canary -> approve -> rollback`。
6. `docs/ops/openclaw-setup.md`
   配置 OpenClaw channel、plugin allowlist 和 delivery path。
7. `docs/ops/codex-claude-integration.md`
   用 Codex / Claude 通过 repo-local skills 走 research / backtest workflow。

## 最佳实践

- `docs/ops/platform-operator-best-practices.md`
  这里记录了远端操作边界、Telegram ownership、Bitget real verification、GitHub push 边界和故障排查主线。
- `docs/architecture/control-plane-boundaries.md`
  这里定义了 `platformd / mcpd / OpenClaw / execd` 的 trust boundary。
- `docs/architecture/runtime-event-contract.md`
  这里定义了 `traderd -> execd -> notifierd` 的 runtime event contract。

## 快速索引

- 最终 E2E 报告：`plan/2026-03-29-platform-e2e-report-cn.md`
- Operator quickstart：`docs/ops/operator-quickstart.md`
- Operator best practices：`docs/ops/platform-operator-best-practices.md`
- OpenClaw setup：`docs/ops/openclaw-setup.md`
- Strategy bundle authoring：`docs/ops/strategy-bundle-authoring.md`
- Promotion governance：`docs/ops/promotion-governance.md`
- Codex / Claude integration：`docs/ops/codex-claude-integration.md`
