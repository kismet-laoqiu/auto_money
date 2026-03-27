# 综合交易系统

## 这是什么

综合交易系统不是把艾略特波浪、价格行为、斐波那契、支撑阻力各自做成四个按钮，然后“信号越多越好”。真正好的系统，是把这些理论拆成不同层次的信息：

1. `regime`：现在是趋势、震荡，还是高波动失真？
2. `location`：价格现在靠近哪里？支撑、阻力、Fib zone、旧 pivot？
3. `structure`：走势是 higher-high / higher-low，还是 lower-low / lower-high？
4. `trigger`：在那个位置，价格有没有给出明确行为，比如 engulfing、break/retest、假突破？
5. `risk`：如果判断错了，什么时候退出？

## 为什么有人用

因为单一理论太脆。只看 wave，容易数浪成瘾；只看 candlestick，容易把噪音当信号；只看 Fib，容易画线画到自己满意；只看支撑阻力，又容易在趋势市场里不断抄底摸顶。综合系统的价值，就是让这些理论彼此纠错。

## 什么时候容易有效

1. 你先分层，再组合，而不是把所有概念揉成一个模糊分数。
2. 你愿意接受“少做”，只在 confluence 真正出现时出手。
3. 你把交易成本、滑点、持仓时间上限一起设计进去。
4. 你做的是跨资产验证，不是只优化一张图。

## 什么时候最容易失灵

1. 你把 confluence 变成“理由越多越能下单”的故事机。
2. 你为了提升回测，偷偷给不同资产塞不同参数。
3. 你把文档里的解释层直接复制成交易规则，没有先做 feature layer。
4. 你没有单独处理 regime，结果趋势和震荡共用一套入口。

## 用真实历史数据举例

### 例子 1：BTCUSDT

BTC 在 2024-02-24 出现 bullish engulfing，后续 5 日收益约 +18.54%。如果只看 K 线，这是一个 trigger；如果再叠加 wave-like 推进背景和最近的关键价位，这个 trigger 的解释力会更强。

### 例子 2：ETHUSDT

ETH 在 2025-04-09 到 2025-04-16 之间出现了清晰的 50% Fib 回踩，后续 10 日约 +15.45%。这说明 Fib 更适合做 `location feature`，然后等 Price Action 做 `trigger feature`。

### 例子 3：CRCL

CRCL 在 2025-07-18 出现 bearish engulfing，后续 5 日约 -13.82%。这个例子说明美股单票波动能非常快，所以 stock bucket 的 risk model 不能照搬 BTC/ETH。

### 例子 4：XAUUSD

XAUUSD 在最近样本里有明显的 3261 一带低点 cluster，也有 2025-05-29 附近 50% Fib 回踩。黄金更适合拿来展示“location 很重要，但 trigger 不能省略”。

## 如何转成机器规则

正确顺序不是“写策略”，而是“先写 feature”：

1. `wave_structure_features`
   例如 pivot sequence、推进/回撤比、higher-high/higher-low 连续性。
2. `level_cluster_features`
   例如 cluster count、bounce count、zone width、distance to zone。
3. `fib_confluence_features`
   例如 distance to 38.2/50/61.8、是否与 zone 重叠。
4. `price_action_trigger_features`
   例如 engulfing、pin bar、收盘位置、break/retest。
5. `volume_confirmation_features`
   例如 breakout 当日量能是否放大。
6. `regime_tags`
   例如 trend、range、high-volatility、event-risk。

然后策略层只做两件事：

- 如何把 feature 组合成 entry / exit score。
- 如何把风险控制成可回测、可执行的仓位和止损规则。

## 如何避免过拟合

1. 先做 feature，再做 rule，能显著减少“为了某次胜利硬改逻辑”。
2. 评分必须跨 bucket：crypto、us_equity、commodity 同时看。
3. 只看平均值不够，还要看 median、worst bucket、positive return ratio。
4. 每一轮实验都要有 hard gate：`go test`、`go build ./cmd/stream`、`lab eval`、stream smoke test。
5. 如果某个理论只有讲故事能力，没有稳定 feature 证据，就不要硬进策略。

## 术语表

- `feature`：机器可直接读取的数值化信号。
- `confluence`：多个独立证据落在同一区域。
- `regime`：市场所处的大环境，如趋势或震荡。
- `bucket`：按资产类别分组的验证集合。
- `hard gate`：任何实验都必须通过的基础质量门槛。

## 来源索引

- `EW-S-001`, `EW-S-002`, `TA-C-001`
- `PA-S-001`, `PA-S-002`, `PA-C-001`, `PA-C-002`
- `FB-S-001`, `FB-C-001`
- `SR-S-001`, `SR-S-002`

这些来源共同支持一个结论：单一理论很容易讲故事，系统化研究必须先转成 feature，再做 out-of-sample 检验。
