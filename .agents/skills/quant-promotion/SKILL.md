---
name: quant-promotion
description: "Use when requesting, advancing, degrading, approving, or rolling back strategy promotion states after deterministic backtest evidence exists"
---

# Quant Promotion

## Overview
Promotion is a state machine, not a free-form status flip. Use the platform API/CLI sequence exactly.

## Sequence
1. Create the request.
   - `go run ./cmd/platformctl promotion request -strategy mstr-wave-fib -version v0.1.0 -config configs/demo-mstr-bundle.yaml`
2. Advance the state.
   - `platformctl promotion start-shadow -id <promotion_id>`
   - `platformctl promotion pass-shadow -id <promotion_id>`
   - `platformctl promotion start-canary -id <promotion_id>`
3. Finish with one terminal choice.
   - `platformctl promotion approve -id <promotion_id>`
   - or `platformctl promotion rollback -id <promotion_id> -reason '<reason>'`

## Guardrails
- Never skip `shadow` or `canary`.
- `rollback` must include a concrete reason.
- Read `docs/ops/promotion-governance.md` before touching a live-facing promotion.
