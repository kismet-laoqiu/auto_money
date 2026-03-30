---
name: quant-theory-study
description: "Use when studying the market theory behind a strategy bundle, preparing a read-only theory summary, or gathering structured context before changing bundle parameters or CEL expressions"
---

# Quant Theory Study

## Overview
Start from theory and current bundle facts before editing strategy parameters. Keep this flow read-only until a backtestable hypothesis exists.

## Inputs
- `docs/theory/index.md`
- `strategies/<strategy_id>/versions/<version>/strategy.yaml`
- `strategies/<strategy_id>/versions/<version>/universe.yaml`
- `strategies/<strategy_id>/versions/<version>/risk.yaml`

## Steps
1. Read the theory index and the current bundle files.
2. Summarize the exact theory claims you want to test.
3. Run a research job.
   - `go run ./cmd/platformctl research run --kind theory_study --strategy mstr-wave-fib --config-path configs/demo-mstr-bundle.yaml --datasets MSTRUSDT:1h,BTCUSDT:1h`
4. Turn the study result into one concrete bundle hypothesis.
5. Only then clone a new version and backtest it.

## Guardrails
- Do not skip the theory read and jump straight to parameter twiddling.
- Do not propose live promotion without a new backtest artifact.
