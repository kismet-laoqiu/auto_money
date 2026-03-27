# Quant Lab Research Roadmap

## Mission

Build a clean, evidence-driven multi-asset quant research system on ECS.
The system must combine Elliott Wave, Price Action, Fibonacci, Support/Resistance,
and Volume into a testable workflow covering:

1. English evidence collection.
2. Chinese beginner-friendly theory documents.
3. Feature specifications and tests.
4. Backtest and live websocket evaluation.
5. A 24-round measured autoresearch loop.

## Hard Constraints

- All search, coding, testing, and execution happen on `root@47.250.138.143:/root/quant-lab`.
- Local machine only keeps mirrored `docs/` and `plan/` for reading.
- CRCL is treated as a US stock, not a crypto pair.
- New system/package dependencies are allowed when they improve execution quality.
- Code must stay clean, fast, and easy to reason about.

## Confirmed Measurement Contract

- Goal: improve robust multi-asset out-of-sample strategy quality and live alert quality.
- Metric command: `PATH=/usr/local/go/bin:$PATH ./scripts/measure.sh`
- Metric extraction: parse `FINAL_SCORE=<float>` from stdout.
- Direction: higher is better.
- Experiment budget: 24.
- Loop branch: continue on `autoresearch/mar26`.

## Score Design

`FINAL_SCORE` will be based on robust out-of-sample quality instead of a plain mean.
The first implementation target is:

- 55% median of per-dataset objective scores
- 25% minimum bucket mean score
- 20% positive out-of-sample return ratio

Buckets:

- `crypto`: BTCUSDT, ETHUSDT
- `us_equity`: CRCL, BABA, PDD, ZM, MSTR, BMNR
- `commodity`: XAUUSD

Hard gates:

- `go test ./...` must pass.
- `go build ./cmd/stream` must pass.
- `lab eval` must complete successfully.
- `stream` smoke test must stay alive until interrupted.

## Delivery Phases

### Phase 1: Evidence Engineering

Deliverables:

- `docs/research/raw/`
- `docs/research/sources.tsv`
- `docs/research/claims.md`

Rules:

- Every theory needs support evidence and counter-evidence.
- Prefer books, academic work, exchange docs, broker research, and recognized institutions.
- Low-quality blogs can seed search, but cannot anchor conclusions.

### Phase 2: Chinese Theory Docs

Deliverables:

- `docs/zh/elliott-wave.md`
- `docs/zh/price-action.md`
- `docs/zh/fibonacci.md`
- `docs/zh/support-resistance.md`
- `docs/zh/system-design.md`

Each document must include:

- What it is
- Why traders use it
- When it tends to work
- When it fails
- Real BTC / ETH / CRCL / XAU historical examples
- How to translate it into machine features
- How to avoid overfitting
- Glossary
- Source index

### Phase 3: Feature Contracts

Deliverables:

- `docs/features/wave-structure.md`
- `docs/features/level-cluster.md`
- `docs/features/fib-confluence.md`
- `docs/features/price-action-trigger.md`
- `docs/features/volume-confirmation.md`
- `docs/features/regime-tags.md`

Each feature spec must define:

- Inputs and outputs
- Edge cases
- Failure modes
- Unit test plan
- Snapshot test plan

### Phase 4: Measured Autoresearch

24 experiments grouped into six batches:

1. metric and gate corrections
2. level quality and rejection quality
3. wave filter upgrades
4. price action trigger upgrades
5. volume and regime filters
6. winner combination and simplification

## Sync Rules

After each meaningful milestone:

1. Keep canonical files on ECS.
2. Mirror `docs/` and `plan/` to the local directory.
3. Do not treat the local mirror as a development repo.

## Current Status

- ECS environment fixed: Go path, ripgrep, rsync, Python 3.11, venv.
- Quant repo branch: `autoresearch/mar26`.
- Research documents: not started before this roadmap.
