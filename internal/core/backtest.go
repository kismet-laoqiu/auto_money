package core

import (
	"math"
	"strings"
	"time"

	"quantlab/internal/config"
)

type activePosition struct {
	Side       Side
	EntryPrice float64
	EntryTime  time.Time
	EntryIndex int
	Stop       float64
	Target     float64
	Score      float64
	Reasons    string
}

func BacktestDataset(dataset Dataset, strategy config.StrategyConfig, objective config.ObjectiveConfig) Report {
	report := Report{
		Name:     dataset.Name,
		Provider: dataset.Provider,
		Symbol:   dataset.Symbol,
		Interval: dataset.Interval,
		Bars:     len(dataset.Bars),
	}
	if len(dataset.Bars) < WarmupBars(strategy)+20 {
		return report
	}

	split := int(float64(len(dataset.Bars)) * 0.6)
	minSplit := WarmupBars(strategy) + 10
	if split < minSplit {
		split = minSplit
	}
	if split > len(dataset.Bars)-10 {
		split = len(dataset.Bars) - 10
	}
	warmup := WarmupBars(strategy)
	start := split - warmup
	if start < 0 {
		start = 0
	}

	inStats, _, _, _ := runSegment(dataset.Bars[:split], dataset.Interval, 0, strategy, objective.RiskFreeRate)
	outStats, trades, equity, signals := runSegment(dataset.Bars[start:], dataset.Interval, split-start, strategy, objective.RiskFreeRate)
	for i := range trades {
		trades[i].Symbol = dataset.Symbol
	}

	report.InSample = inStats
	report.OutOfSample = outStats
	report.Trades = trades
	report.EquityCurve = equity
	report.SignalCount = signals
	report.ObjectiveScore = ObjectiveScore(inStats, outStats, objective)
	return report
}

