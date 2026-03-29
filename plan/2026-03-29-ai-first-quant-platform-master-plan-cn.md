# AI-First Quant Platform Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a non-demo, AI-first quant platform on ECS that supports historical market data management, feature computation, strategy versioning, backtesting, scoring, Telegram/DingTalk/OpenClaw interaction, and real Bitget live trading with safe promotion and rollback.

**Architecture:** Keep the proven Go runtime path for market ingress and live execution, but lift it into a full platform with a central warehouse, a canonical strategy bundle format, a control plane API/MCP layer, and AI/human interaction surfaces. Live trading stays isolated behind `execd`; AI only operates through controlled platform APIs and skills.

**Tech Stack:** Go 1.26, SQLite/WAL edge event log, PostgreSQL 17 + TimescaleDB, Parquet + DuckDB exports, `chi` HTTP API, `sqlc` + migration tool, Bitget REST/WebSocket, Telegram Bot API, DingTalk robots, OpenClaw gateway, OpenAI Responses API, CEL-based strategy expressions.

---

## Scope Rules

- This is a master plan for the full platform, not a one-feature patch.
- Execution must still happen phase by phase.
- Every phase must end with runnable software and a real verification gate.
- No fake completion: Bitget write path, Telegram delivery, DingTalk delivery, and OpenClaw-triggered AI tasks must all be real.

## File Structure Target

### New Commands

- `cmd/platformd/main.go`
- `cmd/execd/main.go`
- `cmd/notifierd/main.go`
- `cmd/researchd/main.go`
- `cmd/warehouse-syncd/main.go`
- `cmd/mcpd/main.go`
- `cmd/platformctl/main.go`

### Existing Commands To Keep/Refactor

- `cmd/marketd/main.go`
- `cmd/traderd/main.go`
- `cmd/lab/main.go`
- `cmd/agentd/main.go` or replace with `cmd/researchd/main.go`

### Commands To Retire

- `cmd/stream/main.go`

### New Core Packages

- `internal/platform/api`
- `internal/platform/authz`
- `internal/platform/promotion`
- `internal/platform/registry`
- `internal/platform/jobs`
- `internal/platform/mcp`
- `internal/platform/telegram`
- `internal/platform/notifier`
- `internal/platform/openclaw`
- `internal/warehouse/catalog`
- `internal/warehouse/ingest`
- `internal/warehouse/export`
- `internal/warehouse/features`
- `internal/strategybundle`
- `internal/backtest`
- `internal/scoring`
- `internal/execution`

### Existing Packages To Refactor

- `internal/adapters/data.go`
- `internal/adapters/notify.go`
- `internal/agent/*`
- `internal/core/core.go`
- `internal/core/features.go`
- `internal/market/*`
- `internal/trader/*`
- `internal/exchange/bitget/*`
- `internal/store/sqlite/*`

### Infra / Config

- `deploy/docker-compose.platform.yml`
- `deploy/systemd/*.service`
- `configs/platform/*.yaml`
- `migrations/postgres/*.sql`
- `strategies/README.md`
- `.agents/skills/quant-platform-operator/SKILL.md`
- `.agents/skills/quant-theory-study/SKILL.md`
- `.agents/skills/quant-backtest-compare/SKILL.md`
- `.agents/skills/quant-promotion/SKILL.md`

---

### Task 1: Freeze Proven Runtime Facts and Establish Platform Baseline

**Files:**
- Modify: `plan/rollout-checklist.md`
- Modify: `scripts/run_mstr_e2e.sh`
- Modify: `internal/exchange/bitget/real_bitget_test.go`
- Create: `plan/2026-03-29-platform-baseline-checklist-cn.md`

- [ ] **Step 1: Record the current proven baseline**
  Run:
  ```bash
  PATH=/usr/local/go/bin:/usr/bin:/bin go test ./...
  PATH=/usr/local/go/bin:/usr/bin:/bin go build ./cmd/lab ./cmd/marketd ./cmd/traderd ./cmd/agentd
  PATH=/usr/local/go/bin:/usr/bin:/bin RUN_BITGET_REAL=1 BITGET_API_KEY=$BITGET_API_KEY BITGET_API_SECRET=$BITGET_API_SECRET BITGET_PASSPHRASE=$BITGET_PASSPHRASE go test ./internal/exchange/bitget -run 'TestRealBitget(FetchFuturesCandles|PrivateReadSurface|OrderLifecycleAndPrivateStream|EnsureFlatPosition)' -v
  PATH=/usr/local/go/bin:/usr/bin:/bin ./scripts/run_mstr_e2e.sh
  ```
  Expected:
  - all tests pass
  - preflight computes live minimum valid size
  - final cleanup returns to flat

