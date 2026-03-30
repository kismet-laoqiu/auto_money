# Export Read Contract

## Purpose

`artifacts/platform/export/` stores read-only Parquet datasets for Codex, Claude, DuckDB, and offline analysis. Writers regenerate a fresh timestamped snapshot under the export root. Readers must treat every exported file as immutable.

## Layout

- Latest run summary: `artifacts/platform/export/summary.json`
- One snapshot per run: `artifacts/platform/export/<UTC timestamp>/`
- Manifest SQL per run: `artifacts/platform/export/<UTC timestamp>/manifest.sql`
- Dataset summary per run: `artifacts/platform/export/<UTC timestamp>/summary.json`
- Partition rule: `data/provider/<provider>/symbol/<symbol>/interval/<interval>/date/<YYYY-MM-DD>/bars.parquet`

## Reader Rules

- Open DuckDB from the repository root. The current `manifest.sql` uses repository-relative Parquet paths.
- Read through `manifest.sql` or directly through the partitioned Parquet files. Do not mutate exported files in place.
- Use `summary.json` to discover the latest snapshot root, file count, byte size, and dataset row counts before scanning Parquet.
- Treat `overwrite_strategy=dataset-replace` as the contract: a rerun replaces every file under the same `provider/symbol/interval` dataset in the new snapshot.
- Readers should prefer the four verified symbols from `P04`: `BTCUSDT`, `ETHUSDT`, `SOLUSDT`, `MSTRUSDT`.

## DuckDB Example

Run from `/root/.config/superpowers/worktrees/quant-lab/autoresearch-20260328-all-plan`:

```bash
run_root=$(jq -r '.latest_root_dir' artifacts/platform/export/summary.json)
{
  cat "$run_root/manifest.sql"
  printf '\nselect count(*) from market_bars_export;\n'
  printf 'select symbol, interval, count(*) from market_bars_export group by 1,2 order by 1,2;\n'
} | /tmp/duckdb-cli/duckdb
```

## Operational Notes

- Historical sync must populate `1m` and `1h` first.
- Aggregation derives `5m/15m/1h` from `1m` and `4h/1d` from `1h`.
- Export is idempotent at the dataset level: rerun creates a new timestamped snapshot and rewrites every file for each exported `provider/symbol/interval` dataset.
- Validation evidence for the current snapshot lives in `/tmp/p04-export.json`, `/tmp/p04-export-rerun.json`, warehouse SQL output, and DuckDB CLI output captured in the terminal session.