func runSegment(bars []Bar, interval string, tradableStart int, cfg config.StrategyConfig, riskFreeRate float64) (Stats, []Trade, []EquityPoint, int) {
	if len(bars) == 0 {
		return Stats{}, nil, nil, 0
	}

	cash := 1.0
	prevEquity := 1.0
	returns := make([]float64, 0, len(bars)-1)
	equityCurve := make([]EquityPoint, 0, len(bars))
	trades := make([]Trade, 0, len(bars)/10)
	signalCount := 0
	cooldownUntil := -1
	var open *activePosition
	var pending *Signal

	for i := 0; i < len(bars); i++ {
		bar := bars[i]
		justEntered := false
		if pending != nil && open == nil && i >= tradableStart {
			open = &activePosition{
				Side:       pending.Side,
				EntryPrice: applySlippage(bar.Open, pending.Side, cfg.SlippageBps),
				EntryTime:  bar.Time,
				EntryIndex: i,
				Stop:       pending.Stop,
				Target:     pending.Target,
				Score:      pending.Score,
				Reasons:    JoinReasons(pending.Reasons),
			}
			pending = nil
			justEntered = true
		}

		if open != nil && !justEntered {
			exitPrice, exitReason, shouldExit := exitCheck(*open, bar, i, cfg)
			if shouldExit {
				closeSide := Long
				if open.Side == Long {
					closeSide = Short
				}
				effectiveExit := applySlippage(exitPrice, closeSide, cfg.SlippageBps)
				netReturn := tradeReturn(open.Side, open.EntryPrice, effectiveExit) - 2*cfg.CommissionBps/10000
				cash *= 1 + netReturn
				if cash < 0 {
					cash = 0
				}
				trades = append(trades, Trade{
					Side:       open.Side,
					EntryTime:  open.EntryTime,
					ExitTime:   bar.Time,
					EntryPrice: open.EntryPrice,
					ExitPrice:  effectiveExit,
					Stop:       open.Stop,
					Target:     open.Target,
					Return:     netReturn,
					BarsHeld:   i - open.EntryIndex,
					Reason:     exitReason + " | " + open.Reasons,
					Score:      open.Score,
				})
				open = nil
				cooldownUntil = i + cfg.CooldownBars
			}
		}

		currentEquity := cash
		if open != nil {
			mtm := tradeReturn(open.Side, open.EntryPrice, bar.Close) - 2*cfg.CommissionBps/10000
			currentEquity = cash * (1 + mtm)
			if currentEquity < 0 {
				currentEquity = 0
			}
		}
		if len(equityCurve) > 0 {
			if prevEquity == 0 {
				returns = append(returns, 0)
			} else {
				returns = append(returns, currentEquity/prevEquity-1)
			}
		}
		equityCurve = append(equityCurve, EquityPoint{Time: bar.Time, Equity: currentEquity})
		prevEquity = currentEquity

		if open == nil && pending == nil && i >= tradableStart && i > cooldownUntil && i < len(bars)-1 {
			signal := EvaluateSignal(bars[:i+1], i, cfg)
			if signal.Side != Flat {
				signalCount++
				pending = &signal
			}
		}
	}

	if open != nil {
		last := bars[len(bars)-1]
		closeSide := Long
		if open.Side == Long {
			closeSide = Short
		}
		effectiveExit := applySlippage(last.Close, closeSide, cfg.SlippageBps)
		netReturn := tradeReturn(open.Side, open.EntryPrice, effectiveExit) - 2*cfg.CommissionBps/10000
		cash *= 1 + netReturn
		if cash < 0 {
			cash = 0
		}
		trades = append(trades, Trade{
			Side:       open.Side,
			EntryTime:  open.EntryTime,
			ExitTime:   last.Time,
			EntryPrice: open.EntryPrice,
			ExitPrice:  effectiveExit,
			Stop:       open.Stop,
			Target:     open.Target,
			Return:     netReturn,
			BarsHeld:   len(bars) - 1 - open.EntryIndex,
			Reason:     "end_of_data | " + open.Reasons,
			Score:      open.Score,
		})
		equityCurve[len(equityCurve)-1].Equity = cash
		if len(equityCurve) > 1 {
			previous := equityCurve[len(equityCurve)-2].Equity
			if previous == 0 {
				returns[len(returns)-1] = 0
			} else {
				returns[len(returns)-1] = cash/previous - 1
			}
		}
	}

	return buildStats(cash, returns, trades, equityCurve, interval, riskFreeRate), trades, equityCurve, signalCount
}

func exitCheck(position activePosition, bar Bar, idx int, cfg config.StrategyConfig) (float64, string, bool) {
	if position.Side == Long {
		if bar.Low <= position.Stop {
			return position.Stop, "stop_loss", true
		}
		if bar.High >= position.Target {
			return position.Target, "take_profit", true
		}
	} else {
		if bar.High >= position.Stop {
			return position.Stop, "stop_loss", true
		}
		if bar.Low <= position.Target {
			return position.Target, "take_profit", true
		}
	}
	if idx-position.EntryIndex >= cfg.MaxHoldBars {
		return bar.Close, "time_stop", true
	}
	return 0, "", false
}

func buildStats(finalEquity float64, returns []float64, trades []Trade, equityCurve []EquityPoint, interval string, riskFreeRate float64) Stats {
	stats := Stats{TotalReturn: finalEquity - 1, Trades: len(trades)}
	if len(equityCurve) == 0 {
		return stats
	}

	periods := periodsPerYear(interval)
	years := 0.0
	if periods > 0 {
		years = float64(len(returns)) / periods
	}
	if years > 0 && finalEquity > 0 {
		stats.CAGR = math.Pow(finalEquity, 1/years) - 1
	}

	mean := average(returns)
	stats.MaxDrawdown = maxDrawdown(equityCurve)
	if stats.MaxDrawdown > 0 {
		stats.Calmar = stats.CAGR / stats.MaxDrawdown
	}
	std := stddev(returns, mean)
	if std > 0 && periods > 0 {
		excess := mean - riskFreeRate/periods
		stats.Sharpe = excess / std * math.Sqrt(periods)
	}

	wins := 0
	profitSum := 0.0
	lossSum := 0.0
	tradeMean := 0.0
	for _, trade := range trades {
		tradeMean += trade.Return
		if trade.Return > 0 {
			wins++
			profitSum += trade.Return
		} else if trade.Return < 0 {
			lossSum += math.Abs(trade.Return)
		}
	}
	if len(trades) > 0 {
		stats.WinRate = float64(wins) / float64(len(trades))
		stats.AverageReturn = tradeMean / float64(len(trades))
	}
	if lossSum > 0 {
		stats.ProfitFactor = profitSum / lossSum
	}
	return stats
}