- [ ] **Step 2: Write a baseline facts doc**
  Capture:
  - current event-log schema
  - current MSTR real proof
  - current `FINAL_SCORE`
  - current active gaps

- [ ] **Step 3: Tighten checklist language**
  Add explicit statements:
  - `stream` is legacy and not a platform completion signal
  - Bitget flat-position cleanup is mandatory after every real test
  - platform completion requires Telegram and DingTalk validation

- [ ] **Step 4: Verify docs-only change does not alter runtime**
  Run:
  ```bash
  PATH=/usr/local/go/bin:/usr/bin:/bin go test ./internal/exchange/bitget ./internal/market ./internal/trader -v
  ```

### Task 2: Introduce Central Warehouse Infrastructure

**Files:**
- Create: `deploy/docker-compose.platform.yml`
- Create: `migrations/postgres/0001_init_market.sql`
- Create: `migrations/postgres/0002_init_strategy.sql`
- Create: `migrations/postgres/0003_init_backtest.sql`
- Create: `migrations/postgres/0004_init_ops.sql`
- Create: `internal/warehouse/catalog/*.go`
- Create: `internal/warehouse/catalog/*_test.go`
- Create: `configs/platform/warehouse.yaml`

- [ ] **Step 1: Stand up PostgreSQL + TimescaleDB**
  Bring up a dedicated warehouse database for platform state.

- [ ] **Step 2: Create warehouse schemas**
  Required schemas:
  - `market`
  - `features`
  - `strategy`
  - `backtest`
  - `ops`
  - `ai`

- [ ] **Step 3: Add repository layer**
  Repositories must support:
  - bars upsert
  - feature snapshot upsert
  - strategy version CRUD
  - backtest run creation
  - promotion request creation

- [ ] **Step 4: Verify DB bootstrap**
  Run:
  ```bash
  docker compose -f deploy/docker-compose.platform.yml up -d
  PATH=/usr/local/go/bin:/usr/bin:/bin go test ./internal/warehouse/catalog -v
  ```
  Expected:
  - migrations apply cleanly
  - repository tests pass

### Task 3: Replace CSV Dataset Cache With Platform Historical Ingestion

**Files:**
- Modify: `internal/adapters/data.go`
- Create: `internal/warehouse/ingest/historical_job.go`
- Create: `internal/warehouse/ingest/historical_job_test.go`
- Create: `cmd/platformctl/main.go`
- Create: `configs/platform/historical-sync.yaml`

- [ ] **Step 1: Split provider fetch from platform ingest**
  `adapters/data.go` should stop being the long-term source of truth.
  Create provider clients that feed warehouse jobs instead of only CSV cache.

- [ ] **Step 2: Add paginated historical sync**
  Support:
  - Bitget futures `1m/5m/15m/1h/4h/1d`
  - existing non-Bitget providers for cross-asset research

- [ ] **Step 3: Persist normalized bars**
  Store:
  - provider raw symbol
  - canonical symbol
  - interval
  - open/high/low/close/volume
  - source timestamp range
  - checksum / ingest version

- [ ] **Step 4: Add CLI**
  Run:
  ```bash
  PATH=/usr/local/go/bin:/usr/bin:/bin go run ./cmd/platformctl historical sync --symbol MSTRUSDT --provider bitget --product-type USDT-FUTURES --interval 1m --days 30
  PATH=/usr/local/go/bin:/usr/bin:/bin go run ./cmd/platformctl historical sync --symbol BTCUSDT --provider bitget --product-type USDT-FUTURES --interval 1h --days 120
  ```

- [ ] **Step 5: Verify data landed**
  Check:
  ```bash
  psql "$PLATFORM_DATABASE_URL" -c "select symbol, interval, count(*) from market.bars group by 1,2 order by 1,2;"
  ```

### Task 4: Build Multi-Timeframe Aggregation and Export Layer

