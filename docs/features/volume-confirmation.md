# Volume Confirmation Features

## Goal

Measure whether a structural or trigger event is supported by unusual participation rather than by weak tape.

## Inputs

- `bars []core.Bar`
- `idx int`
- `volume_window int`
- optional `trigger_features`

## Outputs

- `volume_zscore float64`
- `relative_volume_ratio float64`
- `pullback_volume_dryup_flag bool`
- `breakout_volume_confirmed bool`
- `volume_available bool`

## Extraction Logic

1. Compute rolling mean or median volume.
2. Compare current volume with the rolling baseline.
3. On breakout bars, check whether relative volume is above threshold.
4. On pullback bars, check whether volume contracts while price retraces.

## Edge Cases

- `XAUUSD` daily data has zero reported volume in the current source; return `volume_available=false`
- Newly listed assets may not have enough lookback for stable z-scores
- Exchange outages or data glitches can create false spikes

## Failure Modes

- Missing volume data silently treated as real zero can poison features
- High volume without location context is not enough
- Different providers report volume differently, so cross-provider normalization matters

## Unit Test Plan

- Constant baseline plus one spike should create positive `volume_zscore`
- Breakout with high relative volume should set `breakout_volume_confirmed=true`
- Pullback on shrinking volume should set `pullback_volume_dryup_flag=true`
- Zero-volume provider rows should set `volume_available=false`

## Snapshot Test Plan

- `BTCUSDT` breakout windows around 2024-02-24
- `ETHUSDT` rebound windows around 2025-04-16
- `XAUUSD` daily rows should snapshot as `volume_available=false`

