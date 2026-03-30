---
name: warehouse-pg-readonly
description: Use when inspecting this repository's Timescale/Postgres warehouse for market_bars coverage, freshness, gaps, symbol coverage, interval coverage, or other read-only SQL analysis through the repo's docker compose database
---

# Warehouse PG Readonly

## Overview
Read the warehouse through the repository's own runtime wiring only. The fixed truth sources are `configs/platform/warehouse.yaml`, `configs/platform/watchlist.yaml`, and `deploy/docker-compose.platform.yml`.

## Guardrails
- Read-only only. Do not run `INSERT`, `UPDATE`, `DELETE`, `ALTER`, `TRUNCATE`, `DROP`, or migrations from this skill.
- Default scope is the current watchlist symbols and `historical_intervals`.
- Standard table is `market_bars`; use real column names from `migrations/postgres/0001_init_market.sql`.

## Commands
- Coverage:
  - `.agents/skills/warehouse-pg-readonly/scripts/warehouse_query.sh coverage`
- Freshness:
  - `.agents/skills/warehouse-pg-readonly/scripts/warehouse_query.sh freshness`
- Gap analysis:
  - `.agents/skills/warehouse-pg-readonly/scripts/warehouse_query.sh gaps`
- Symbol coverage:
  - `.agents/skills/warehouse-pg-readonly/scripts/warehouse_query.sh symbol-coverage`
- Interval coverage:
  - `.agents/skills/warehouse-pg-readonly/scripts/warehouse_query.sh interval-coverage`
- Custom read-only SQL:
  - `.agents/skills/warehouse-pg-readonly/scripts/warehouse_query.sh sql --sql "select count(*) from market_bars;"`

## Useful Flags
- `--provider bitget`
- `--symbols BTCUSDT,ETHUSDT`
- `--intervals 15m,1h,4h,1d,1w`
- `--all-symbols`
- `--all-intervals`
- `--limit 50`

## Notes
- The script parses `configs/platform/warehouse.yaml` to locate the warehouse connection identity.
- The script defaults to the current watchlist, so newly added symbols are visible to analysis as soon as `configs/platform/watchlist.yaml` changes.
- Queries run through `docker compose -f deploy/docker-compose.platform.yml exec -T warehouse-db ... psql` with `default_transaction_read_only=on`.
