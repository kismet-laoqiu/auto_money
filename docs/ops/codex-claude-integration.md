# Codex Claude Integration

## Repo Skill Pack

当前仓库已经提供以下 repo-local skills：

- `.agents/skills/quant-workflow/SKILL.md`
- `.agents/skills/quant-plan/SKILL.md`
- `.agents/skills/quant-plan-review/SKILL.md`
- `.agents/skills/quant-coding/SKILL.md`
- `.agents/skills/quant-code-review/SKILL.md`
- `.agents/skills/quant-test/SKILL.md`
- `.agents/skills/quant-changelog/SKILL.md`
- `.agents/skills/quant-docs/SKILL.md`
- `.agents/skills/quant-retrospective/SKILL.md`
- `.agents/skills/quant-platform-operator/SKILL.md`
- `.agents/skills/quant-theory-study/SKILL.md`
- `.agents/skills/quant-backtest-compare/SKILL.md`
- `.agents/skills/quant-promotion/SKILL.md`

这些 skill 只描述触发条件与控制面 workflow，不授予任何 exchange 直写权限。

其中：

- `quant-workflow` 负责长时间执行时的 phase / gate / resume
- `quant-test` 明确要求 proof test + real verification + impact safety
- `quant-docs` 要求 completion 前同步 durable docs

## 已验证 Research 入口

`2026-03-29` 已在远端仓库真实验证：

```bash
cd /root/.config/superpowers/worktrees/quant-lab/autoresearch-20260328-all-plan
OPENAI_API_KEY=... \
OPENAI_BASE_URL=https://right.codes/codex/v1 \
PATH=/usr/local/go/bin:/usr/bin:/bin \
go run ./cmd/platformctl research run \
  --kind nightly_report \
  --strategy mstr-wave-fib \
  --config-path configs/demo-mstr-bundle.yaml \
  --datasets MSTRUSDT:1h,BTCUSDT:1h
```

真实返回：

- `request_id=resp_00998a8df8fc81b40169c9378975c8819886178d53746cc0ff`
- `status=completed`
- artifact 根目录：`artifacts/platform/research/20260329T143032Z-nightly-report-mstr-wave-fib`

该目录当前真实包含：

- `request.json`
- `response.json`
- `output.txt`
- `summary.json`

## Codex CLI

`codex exec` 是当前最适合的非交互入口。帮助信息已验证支持：`--cd`、`--dangerously-bypass-approvals-and-sandbox`、`--output-last-message`。

只读检查样例：

```bash
/usr/bin/codex exec \
  --dangerously-bypass-approvals-and-sandbox \
  -C /root/.config/superpowers/worktrees/quant-lab/autoresearch-20260328-all-plan \
  "Use quant-platform-operator. Inspect platform status and strategy versions. Do not edit files."
```

workflow 样例：

```bash
/usr/bin/codex exec \
  --dangerously-bypass-approvals-and-sandbox \
  -C /root/.config/superpowers/worktrees/quant-lab/autoresearch-20260328-all-plan \
  "Use quant-workflow. Explain the resume order, stage gates, and durable docs required before completion. Do not edit files."
```

testing 规则样例：

```bash
/usr/bin/codex exec \
  --dangerously-bypass-approvals-and-sandbox \
  -C /root/.config/superpowers/worktrees/quant-lab/autoresearch-20260328-all-plan \
  "Use quant-test. For a change touching execd, summarize the proof-test gate, the real-verification gate, and the impact-safety gate only. Do not edit files."
```

研究样例：

```bash
/usr/bin/codex exec \
  --dangerously-bypass-approvals-and-sandbox \
  -C /root/.config/superpowers/worktrees/quant-lab/autoresearch-20260328-all-plan \
  "Use quant-theory-study. Read the current bundle and propose one backtestable hypothesis for mstr-wave-fib."
```

## Claude CLI

`claude -p` 是当前可用的非交互入口。帮助信息已验证支持 `-p/--print` 与 `--dangerously-skip-permissions`。帮助文本同时说明 skills 可通过 `/skill-name` 解析。

只读检查样例：

```bash
/usr/bin/claude -p \
  --dangerously-skip-permissions \
  --add-dir /root/.config/superpowers/worktrees/quant-lab/autoresearch-20260328-all-plan \
  "/quant-platform-operator Inspect platform status and strategy versions only. Do not edit files."
```

研究样例：

```bash
/usr/bin/claude -p \
  --dangerously-skip-permissions \
  --add-dir /root/.config/superpowers/worktrees/quant-lab/autoresearch-20260328-all-plan \
  "/quant-backtest-compare Summarize what evidence is still required before promoting a new mstr-wave-fib candidate."
```

## Guardrails

- bundle 编辑只发生在 `strategies/<strategy_id>/versions/<version>`。
- deterministic 验证只通过 `platformctl backtest run` 或 `platformd` API。
- promotion 只通过 `platformctl promotion ...` 推进。
- emergency flatten 只通过 `platformctl live flatten`。
- AI 不应直接触碰 Bitget 写接口或 runtime SQLite 状态文件。
- 长时间 workflow completion 前必须同步 durable docs，而不是只改代码。
