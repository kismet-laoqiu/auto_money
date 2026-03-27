# Fib Confluence Features

## Goal

Turn Fibonacci retracement into deterministic confluence signals anchored by explicit swing rules.

## Inputs

- `bars []core.Bar`
- `idx int`
- `pivot_window int`
- `swing_lookback int`
- `fib_tolerance float64`
- `level_clusters`: optional output from `level-cluster`

## Outputs

- `fib_382_distance_pct float64`
- `fib_500_distance_pct float64`
- `fib_618_distance_pct float64`
- `fib_zone_hit_count int`
- `fib_cluster_overlap_score float64`
- `active_swing_direction string`
- `valid_fib_context bool`

## Extraction Logic

1. Find the latest valid swing high/low pair from deterministic pivots.
2. Compute 38.2%, 50.0%, and 61.8% retracement bands.
3. Measure current price distance to each band.
4. Add confluence score if Fib bands overlap with support/resistance zones.
5. Return context only if the anchor swing is recent and sufficiently large.

## Edge Cases

- Tiny swings should not generate Fib context
- Overlapping swings can create multiple anchor candidates; choose the most recent dominant swing by amplitude
- Gap assets should use bands, not single exact prices

## Failure Modes

- Manual anchor picking destroys reproducibility
- Using Fib without trend context creates many false positives in chop
- Exact-level matching is too brittle for live trading

## Unit Test Plan

- Up-swing followed by 50% pullback should reduce `fib_500_distance_pct`
- Down-swing followed by 61.8% rally should reduce `fib_618_distance_pct`
- Small swings should return `valid_fib_context=false`
- Overlap with a cluster should increase `fib_cluster_overlap_score`

## Snapshot Test Plan

- `ETHUSDT` 2025-04-09 to 2025-04-16
- `CRCL` 2025-09-05 to 2025-09-26
- `XAUUSD` 2025-05-15 to 2025-05-29

