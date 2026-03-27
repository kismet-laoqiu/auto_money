# Research Claims Ledger

This file tracks claims that will later appear in the Chinese theory documents.
Each claim links to at least one support source and, where relevant, one
counter-evidence source.

## Elliott Wave

| claim_id | claim | support_sources | counter_sources | status | notes |
| --- | --- | --- | --- | --- | --- |
| EW-C1 | Elliott Wave is commonly framed as a sequence of impulsive and corrective swings, and practitioners often combine wave counts with retracement logic. | EW-S-001 | TA-C-001 | supported | Good for concept teaching; not enough for standalone systematic trading. |
| EW-C2 | One academic study on EUR/USD reported that Elliott-based forecasting achieved high accuracy over its sample window. | EW-S-002 | TA-C-001 | contested | Single market and sample period; not a robust universal proof. |
| EW-C3 | The main risk in wave usage is subjectivity: different analysts can count different waves on the same chart, so machine rules must avoid discretionary relabeling. | EW-S-001 | TA-C-001 | supported | Feature design should use pivots, swing alternation, and retracement depth instead of manual wave labels. |

## Price Action

| claim_id | claim | support_sources | counter_sources | status | notes |
| --- | --- | --- | --- | --- | --- |
| PA-C1 | Price action frameworks focus on OHLC bars, local structure, break/retest behavior, and the interaction between trend and key levels. | PA-S-001 | TA-C-001 | supported | Useful as a human-readable concept layer and as a source of trigger features. |
| PA-C2 | Some candlestick studies report economically meaningful profitability, especially when trend definition and holding rules are chosen carefully. | PA-S-002 | PA-C-001, PA-C-002 | conditional | Strong sample dependence; trend definition changes results materially. |
| PA-C3 | Other studies find no statistically significant predictive power or no net positive returns after costs for candlestick-based rules. | PA-C-001, PA-C-002 | PA-S-002 | supported | This is the core warning against naive candlestick overfitting. |

## Fibonacci

| claim_id | claim | support_sources | counter_sources | status | notes |
| --- | --- | --- | --- | --- | --- |
| FB-C1 | Fibonacci retracement is used to mark potential pullback zones rather than exact reversal points. | FB-S-001 | FB-C-001 | supported | The levels are better treated as price zones with tolerance bands. |
| FB-C2 | Fibonacci levels are usually considered more useful when combined with trend, support/resistance, or other confirmations. | FB-S-001 | FB-C-001 | supported | Good candidate for confluence features instead of standalone entries. |
| FB-C3 | The main criticism is that the chosen swing high/low is subjective, so different anchors can produce different levels and narratives. | FB-C-001 | FB-S-001 | supported | Machine rules must specify deterministic swing detection. |

## Support / Resistance

| claim_id | claim | support_sources | counter_sources | status | notes |
| --- | --- | --- | --- | --- | --- |
| SR-C1 | Published support/resistance levels from FX market participants predicted intraday trend interruptions better than arbitrary levels in one FRBNY study. | SR-S-001 | TA-C-001 | supported | Strong source, but domain is intraday FX in a specific period. |
| SR-C2 | Algorithmically discovered support/resistance levels with more prior bounces show statistically significant bounce behavior in intraday data. | SR-S-002 | TA-C-001 | supported | Encourages feature engineering around bounce count, age, and distance. |
| SR-C3 | Even when support/resistance effects exist, portability across assets and timeframes should not be assumed without out-of-sample testing. | SR-S-001, SR-S-002 | TA-C-001 | supported | This is the bridge from research evidence to cautious feature design. |

## Integrated System Design

| claim_id | claim | support_sources | counter_sources | status | notes |
| --- | --- | --- | --- | --- | --- |
| SYS-C1 | Technical-analysis evidence is mixed overall: positive pockets exist, but many results weaken after costs, different trend definitions, or new samples. | TA-C-001, PA-C-001, PA-C-002 | EW-S-002, PA-S-002, SR-S-001, SR-S-002 | supported | This is the central anti-overfitting premise for the project. |
| SYS-C2 | Theories should be translated into explicit features and evaluated out-of-sample, rather than copied as discretionary trading folklore. | TA-C-001, SR-S-002 | EW-S-001, PA-S-001, FB-S-001 | supported | Matches the project requirement to build features before rules. |
| SYS-C3 | Confluence, regime awareness, and transaction-cost realism are more defensible than single-pattern rules. | FB-S-001, SR-S-001, PA-C-002 | TA-C-001 | conditional | This claim guides feature composition and later measured experiments. |
