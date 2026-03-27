# Price Action Trigger Features

## Goal

Encode bar-level triggers such as engulfing, rejection, and breakout/retest as reusable inputs to the strategy layer.

## Inputs

- `bars []core.Bar`
- `idx int`
- `atr_window int`
- optional `level_clusters`
- optional `fib_context`

## Outputs

- `bullish_engulfing_flag bool`
- `bearish_engulfing_flag bool`
- `pin_bar_bull_score float64`
- `pin_bar_bear_score float64`
- `close_location_value float64`
- `range_expansion_ratio float64`
- `break_retest_flag bool`
- `trigger_quality_score float64`

## Extraction Logic

1. Detect deterministic bar patterns from the last one to three bars.
2. Normalize bar size by ATR.
3. Measure close location inside the bar range.
4. Reward triggers that occur near support/resistance or Fib bands.
5. Penalize triggers that appear in the middle of nowhere.

## Edge Cases

- Zero-range bars should return neutral scores
- Event bars with huge gaps should be tagged but down-weighted
- Consecutive engulfing signals in opposite directions should reduce quality

## Failure Modes

- Pattern names without context overfit easily
- Trigger-only systems often die after fees
- Different markets may need the same definitions but different tolerance settings at the feature scaling layer

## Unit Test Plan

- Canonical bullish engulfing sample should set `bullish_engulfing_flag=true`
- Canonical bearish engulfing sample should set `bearish_engulfing_flag=true`
- Long lower wick near support should lift `pin_bar_bull_score`
- Middle-of-range close should depress `trigger_quality_score`

## Snapshot Test Plan

- `BTCUSDT` 2024-02-24 bullish engulfing
- `ETHUSDT` 2024-03-14 bearish engulfing
- `CRCL` 2025-07-18 bearish engulfing

