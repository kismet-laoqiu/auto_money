# Level Cluster Features

## Goal

Represent support and resistance as price zones derived from repeated pivots, not as hand-drawn single lines.

## Inputs

- `bars []core.Bar`
- `idx int`
- `pivot_window int`
- `cluster_lookback int`
- `level_tolerance float64`

## Outputs

- `nearest_support_distance_pct float64`
- `nearest_resistance_distance_pct float64`
- `support_cluster_count int`
- `resistance_cluster_count int`
- `support_bounce_count int`
- `resistance_bounce_count int`
- `zone_width_pct float64`
- `recent_breakout_flag bool`

## Extraction Logic

1. Detect recent pivot highs and lows.
2. Group pivots into zones if price distance is within `level_tolerance`.
3. Separate zones by dominant type: support, resistance, mixed.
4. Measure current distance to the nearest zone.
5. Track whether the last interaction was a bounce or a clean breakout.

## Edge Cases

- Mixed zones with both highs and lows should keep separate bounce counts
- Fast trend markets can leave stale zones behind; age must be included in filtering
- Assets with sparse history should down-weight cluster confidence

## Failure Modes

- Treating a zone as a point creates unstable hits
- Ignoring breakout confirmation turns trends into false mean-reversion signals
- Reusing ancient zones in a new regime creates stale context

## Unit Test Plan

- Repeated lows around the same area should merge into one support cluster
- Repeated highs around the same area should merge into one resistance cluster
- Wide zones beyond tolerance should split into two clusters
- Closing above the zone after multiple failed tests should set `recent_breakout_flag=true`

## Snapshot Test Plan

- `ETHUSDT` recent cluster around 3058.52
- `XAUUSD` recent low cluster around 3261.37
- `BTCUSDT` recent mixed cluster around 107019.78