**Files:**
- Create: `internal/warehouse/ingest/aggregate_job.go`
- Create: `internal/warehouse/export/parquet_export.go`
- Create: `internal/warehouse/export/parquet_export_test.go`
- Create: `internal/warehouse/export/duckdb_manifest.go`
- Create: `configs/platform/export.yaml`

- [ ] **Step 1: Materialize derived intervals**
  Use canonical lower timeframe bars to derive higher timeframe bars where appropriate.

- [ ] **Step 2: Add Parquet export**
  Partition by:
  - `provider`
  - `symbol`
  - `interval`
  - `date`

- [ ] **Step 3: Emit DuckDB-readable manifests**
  AI agents should be able to read market datasets without touching runtime DB files.

- [ ] **Step 4: Verify export**
  Run:
  ```bash
  PATH=/usr/local/go/bin:/usr/bin:/bin go run ./cmd/platformctl export bars --symbol MSTRUSDT --interval 1m --out artifacts/exports/mstr-1m
  duckdb -c "select count(*) from read_parquet('artifacts/exports/mstr-1m/**/*.parquet');"
  ```

### Task 5: Versionize Feature Materialization

**Files:**
- Modify: `internal/core/features.go`
- Create: `internal/warehouse/features/materializer.go`
- Create: `internal/warehouse/features/materializer_test.go`
- Create: `internal/warehouse/features/version.go`
- Create: `cmd/platformctl/features.go`

- [ ] **Step 1: Give every feature group a versioned identity**
  Persist:
  - feature set version
  - strategy config dependency
  - symbol
  - interval
  - bar timestamp

- [ ] **Step 2: Batch materialize snapshots**
  The warehouse must store `wave`, `levels`, `fib`, `trigger`, `volume`, `regime`.

- [ ] **Step 3: Add CLI and API hooks**
  Run:
  ```bash
  PATH=/usr/local/go/bin:/usr/bin:/bin go run ./cmd/platformctl features materialize --symbol MSTRUSDT --interval 1h --from 2025-01-01 --to 2025-03-01
  ```

- [ ] **Step 4: Verify snapshot persistence**
  Check:
  ```bash
  psql "$PLATFORM_DATABASE_URL" -c "select feature_version, count(*) from features.snapshots group by 1;"
  ```

### Task 6: Introduce Canonical Strategy Bundle

**Files:**
- Create: `internal/strategybundle/bundle.go`
- Create: `internal/strategybundle/loader.go`
- Create: `internal/strategybundle/validator.go`
- Create: `internal/strategybundle/*_test.go`
- Create: `strategies/README.md`
- Create: `strategies/mstr-wave-fib/versions/v0.1.0/*`
- Modify: `internal/config/config.go`

- [ ] **Step 1: Define bundle schema**
  Required files:
  - `strategy.yaml`
  - `score.cel`
  - `gates.cel`
  - `risk.yaml`
  - `universe.yaml`

- [ ] **Step 2: Add bundle validation**
  Validation must reject:
  - missing files
  - unknown features
  - invalid CEL expressions
  - unsafe promotion policy

- [ ] **Step 3: Add versioned storage**
  Warehouse tables:
  - `strategy.strategies`
  - `strategy.strategy_versions`
  - `strategy.strategy_files`

- [ ] **Step 4: Verify bundle round-trip**
  Run:
  ```bash
  PATH=/usr/local/go/bin:/usr/bin:/bin go test ./internal/strategybundle -v
  PATH=/usr/local/go/bin:/usr/bin:/bin go run ./cmd/platformctl strategy validate --path strategies/mstr-wave-fib/versions/v0.1.0
  ```

### Task 7: Move Signal Logic From Hardcoded Weights To Bundle-Driven Evaluation

**Files:**
- Modify: `internal/core/core.go`
- Modify: `internal/trader/legacy_strategy.go`
- Modify: `internal/trader/engine.go`
- Create: `internal/trader/bundle_strategy.go`
- Create: `internal/trader/bundle_strategy_test.go`

- [ ] **Step 1: Preserve current strategy as bundle baseline**
  Encode current `EvaluateSignal` behavior into the first `StrategyBundle`.

- [ ] **Step 2: Replace hardcoded strategy coupling**
  `traderd` and `lab backtest` should load bundle-based evaluation instead of implicit fixed score weights.

