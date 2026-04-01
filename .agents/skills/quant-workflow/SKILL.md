---
name: quant-workflow
description: Use when executing or resuming long-running development work for this quant platform, especially when work must coordinate durable workspace docs with code and verification on the remote ECS repository
---

# Quant Workflow

## Overview

This is the strong-state orchestrator for this repository. It manages workspace selection, resume order, stage gates, real-verification gates, and durable-doc completion. It does not do plan, coding, testing, changelog, or doc writing itself; it dispatches those responsibilities to the `quant-*` child skills.

## Workspace Roots

- Primary controller skill install:
  `/Users/qiukeming/Documents/projects/ob/obsidian/ecs/.agents/skills/quant-*/SKILL.md`
- Controller canonical workspace root:
  `/Users/qiukeming/Documents/projects/ob/obsidian/ecs/workspace`
- Repo mirror skill/runtime root:
  `/root/.config/superpowers/worktrees/quant-lab/autoresearch-20260328-all-plan/.agents/skills/quant-*/SKILL.md`
- Repo-local mirror workspace root:
  `workspace/`

Rules:

- Keep the `quant-*` skill pack on both sides. The controller canonical project install is the default entry; the repo mirror exists for direct-on-ECS development and remote `codex exec` / OpenClaw discovery.
- A legacy compatibility copy may exist under `/Users/qiukeming/.codex/skills/quant-*/SKILL.md`, but it is not the truth source and must not drift from the controller canonical project install.
- If the controller root exists in the current environment, it is the truth source for durable docs and run workspaces.
- If the controller root does not exist, use the repo-local mirror `workspace/` and keep the same filenames and structure.

## Durable Docs

Before completion, these five files must match the current system truth:

- `core.md`
- `quant-platform-功能说明.md`
- `quant-platform-架构文档.md`
- `quant-platform-接口文档.md`
- `quant-platform-通知与机器人操作文档.md`

## Resume Order

On start, resume, or after compaction, read in this order:

1. controller `workspace/core.md`, if present; otherwise repo-local `workspace/core.md`
2. controller `AGENTS.md`, if present
3. repo `AGENTS.md`
4. current run workspace `status.md`
5. current run workspace `execution-log.md`

## Controller-to-ECS Transport Guardrail

When operating from the Mac controller to `47.250.138.143`, do not treat a controller-side OpenSSH transport error as proof that ECS is down.

- Known controller failure mode:
  - `ssh: connect to host 47.250.138.143 port 22: Bad file descriptor`
- First distinguish transport from reachability:
  - run `nc -zvw5 47.250.138.143 22`
- If TCP `22` is reachable, retry `ssh` and any already-verified `rsync` path with:
  - `-o ProxyCommand='nc %h %p'`
- For controller-to-ECS file writes in this environment, do **not** assume `scp` is available.
  - Prefer streaming a verified local file over `ssh` stdin, for example:
    - `ssh ... 'cat > /remote/path/file' < /local/path/file`
- Do not spend turns probing `scp` in this environment. Treat it as unsupported unless the user states otherwise.
- Record this fallback in the current run workspace `execution-log.md` when used, so later turns do not misdiagnose the same controller-local issue as an ECS outage.

If the phase is `coding` or later, run the remote baseline next:

- `git status --short --branch`
- `PATH=/usr/local/go/bin:$PATH go test ./...`
- `PATH=/usr/local/go/bin:$PATH go build ./cmd/lab ./cmd/marketd ./cmd/traderd ./cmd/agentd`
- check `BITGET_API_KEY` `BITGET_API_SECRET` `BITGET_PASSPHRASE` `OPENAI_API_KEY` presence when relevant

## Run Workspace

Create run workspaces under:

- controller root: `workspace/runs/YYYYMMDD_HHMMSS_<title>/`
- fallback mirror: `workspace/runs/YYYYMMDD_HHMMSS_<title>/`

Required files:

- `status.md`
- `execution-log.md`
- `plan.md`
- `plan-review.md`
- `code-review.md`
- `测试记录.md`
- `测试报告.md`
- `changelog.md`
- `insights.md`
- `tmp/`

`status.md` must include:

- `remote_repo`
- `branch`
- `base_commit`
- `phase`
- `current_requirement`
- `durable_docs`
- `last_resume_anchor`

## Stages

1. `recover`
2. `plan`
3. `plan-review`
4. `coding`
5. `code-review`
6. `testing`
7. `docs`
8. `changelog`
9. `retrospective`
10. `done`

Use these child skills:

- `quant-plan`
- `quant-plan-review`
- `quant-coding`
- `quant-code-review`
- `quant-test`
- `quant-docs`
- `quant-changelog`
- `quant-retrospective`

## Gates

- Do not skip `plan-review`, `code-review`, `testing`, or `docs`.
- Do not claim completion until the five durable docs are updated.
- Do not treat `go test ./...` as sufficient verification.
- For runtime-facing changes, do not mark the run `done` until the source is pushed, the target ECS service is rebuilt/restarted as needed, and both `127.0.0.1` plus the public entry verify the new behavior.
- High-risk changes must stop for user confirmation:
  - real trading semantics
  - hard risk defaults
  - irreversible schema changes
  - secrets, bot ownership, or production addresses
  - promotion or execution trust-boundary changes

## Testing Gate

Testing is always three layers:

1. proof test
2. real verification
3. impact safety

If a real test cannot guarantee that it will not disturb live positions, watchlist, promotion state, Telegram ownership, or OpenClaw ownership, stop and ask the user.
