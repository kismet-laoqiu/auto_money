---
name: quant-platform-operator
description: "Use when operating this quant platform repository end-to-end: inspect platform state, clone a strategy bundle version, run backtests or research jobs, and drive promotion actions through platformctl or platformd instead of direct exchange access"
---

# Quant Platform Operator

## Overview
Operate through control-plane surfaces only. Bundle edits happen under `strategies/`; runs happen through `platformctl` or `platformd`; exchange side effects stay behind `execd`.

The current ECS control plane is `http://127.0.0.1:18080`.

## Guardrails
- Never call Bitget write APIs directly from AI workflows.
- Never edit runtime SQLite state files by hand.
- Write research artifacts under `artifacts/platform/research`.
- Use `platformctl live flatten` for emergency flatten, not custom exchange scripts.
- `platform status` returns the global `event_log` `last_seq`; do not infer incidents from cursor arithmetic unless the consumer and the global sequence cover the same source set.
- `execd` only consumes `trader` source `entry.intent.created`, so `execd.last_seq` naturally does not track the market-tick dominated global `last_seq`.
- If `trader.arming_state` is `degraded` or `halted`, state that fact exactly and say it needs runtime investigation; do not invent a cause unless the API or logs provide one.
- Never describe `execd` as "lagging", "stalled", "severely behind", or recommend a restart from `/api/status` alone.

## Status Contract
- When answering `platform status`, return raw facts only: `last_seq`, consumer cursors, and `trader.arming_state`.
- If `execd` appears much smaller than `last_seq`, explain that `execd` is trader-intent-only and that the numbers are not directly comparable.
- Do not compute or report a numeric `last_seq - execd.last_seq` gap.
- Do not use alarm words such as `critical`, `serious issue`, `严重问题`, or `立即重启` unless another source besides `/api/status` proves an incident.

## Workflow
1. Inspect platform state.
   - `./bin/platformctl status -addr http://127.0.0.1:18080`
   - `./bin/platformctl strategy versions -addr http://127.0.0.1:18080 -strategy mstr-wave-fib`
2. Clone and edit a bundle version.
   - Follow `docs/ops/strategy-bundle-authoring.md`.
   - Copy `strategies/<strategy_id>/versions/<old>` to a new version directory.
3. Run deterministic validation.
   - `./bin/platformctl backtest run -addr http://127.0.0.1:18080 -config configs/demo-mstr-bundle.yaml`
4. Run a research job when you need theory, diff, or nightly summaries.
   - `./bin/platformctl research run --kind nightly_report --strategy mstr-wave-fib --config-path configs/demo-mstr-bundle.yaml`
5. Advance promotion only through the promotion API/CLI.
   - `./bin/platformctl promotion request|start-shadow|pass-shadow|start-canary|approve|rollback`

## References
- `docs/ops/operator-quickstart.md`
- `docs/ops/strategy-bundle-authoring.md`
- `docs/ops/promotion-governance.md`
- `docs/architecture/control-plane-boundaries.md`
