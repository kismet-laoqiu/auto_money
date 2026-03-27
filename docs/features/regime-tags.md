# Regime Tags

## Goal

Classify market environment before any entry logic runs, so the strategy can avoid applying trend tools to range regimes and vice versa.

## Inputs

- `bars []core.Bar`
- `idx int`
- `fast_window int`
- `slow_window int`
- `atr_window int`
- `lookback int`

## Outputs

- `trend_up_flag bool`
- `trend_down_flag bool`
- `range_flag bool`
- `high_vol_flag bool`
- `compression_flag bool`
- `regime_score float64`

## Extraction Logic

1. Use moving-average slope and separation to estimate directional bias.
2. Use ATR percentile to estimate volatility state.
3. Use recent cluster density and swing overlap to estimate range behavior.
4. Combine these into mutually interpretable tags, not a black-box label.

## Edge Cases

- Regime transitions should not flip every bar; hysteresis or smoothing is needed
- New IPO-style datasets such as `CRCL` can spend long periods in unstable high-vol regimes
- Missing or sparse volume must not be required for regime labeling

## Failure Modes

- If regime tags are too reactive, they become noise
- If regime tags are too slow, they miss transitions and hold stale bias
- Mixing trend and range rules without regime separation creates hidden branching complexity

## Unit Test Plan

- Rising moving averages plus low overlap should tag `trend_up_flag=true`
- Falling moving averages should tag `trend_down_flag=true`
- High overlap plus narrow ATR percentile should tag `range_flag=true`
- Large ATR spike should tag `high_vol_flag=true`

## Snapshot Test Plan

- `BTCUSDT` uptrend windows around 2025-07-02 to 2025-07-15
- `ETHUSDT` high-vol rebound around 2025-04-09 to 2025-04-16
- `CRCL` early listing windows should show unstable high-vol behavior
- `XAUUSD` quieter cluster periods should show `range_flag=true`

