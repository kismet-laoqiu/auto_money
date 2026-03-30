---
name: quant-backtest-compare
description: "Use when comparing baseline and candidate backtest outcomes, summarizing score deltas, or preparing a research diff before promotion decisions"
---

# Quant Backtest Compare

## Overview
Compare deterministic evidence, not vibes. The comparison unit is a pair of backtest artifacts or archived run IDs tied to known bundle/config inputs.

## Steps
1. Locate the baseline and candidate bundle versions.
   - `go run ./cmd/platformctl strategy versions -strategy mstr-wave-fib`
2. Ensure both variants have backtest artifacts.
   - `go run ./cmd/platformctl backtest run -config configs/demo-mstr-bundle.yaml`
3. If archived run IDs are available, ask the research plane for a diff summary.
   - `go run ./cmd/platformctl research run --kind backtest_diff_summary --strategy mstr-wave-fib --baseline-run <baseline_run_id> --candidate-run <candidate_run_id> --config-path configs/demo-mstr-bundle.yaml`
4. Read `backtest-result.json` and `backtest-report.md` before making promotion claims.

## Guardrails
- Compare runs from the same strategy family and a known dataset window.
- Do not claim improvement unless the artifact paths and scores both line up.