- [ ] **Step 3: Keep deterministic parity**
  Current baseline bundle must reproduce current `FINAL_SCORE` within a tight tolerance.

- [ ] **Step 4: Verify parity**
  Run:
  ```bash
  PATH=/usr/local/go/bin:/usr/bin:/bin ./scripts/measure.sh configs/baseline.yaml
  PATH=/usr/local/go/bin:/usr/bin:/bin go run ./cmd/platformctl strategy backtest --strategy mstr-wave-fib --version v0.1.0
  ```
  Expected:
  - bundle-driven score matches baseline within documented tolerance

### Task 8: Build Platform Backtest and Scoring Control Plane

**Files:**
- Create: `internal/backtest/runner.go`
- Create: `internal/backtest/report.go`
- Create: `internal/scoring/final_score.go`
- Create: `internal/scoring/final_score_test.go`
- Modify: `cmd/lab/main.go`
- Create: `internal/platform/api/backtest_handlers.go`

- [ ] **Step 1: Turn `lab` into a reusable engine**
  `cmd/lab` should remain the canonical deterministic backtest binary, but callable by platform APIs/jobs.

- [ ] **Step 2: Persist runs and reports**
  Save:
  - run id
  - strategy version
  - dataset set
  - metrics
  - report artifact paths
  - diff against baseline

- [ ] **Step 3: Add compare API**
  Platform should answer:
  - did this version beat baseline
  - where did it improve or degrade
  - which bucket got worse

- [ ] **Step 4: Verify**
  Run:
  ```bash
  PATH=/usr/local/go/bin:/usr/bin:/bin go run ./cmd/platformctl strategy backtest --strategy mstr-wave-fib --version v0.1.0 --json
  PATH=/usr/local/go/bin:/usr/bin:/bin go test ./internal/backtest ./internal/scoring -v
  ```

### Task 9: Split Execution Side Effects Into `execd`

**Files:**
- Create: `cmd/execd/main.go`
- Create: `internal/execution/runtime.go`
- Create: `internal/execution/runtime_test.go`
- Create: `internal/execution/intent.go`
- Modify: `internal/trader/runtime.go`
- Modify: `internal/trader/runtime_events.go`

- [ ] **Step 1: Stop `traderd` from writing to exchange**
  `traderd` emits `entry.intent.created`, not `PlaceOrder`.

- [ ] **Step 2: Let `execd` consume intents**
  `execd` owns:
  - leverage set
  - place order
  - query order
  - reconcile
  - reduce-only close

- [ ] **Step 3: Preserve current Bitget proof**
  Reuse current proven order lifecycle logic from `internal/exchange/bitget/real_bitget_test.go`.

- [ ] **Step 4: Verify real execution**
  Run:
  ```bash
  PATH=/usr/local/go/bin:/usr/bin:/bin RUN_BITGET_REAL=1 BITGET_API_KEY=$BITGET_API_KEY BITGET_API_SECRET=$BITGET_API_SECRET BITGET_PASSPHRASE=$BITGET_PASSPHRASE go test ./internal/execution ./internal/exchange/bitget -run RealBitget -v
  ```
  Expected:
  - real canary lifecycle passes
  - final position is flat

### Task 10: Build Direct Telegram + DingTalk Notification Plane

**Files:**
- Modify: `internal/adapters/notify.go`
- Create: `internal/platform/notifier/telegram.go`
- Create: `internal/platform/notifier/notifier.go`
- Create: `internal/platform/notifier/*_test.go`
- Create: `cmd/notifierd/main.go`
- Modify: `internal/config/config.go`

- [ ] **Step 1: Keep DingTalk and add Telegram sender**
  Telegram send path should use Bot API directly for critical alerts.

- [ ] **Step 2: Add notification routing rules**
  Routes:
  - fills
  - risk halt
  - promotion approved
  - canary degraded
  - daily summary

- [ ] **Step 3: Verify outbound delivery**
  Run:
  ```bash
  PATH=/usr/local/go/bin:/usr/bin:/bin go test ./internal/platform/notifier -v
  PATH=/usr/local/go/bin:/usr/bin:/bin go run ./cmd/platformctl notify test --channel dingtalk
  PATH=/usr/local/go/bin:/usr/bin:/bin go run ./cmd/platformctl notify test --channel telegram
  ```

