---
name: quant-platform-operator
description: "Use when operating this quant platform repository end-to-end: inspect platform state, clone a strategy bundle version, run backtests or research jobs, and drive promotion actions through platformctl or platformd instead of direct exchange access"
---

# Quant Platform Operator

## Overview
Operate through control-plane surfaces only. Bundle edits happen under `strategies/`; runs happen through `platformctl` or `platformd`; exchange side effects stay behind `execd`.

## Guardrails
- Never call Bitget write APIs directly from AI workflows.
- Never edit runtime SQLite state files by hand.
- Write research artifacts under `artifacts/platform/research`.
- Use `platformctl live flatten` for emergency flatten, not custom exchange scripts.

## Workflow
1. Inspect platform state.
   - `go run ./cmd/platformctl status`
   - `go run ./cmd/platformctl strategy versions -strategy mstr-wave-fib`
2. Clone and edit a bundle version.
   - Follow `docs/ops/strategy-bundle-authoring.md`.
   - Copy `strategies/<strategy_id>/versions/<old>` to a new version directory.
3. Run deterministic validation.
   - `go run ./cmd/platformctl backtest run -config configs/demo-mstr-bundle.yaml`
4. Run a research job when you need theory, diff, or nightly summaries.
   - `go run ./cmd/platformctl research run --kind nightly_report --strategy mstr-wave-fib --config-path configs/demo-mstr-bundle.yaml`
5. Advance promotion only through the promotion API/CLI.
   - `platformctl promotion request|start-shadow|pass-shadow|start-canary|approve|rollback`

## References
- `docs/ops/operator-quickstart.md`
- `docs/ops/strategy-bundle-authoring.md`
- `docs/ops/promotion-governance.md`
- `docs/architecture/control-plane-boundaries.md`
