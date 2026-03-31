---
name: quant-docs
description: Use when quant platform work is nearing completion and the durable docs plus affected repository docs must be synchronized to the current end state
---

# Quant Docs

## Overview

This is the durable-doc gate. A run cannot finish until the docs match the as-built system.

## Mandatory Durable Docs

Update these files in the controller workspace root if present, otherwise in the repo-local mirror `workspace/`:

- `core.md`
- `quant-platform-功能说明.md`
- `quant-platform-架构文档.md`
- `quant-platform-接口文档.md`
- `quant-platform-通知与机器人操作文档.md`

## Update Rules

- `core.md` stores durable operational memory, not user-facing prose.
- `功能说明` answers what is delivered now.
- `架构文档` answers boundaries and data flow.
- `接口文档` answers stable HTTP / CLI / health surfaces.
- `通知与机器人操作文档` answers DingTalk / Telegram / OpenClaw operation and ownership.

## Repo Docs

If the run changes repo-visible behavior or skill usage, update the affected repo docs too, such as:

- `docs/README.md`
- `docs/ops/codex-claude-integration.md`
- `docs/ops/operator-quickstart.md`
- `docs/architecture/*`

## Completion Gate

If code, verification evidence, or runtime behavior changed but the durable docs did not, this skill must fail the run.
