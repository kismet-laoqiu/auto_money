---
name: quant-test
description: Use when verifying quant platform changes and the work requires both proof tests and safe real validation against the shared ECS runtime
---

# Quant Test

## Overview

Testing for this repository is never only `go test`. Every change needs proof tests and real verification, but real verification must not disturb the shared live environment.

## Three-Layer Gate

### 1. Proof Test

Before real verification, run a test that proves the change:

- unit test for new behavior
- integration test for multi-component logic
- reproducing test first for bug fixes

No proof test means no pass.

### 2. Real Verification

Run on the remote ECS environment using real runtime surfaces:

- real HTTP APIs
- real sqlite / warehouse reads
- real notifier runtime
- real OpenAI / Codex skill smoke when AI skills changed
- real Bitget safe path only when exchange write logic changed

### 3. Impact Safety

The shared environment must not be harmed.

Unsafe without approval:

- changing live watchlist in a way that cannot be restored immediately
- mutating promotion state on the active path without rollback containment
- changing Telegram bot ownership
- any exchange write path that is not the fixed MSTR safety line

## Surface Matrix

- `platformd` / dashboard / handlers:
  - run proof tests
  - run `curl /health`
  - run read-only API smokes
- `warehouse` / watchlist read path:
  - run proof tests
  - run `warehouse_query.sh` coverage/freshness/gaps
- notifier / Telegram / DingTalk:
  - run proof tests
  - run real notifier health and safe notify smoke if message noise is acceptable
- skill pack changes:
  - run proof checks on file presence and content expectations
  - run real `codex exec` skill smoke
- Bitget / execd:
  - run proof tests
  - run `TestRealBitgetEnsureFlatPosition`
  - run the minimum safe real lifecycle on `MSTRUSDT`
  - run `TestRealBitgetEnsureFlatPosition` again

## Evidence

Write all test evidence into the current run workspace:

- `测试记录.md`
- `测试报告.md`
- raw artifacts under `tmp/testing/`

## Stop Rule

If a real test would risk current positions, watchlist, promotion state, or bot ownership and cannot be automatically restored, stop and ask the user.