### Task 11: Build `platformd` HTTP API and MCP Server

**Files:**
- Create: `cmd/platformd/main.go`
- Create: `cmd/mcpd/main.go`
- Create: `internal/platform/api/router.go`
- Create: `internal/platform/api/market_handlers.go`
- Create: `internal/platform/api/strategy_handlers.go`
- Create: `internal/platform/api/ops_handlers.go`
- Create: `internal/platform/mcp/server.go`
- Create: `internal/platform/mcp/tools_*.go`
- Create: `internal/platform/mcp/*_test.go`

- [ ] **Step 1: Add REST control plane**
  Required APIs:
  - health
  - bars query
  - feature query
  - strategy version list/get/compare
  - backtest run create/get
  - promotion request create/get/approve
  - live ops get status / halt / flatten

- [ ] **Step 2: Add read-only MCP**
  Tools:
  - bars search/fetch
  - feature snapshot fetch
  - strategy list/fetch
  - backtest result fetch
  - live status fetch

- [ ] **Step 3: Add write-protected MCP**
  Tools:
  - clone strategy version
  - run backtest
  - create promotion request
  - halt / flatten

- [ ] **Step 4: Verify**
  Run:
  ```bash
  PATH=/usr/local/go/bin:/usr/bin:/bin go test ./internal/platform/api ./internal/platform/mcp -v
  PATH=/usr/local/go/bin:/usr/bin:/bin go build ./cmd/platformd ./cmd/mcpd
  curl -fsS http://127.0.0.1:8080/health
  ```

### Task 12: Integrate OpenClaw as Conversational Gateway

**Files:**
- Create: `internal/platform/openclaw/client.go`
- Create: `internal/platform/openclaw/client_test.go`
- Create: `internal/platform/api/openclaw_handlers.go`
- Create: `configs/platform/openclaw.yaml`
- Create: `docs/ops/openclaw-setup.md`

- [ ] **Step 1: Treat OpenClaw as chat/job ingress**
  Supported flows:
  - Telegram command -> OpenClaw -> platform job
  - platform result -> OpenClaw -> Telegram delivery

- [ ] **Step 2: Add platform commands**
  Required actions:
  - start research run
  - poll research run
  - send backtest summary
  - request promotion approval

- [ ] **Step 3: Verify OpenClaw connectivity**
  Run:
  ```bash
  openclaw channels list
  openclaw channels add --channel telegram --token "$TELEGRAM_BOT_TOKEN"
  openclaw agent --local --message "platform status" --json
  ```

### Task 13: Replace Advisory `agentd` With `researchd`

**Files:**
- Create: `cmd/researchd/main.go`
- Create: `internal/platform/jobs/research_job.go`
- Create: `internal/platform/jobs/research_job_test.go`
- Modify: `internal/agent/*` or move to `internal/platform/jobs/*`

- [ ] **Step 1: Stop treating AI as a sidecar explainer only**
  `researchd` must support:
  - theory study jobs
  - dataset scout jobs
  - backtest diff summarization
  - nightly report generation

- [ ] **Step 2: Support background mode + webhook/polling**
  Persist:
  - request id
  - status
  - output artifact
  - tool calls summary

- [ ] **Step 3: Verify**
  Run:
  ```bash
  PATH=/usr/local/go/bin:/usr/bin:/bin go test ./internal/platform/jobs -v
  PATH=/usr/local/go/bin:/usr/bin:/bin go run ./cmd/platformctl research run --kind nightly_report --strategy mstr-wave-fib
  ```

### Task 14: Ship AI Skill Pack for Codex and Claude Code

**Files:**
- Create: `.agents/skills/quant-platform-operator/SKILL.md`
- Create: `.agents/skills/quant-theory-study/SKILL.md`
- Create: `.agents/skills/quant-backtest-compare/SKILL.md`
- Create: `.agents/skills/quant-promotion/SKILL.md`
- Create: `docs/ops/codex-claude-integration.md`

- [ ] **Step 1: Create one umbrella operational skill**
  It must teach the platform workflow:
  - inspect platform state
  - fetch theory/data
  - clone strategy
  - edit bundle
  - run backtest
  - compare baseline
  - create promotion request
  - watch shadow/canary

