# Agentic Trading Rollout Checklist

## Demo Gate Requirements

- `marketd` stays connected for at least 15 minutes before promotion.
- `traderd` remains in `safe` mode until reconciliation passes and a human arms it.
- Demo replay returns positive `event_count` and positive `candidate_count`.
- Demo order lifecycle reaches `filled` or `canceled` without orphan local state.
- Reduce-only exit is verified against a demo position before any live arming.
- Daily risk halt remains manually triggerable throughout rollout.
- `agentd` remains advisory-only and never gains exchange write authority.

## Preflight

- Confirm `configs/demo-bitget.yaml` keeps `live.runtime.observe_only: true`.
- Confirm both demo and live configs keep `live.runtime.arming_state: safe`.
- Confirm demo and live do not share the same `state_db_path`.
- Keep `live.agent.advisory_only: true`; only enable `agentd` after `OPENAI_API_KEY` is present.

## Verification Commands

```bash
PATH=/usr/local/go/bin:/usr/bin:/bin ./scripts/measure.sh configs/baseline.yaml
PATH=/usr/local/go/bin:/usr/bin:/bin go build ./cmd/lab ./cmd/marketd ./cmd/traderd ./cmd/agentd
PATH=/usr/local/go/bin:/usr/bin:/bin go run ./cmd/lab replay -config configs/demo-bitget.yaml
PATH=/usr/local/go/bin:/usr/bin:/bin RUN_LIVE_SMOKE=1 SMOKE_SECONDS=15 ./scripts/measure.sh configs/demo-bitget.yaml
PATH=/usr/local/go/bin:/usr/bin:/bin go build -o /tmp/quantlab-marketd ./cmd/marketd
PATH=/usr/local/go/bin:/usr/bin:/bin go build -o /tmp/quantlab-traderd ./cmd/traderd
PATH=/usr/local/go/bin:/usr/bin:/bin timeout --preserve-status --signal=INT --kill-after=2s 900s /tmp/quantlab-marketd -config configs/demo-bitget.yaml
PATH=/usr/local/go/bin:/usr/bin:/bin timeout --preserve-status --signal=INT --kill-after=2s 900s /tmp/quantlab-traderd -config configs/demo-bitget.yaml
```

## Staged Rollout

1. Demo trading with `observe_only=true`.
2. Demo trading with `observe_only=false` and `arming_state=safe`.
3. Demo trading with manual arm to `armed` only after reconciliation and reduce-only exit checks pass.
4. Live trading with `observe_only=true`.
5. Live trading with `observe_only=false` and `arming_state=safe`.
6. Live trading with manual arm to `armed` only after all demo gates remain green.

## Promotion Gates

- `go test ./...` passes.
- `go build ./cmd/lab ./cmd/marketd ./cmd/traderd ./cmd/agentd` passes.
- `lab replay -config configs/demo-bitget.yaml` exits `0` and writes a replay report under `artifacts/demo-bitget/`.
- Demo smoke keeps `marketd` and `traderd` alive for the configured smoke window.
- No manual override bypasses `safe`, `degraded`, or `halted` behavior.

## Cleanup

- `./scripts/measure.sh` writes `/tmp/quantlab-*` binaries and updates `metrics.json`.
- Direct `go build ./cmd/...` commands without `-o` leave repo-root binaries such as `lab`, `marketd`, `traderd`, and `agentd`.
- Before commit, clean transient build artifacts and restore `metrics.json` if verification was the only change.

## Known Limits

- Current `marketd` smoke validates startup, config loading, liveness, and signal handling. It does not prove full public websocket ingestion health end-to-end.
- Current `agentd` gate is advisory-only configuration plus build verification. It does not participate in order execution.