func ObjectiveScore(inSample, outOfSample Stats, objective config.ObjectiveConfig) float64 {
	overfitPenalty := 0.0
	if inSample.Sharpe > outOfSample.Sharpe {
		overfitPenalty += (inSample.Sharpe - outOfSample.Sharpe) * objective.OverfitPenaltyWeight
	}
	if inSample.TotalReturn > outOfSample.TotalReturn {
		overfitPenalty += (inSample.TotalReturn - outOfSample.TotalReturn) * objective.OverfitPenaltyWeight * 0.5
	}
	return outOfSample.Sharpe + objective.CalmarWeight*outOfSample.Calmar + objective.PnLWeight*outOfSample.TotalReturn - objective.DrawdownWeight*outOfSample.MaxDrawdown - overfitPenalty
}

func AggregateReports(reports []Report) Aggregate {
	aggregate := Aggregate{Metrics: map[string]float64{}}
	if len(reports) == 0 {
		return aggregate
	}
	for _, report := range reports {
		aggregate.ObjectiveScore += report.ObjectiveScore
		aggregate.Metrics["avg_oos_sharpe"] += report.OutOfSample.Sharpe
		aggregate.Metrics["avg_oos_calmar"] += report.OutOfSample.Calmar
		aggregate.Metrics["avg_oos_return"] += report.OutOfSample.TotalReturn
		aggregate.Metrics["avg_oos_drawdown"] += report.OutOfSample.MaxDrawdown
		aggregate.Metrics["avg_trade_count"] += float64(report.OutOfSample.Trades)
	}
	count := float64(len(reports))
	aggregate.ObjectiveScore /= count
	aggregate.Metrics["dataset_count"] = count
	for key, value := range aggregate.Metrics {
		if key == "dataset_count" {
			continue
		}
		aggregate.Metrics[key] = value / count
	}
	return aggregate
}

func applySlippage(price float64, side Side, bps float64) float64 {
	shift := bps / 10000
	switch side {
	case Long:
		return price * (1 + shift)
	case Short:
		return price * (1 - shift)
	default:
		return price
	}
}

func tradeReturn(side Side, entry, exit float64) float64 {
	if entry <= 0 || exit <= 0 {
		return 0
	}
	if side == Long {
		return exit/entry - 1
	}
	if side == Short {
		return (entry - exit) / entry
	}
	return 0
}

func periodsPerYear(interval string) float64 {
	switch strings.ToLower(interval) {
	case "1m":
		return 365 * 24 * 60
	case "5m":
		return 365 * 24 * 12
	case "15m":
		return 365 * 24 * 4
	case "1h":
		return 365 * 24
	case "4h":
		return 365 * 6
	case "1d":
		return 252
	case "1w":
		return 52
	default:
		return 252
	}
}

func maxDrawdown(curve []EquityPoint) float64 {
	peak := 0.0
	maxDD := 0.0
	for _, point := range curve {
		if point.Equity > peak {
			peak = point.Equity
		}
		if peak == 0 {
			continue
		}
		drawdown := (peak - point.Equity) / peak
		if drawdown > maxDD {
			maxDD = drawdown
		}
	}
	return maxDD
}

func average(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, value := range values {
		sum += value
	}
	return sum / float64(len(values))
}

func stddev(values []float64, mean float64) float64 {
	if len(values) < 2 {
		return 0
	}
	sum := 0.0
	for _, value := range values {
		delta := value - mean
		sum += delta * delta
	}
	return math.Sqrt(sum / float64(len(values)-1))
}