- [ ] **Step 2: Attach MCP usage contracts**
  The skill must instruct agents when to use:
  - read MCP
  - write MCP
  - platformctl CLI
  - OpenClaw delivery

- [ ] **Step 3: Verify agent usability**
  Run real smoke:
  ```bash
  codex exec "Use the quant-platform-operator workflow to inspect current strategy status"
  claude --print "/quant-platform-operator inspect current strategy status"
  ```
  Expected:
  - both can read platform state
  - neither gains exchange write authority directly

### Task 15: Build Promotion, Shadow, Canary, and Rollback Governance

**Files:**
- Create: `internal/platform/promotion/state_machine.go`
- Create: `internal/platform/promotion/state_machine_test.go`
- Create: `internal/platform/promotion/shadow.go`
- Create: `internal/platform/promotion/canary.go`
- Create: `internal/platform/promotion/rollback.go`
- Modify: `plan/rollout-checklist.md`

- [ ] **Step 1: Add promotion states**
  Required:
  - `backtest_passed`
  - `shadow_running`
  - `shadow_passed`
  - `canary_running`
  - `live_active`
  - `rolled_back`

- [ ] **Step 2: Implement shadow**
  Shadow must:
  - read live events
  - evaluate candidate strategy
  - not write orders
  - compare signal frequency and quality against active strategy

- [ ] **Step 3: Implement canary**
  Canary must:
  - trade with minimum valid real size
  - run under tighter risk cap
  - allow immediate rollback

- [ ] **Step 4: Verify**
  Run:
  ```bash
  PATH=/usr/local/go/bin:/usr/bin:/bin go test ./internal/platform/promotion -v
  PATH=/usr/local/go/bin:/usr/bin:/bin go run ./cmd/platformctl promotion start-shadow --strategy mstr-wave-fib --version v0.1.1
  PATH=/usr/local/go/bin:/usr/bin:/bin go run ./cmd/platformctl promotion start-canary --strategy mstr-wave-fib --version v0.1.1
  ```

### Task 16: Full End-to-End Real Validation

**Files:**
- Create: `plan/2026-03-29-platform-e2e-report-cn.md`
- Create: `scripts/run_platform_e2e.sh`
- Create: `scripts/run_platform_skill_smoke.sh`

- [ ] **Step 1: Full platform smoke**
  Run:
  ```bash
  PATH=/usr/local/go/bin:/usr/bin:/bin ./scripts/run_platform_e2e.sh
  ```
  The script must validate:
  - warehouse ingest
  - feature materialization
  - bundle validation
  - backtest run
  - Telegram outbound
  - DingTalk outbound
  - OpenClaw job dispatch
  - shadow start
  - real Bitget order lifecycle
  - final flat position

- [ ] **Step 2: AI skill smoke**
  Run:
  ```bash
  PATH=/usr/local/go/bin:/usr/bin:/bin ./scripts/run_platform_skill_smoke.sh
  ```
  Validate:
  - Codex can inspect, clone, backtest, compare
  - Claude can inspect, clone, backtest, compare
  - neither can directly bypass promotion or exec controls

- [ ] **Step 3: Write evidence report**
  The final report must contain:
  - exact commands
  - output snippets
  - artifact paths
  - Bitget order ids
  - final position-flat proof
  - Telegram/DingTalk delivery proof
  - OpenClaw session proof

---

## Completion Gates

- `go test ./...` passes.
- `go build ./cmd/platformd ./cmd/marketd ./cmd/traderd ./cmd/execd ./cmd/notifierd ./cmd/researchd ./cmd/platformctl ./cmd/mcpd` passes.
- Warehouse ingest is repeatable and idempotent.
- Strategy bundle validation is enforced.
- `traderd` no longer writes orders directly.
- `execd` is the only Bitget write path.
- Telegram and DingTalk are both real.
- OpenClaw is integrated and usable.
- Codex and Claude both have a working platform skill workflow.
- Real Bitget lifecycle passes and exits flat.

## Residual Risks To Watch During Execution

- TimescaleDB operational complexity on a single ECS.
- Telegram inbound ownership when OpenClaw and platform notifier share the same bot token.
- CEL bundle design drifting into an underpowered DSL.
- Warehouse sync lag causing stale AI reads if not instrumented.
- Feature materialization cost if backfilled too broadly without batching.
