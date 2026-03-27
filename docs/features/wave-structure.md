# Wave Structure Features

## Goal

Translate Elliott-style market structure into deterministic features built from `[]core.Bar` and pivot sequences. This layer must describe structure; it must not emit direct trade directions.

## Inputs

- `bars []core.Bar`
- `idx int`: feature evaluation index
- `pivot_window int`
- `lookback int`
- `min_swing_pct float64`

## Outputs

- `wave_up_score float64`: strength of higher-high / higher-low continuation
- `wave_down_score float64`: strength of lower-low / lower-high continuation
- `impulse_extension_ratio float64`: latest impulse length divided by prior impulse length
- `corrective_depth_ratio float64`: latest correction depth divided by prior impulse length
- `swing_overlap_ratio float64`: overlap between adjacent swing ranges
- `structure_age_bars int`: bars since the oldest pivot used in the current structure
- `valid_structure bool`

## Extraction Logic

1. Detect pivots with a fixed `pivot_window`.
2. Keep only pivots inside `lookback`.
3. Build alternating pivot sequences.
4. Score trend continuity from the last five to seven pivots.
5. Penalize overlap, broken higher lows, broken lower highs, and deep corrections.

## Edge Cases

- Less than 5 pivots in window: `valid_structure=false`
- Equal highs/lows: treat as low-confidence continuation, not as clean structure
- Very young assets such as `CRCL`: allow shorter age but do not bypass minimum pivot count
- One-bar news spikes: do not treat a single outlier wick as a full wave transition without pivot confirmation

## Failure Modes

- Subjective relabeling if pivot rules change per asset
- Too-small pivot window creates fake waves in noisy data
- Too-large pivot window misses real structural turns in faster markets

## Unit Test Plan

- Rising synthetic pivot sequence should yield `wave_up_score > wave_down_score`
- Falling synthetic pivot sequence should yield `wave_down_score > wave_up_score`
- Deep correction beyond prior swing low should collapse `wave_up_score`
- Insufficient pivots should return `valid_structure=false`

## Snapshot Test Plan

- `BTCUSDT` 2025-07-02 to 2025-07-15
- `ETHUSDT` 2025-04-30 to 2025-05-18
- `CRCL` 2025-10-10 to 2025-12-12

