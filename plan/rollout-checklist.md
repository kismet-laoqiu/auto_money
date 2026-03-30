# Agentic Trading Rollout Checklist

## Platform Baseline Gate

- `plan/2026-03-29-platform-baseline-checklist-cn.md` must exist and stay aligned with the live repo state.
- `scripts/run_mstr_e2e.sh` completes preflight on `MSTRUSDT` before any promotion.
- Any real Bitget verification must end with a flat position.
- `marketd` stays connected for at least 15 minutes before promotion.
- `traderd` remains in `safe` mode until reconciliation passes and a human arms it.
- `agentd` remains advisory-only and never gains exchange write authority.
- `cmd/stream` is legacy-only; it is not a platform completion signal.
- `execd` is the only allowed Bitget write path; any new `traderd` write call is a regression.

## Preflight

- Confirm `configs/demo-mstr-e2e.yaml` keeps `live.runtime.observe_only: true`.
- Confirm both demo and live configs keep `live.runtime.arming_state: safe`.
- Confirm `configs/demo-mstr-e2e.yaml` and live config do not share the same `state_db_path`.
- Confirm `BITGET_API_KEY`, `BITGET_API_SECRET`, and `BITGET_PASSPHRASE` are present before running any MSTR E2E step.
- Keep `live.agent.advisory_only: true`; only enable `agentd` after `OPENAI_API_KEY` is present.
- Treat `MSTRUSDT` real minimum effective live size as `0.04` until a fresh preflight proves a new value.

## Verification Commands

```bash
PATH=/usr/local/go/bin:/usr/bin:/bin ./scripts/measure.sh configs/baseline.yaml
PATH=/usr/local/go/bin:/usr/bin:/bin go test ./...
PATH=/usr/local/go/bin:/usr/bin:/bin go build ./cmd/lab ./cmd/marketd ./cmd/traderd ./cmd/agentd ./cmd/platformd ./cmd/platformctl ./cmd/notifierd
PATH=/usr/local/go/bin:/usr/bin:/bin go build ./cmd/execd
PATH=/usr/local/go/bin:/usr/bin:/bin ./scripts/run_mstr_e2e.sh
PATH=/usr/local/go/bin:/usr/bin:/bin go run ./cmd/lab replay -config configs/demo-mstr-e2e.yaml
PATH=/usr/local/go/bin:/usr/bin:/bin go build -o /tmp/quantlab-marketd ./cmd/marketd
PATH=/usr/local/go/bin:/usr/bin:/bin go build -o /tmp/quantlab-traderd ./cmd/traderd
RUN_BITGET_REAL=1 BITGET_API_KEY=... BITGET_API_SECRET=... BITGET_PASSPHRASE=... PATH=/usr/local/go/bin:/usr/bin:/bin go test ./internal/execution -run TestRealExecutionRuntimeIntentLifecycle -v
PATH=/usr/local/go/bin:/usr/bin:/bin timeout --preserve-status --signal=INT --kill-after=2s 900s /tmp/quantlab-marketd -config configs/demo-mstr-e2e.yaml
PATH=/usr/local/go/bin:/usr/bin:/bin timeout --preserve-status --signal=INT --kill-after=2s 900s /tmp/quantlab-traderd -config configs/demo-mstr-e2e.yaml
```

## Staged Rollout

1. Demo trading with `observe_only=true`.
2. Demo trading with `observe_only=false` and `arming_state=safe`.
3. Demo trading with manual arm to `armed` only after reconciliation and reduce-only exit checks pass.
4. Live trading with `observe_only=true`.
5. Live trading with `observe_only=false` and `arming_state=safe`.
6. Live trading with manual arm to `armed` only after all demo gates remain green.
7. Keep `execd` as the sole write sidecar and block any platform-level promotion if `traderd` regains exchange write code.

## Promotion Gates

- `go test ./...` passes.
- `go build ./cmd/lab ./cmd/marketd ./cmd/traderd ./cmd/agentd ./cmd/platformd ./cmd/platformctl ./cmd/notifierd ./cmd/execd` passes.
- `lab replay -config configs/demo-mstr-e2e.yaml` exits `0` and writes a replay report under `artifacts/mstr-e2e/`.
- `scripts/run_mstr_e2e.sh` writes a preflight summary under `artifacts/mstr-e2e/<run_id>/`.
- Demo smoke keeps `marketd` and `traderd` alive for the configured smoke window.
- No manual override bypasses `safe`, `degraded`, or `halted` behavior.
- No run that leaves open test exposure can be called successful.

## Cleanup

- `./scripts/measure.sh` writes `/tmp/quantlab-*` binaries and updates `metrics.json`.
- Direct `go build ./cmd/...` commands without `-o` leave repo-root binaries such as `lab`, `marketd`, `traderd`, and `agentd`.
- Repo runtime state lives under `var/`; artifacts live under `artifacts/`; both must stay out of commits.
- Before commit, clean transient build artifacts and restore `metrics.json` if verification was the only change.

## Known Limits

- Current `marketd` smoke validates startup, config loading, liveness, and signal handling. It does not prove full public websocket ingestion health end-to-end.
- Current `agentd` gate is advisory-only configuration plus build verification. It does not participate in order execution.
- Current `platformd` now serves `bars / features / backtest run / promotion request+gate` APIs, but it still lacks `strategy versions` and `live ops`.
- Current Telegram bot integration is outbound-only; command ingress has not been wired yet.
- `execd` real lifecycle is verified via package test and temp SQLite intent replay; service orchestration and systemd wiring are still pending.
