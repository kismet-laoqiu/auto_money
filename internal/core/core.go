package core

import (
	"math"
	"strings"
	"time"

	"quantlab/internal/config"
)

type Side string

const (
	Flat  Side = "flat"
	Long  Side = "long"
	Short Side = "short"
)

type Bar struct {
	Time   time.Time `json:"time"`
	Open   float64   `json:"open"`
	High   float64   `json:"high"`
	Low    float64   `json:"low"`
	Close  float64   `json:"close"`
	Volume float64   `json:"volume"`
}

type Dataset struct {
	Name     string `json:"name"`
	Provider string `json:"provider"`
	Symbol   string `json:"symbol"`
	Interval string `json:"interval"`
	Bars     []Bar  `json:"bars"`
}

type Signal struct {
	Side    Side     `json:"side"`
	Score   float64  `json:"score"`
	Entry   float64  `json:"entry"`
	Stop    float64  `json:"stop"`
	Target  float64  `json:"target"`
	Reasons []string `json:"reasons"`
}

type Trade struct {
	Symbol     string    `json:"symbol"`
	Side       Side      `json:"side"`
	EntryTime  time.Time `json:"entry_time"`
	ExitTime   time.Time `json:"exit_time"`
	EntryPrice float64   `json:"entry_price"`
	ExitPrice  float64   `json:"exit_price"`
	Stop       float64   `json:"stop"`
	Target     float64   `json:"target"`
	Return     float64   `json:"return"`
	BarsHeld   int       `json:"bars_held"`
	Reason     string    `json:"reason"`
	Score      float64   `json:"score"`
}

type EquityPoint struct {
	Time   time.Time `json:"time"`
	Equity float64   `json:"equity"`
}

type Stats struct {
	TotalReturn   float64 `json:"total_return"`
	CAGR          float64 `json:"cagr"`
	Sharpe        float64 `json:"sharpe"`
	Calmar        float64 `json:"calmar"`
	MaxDrawdown   float64 `json:"max_drawdown"`
	WinRate       float64 `json:"win_rate"`
	ProfitFactor  float64 `json:"profit_factor"`
	Trades        int     `json:"trades"`
	AverageReturn float64 `json:"average_return"`
}

type Report struct {
	Name           string        `json:"name"`
	Provider       string        `json:"provider"`
	Symbol         string        `json:"symbol"`
	Interval       string        `json:"interval"`
	Bars           int           `json:"bars"`
	InSample       Stats         `json:"in_sample"`
	OutOfSample    Stats         `json:"out_of_sample"`
	ObjectiveScore float64       `json:"objective_score"`
	SignalCount    int           `json:"signal_count"`
	Trades         []Trade       `json:"trades"`
	EquityCurve    []EquityPoint `json:"equity_curve"`
}

type Aggregate struct {
	ObjectiveScore float64            `json:"objective_score"`
	Metrics        map[string]float64 `json:"metrics"`
}

type Pivot struct {
	Index int
	Price float64
	Kind  string
}

type Swing struct {
	Low  float64
	High float64
}

func WarmupBars(cfg config.StrategyConfig) int {
	warmup := cfg.SlowSMA
	if cfg.ATRWindow > warmup {
		warmup = cfg.ATRWindow
	}
	if cfg.LevelLookback > warmup {
		warmup = cfg.LevelLookback
	}
	return warmup + cfg.PivotWindow*2 + 2
}

func EvaluateSignal(bars []Bar, idx int, cfg config.StrategyConfig) Signal {
	if idx < WarmupBars(cfg) || idx <= 0 {
		return Signal{Side: Flat}
	}

	fast, okFast := smaAt(bars, idx, cfg.FastSMA)
	slow, okSlow := smaAt(bars, idx, cfg.SlowSMA)
	atr := atrAt(bars, idx, cfg.ATRWindow)
	if !okFast || !okSlow || atr <= 0 {
		return Signal{Side: Flat}
	}

	start := idx - cfg.LevelLookback - cfg.PivotWindow*2
	if start < 0 {
		start = 0
	}
	recent := bars[start : idx+1]
	pivots := findPivots(recent, cfg.PivotWindow)
	if len(pivots) < 4 {
		return Signal{Side: Flat}
	}

	closePrice := bars[idx].Close
	support, resistance := findNearestLevels(closePrice, pivots)
	upSwing, hasUp, downSwing, hasDown := findRecentSwings(pivots)
	waveLong, waveShort := waveBias(pivots)
	features := ExtractFeatureSet(bars, idx, cfg)

	longScore := 0.0
	shortScore := 0.0
	longReasons := make([]string, 0, 8)
	shortReasons := make([]string, 0, 8)

	if fast > slow {
		longScore += 1.25
		longReasons = append(longReasons, "SMA trend up")
	}
	if fast < slow {
		shortScore += 1.25
		shortReasons = append(shortReasons, "SMA trend down")
	}
	if !math.IsNaN(support) && support < closePrice && nearLevel(closePrice, support, cfg.LevelTolerance) {
		longScore += 1.0
		longReasons = append(longReasons, "support reaction")
	}
	if !math.IsNaN(resistance) && resistance > closePrice && nearLevel(closePrice, resistance, cfg.LevelTolerance) {
		shortScore += 1.0
		shortReasons = append(shortReasons, "resistance reaction")
	}
	if hasUp && nearAny(closePrice, retracementLevels(upSwing.High, upSwing.Low, true), cfg.FibTolerance) {
		longScore += 1.0
		longReasons = append(longReasons, "Fib pullback zone")
	}
	if hasDown && nearAny(closePrice, retracementLevels(downSwing.High, downSwing.Low, false), cfg.FibTolerance) {
		shortScore += 1.0
		shortReasons = append(shortReasons, "Fib pullback zone")
	}
	if bullishPriceAction(bars[idx-1], bars[idx]) {
		longScore += 1.25
		longReasons = append(longReasons, "bullish price action")
	}
	if bearishPriceAction(bars[idx-1], bars[idx]) {
		shortScore += 1.25
		shortReasons = append(shortReasons, "bearish price action")
	}
	if waveLong {
		longScore += 1.0
		longReasons = append(longReasons, "impulse-like higher highs/lows")
	}
	if waveShort {
		shortScore += 1.0
		shortReasons = append(shortReasons, "impulse-like lower highs/lows")
	}

	if features.Trigger.BreakRetestFlag {
		if features.Trigger.CloseLocationValue >= 0.55 {
			longScore += 0.15
			longReasons = append(longReasons, "break retest")
		}
		if features.Trigger.CloseLocationValue <= 0.45 {
			shortScore += 0.15
			shortReasons = append(shortReasons, "break retest")
		}
	}
	if features.Volume.VolumeAvailable {
		if features.Volume.BreakoutVolumeConfirmed {
			if longScore >= shortScore {
				longScore += 0.25
				longReasons = append(longReasons, "volume confirmed")
			} else {
				shortScore += 0.25
				shortReasons = append(shortReasons, "volume confirmed")
			}
		}
		if features.Volume.PullbackVolumeDryupFlag {
			if features.Regime.TrendUpFlag || features.Fib.ActiveSwingDirection == "up" {
				longScore += 0.25
				longReasons = append(longReasons, "pullback volume dry-up")
			}
			if features.Regime.TrendDownFlag || features.Fib.ActiveSwingDirection == "down" {
				shortScore += 0.25
				shortReasons = append(shortReasons, "pullback volume dry-up")
			}
		}
	}
	if features.Regime.HighVolFlag && !features.Volume.BreakoutVolumeConfirmed {
		longScore -= 0.10
		shortScore -= 0.10
	}
	if features.Regime.CompressionFlag && !features.Trigger.BreakRetestFlag {
		longScore -= 0.05
		shortScore -= 0.05
	}

	threshold := cfg.SignalThreshold
	if longScore < threshold && shortScore < threshold {
		return Signal{Side: Flat}
	}
	if longScore == shortScore {
		return Signal{Side: Flat}
	}

	if longScore > shortScore {
		risk := math.Max(atr*cfg.StopATR, atr*0.8)
		if !math.IsNaN(support) && support < closePrice {
			risk = math.Max(risk, closePrice-support)
		}
		return Signal{Side: Long, Score: longScore, Entry: closePrice, Stop: closePrice - risk, Target: closePrice + risk*cfg.RewardRisk, Reasons: longReasons}
	}

	risk := math.Max(atr*cfg.StopATR, atr*0.8)
	if !math.IsNaN(resistance) && resistance > closePrice {
		risk = math.Max(risk, resistance-closePrice)
	}
	return Signal{Side: Short, Score: shortScore, Entry: closePrice, Stop: closePrice + risk, Target: closePrice - risk*cfg.RewardRisk, Reasons: shortReasons}
}

func findPivots(bars []Bar, window int) []Pivot {
	if len(bars) < window*2+1 {
		return nil
	}
	pivots := make([]Pivot, 0, len(bars)/(window+1))
	for i := window; i < len(bars)-window; i++ {
		isHigh := true
		isLow := true
		for j := i - window; j <= i+window; j++ {
			if j == i {
				continue
			}
			if bars[j].High >= bars[i].High {
				isHigh = false
			}
			if bars[j].Low <= bars[i].Low {
				isLow = false
			}
			if !isHigh && !isLow {
				break
			}
		}
		if isHigh {
			pivots = append(pivots, Pivot{Index: i, Price: bars[i].High, Kind: "high"})
		}
		if isLow {
			pivots = append(pivots, Pivot{Index: i, Price: bars[i].Low, Kind: "low"})
		}
	}
	return pivots
}

func findNearestLevels(price float64, pivots []Pivot) (float64, float64) {
	support := math.NaN()
	resistance := math.NaN()
	for _, pivot := range pivots {
		switch pivot.Kind {
		case "low":
			if pivot.Price <= price && (math.IsNaN(support) || pivot.Price > support) {
				support = pivot.Price
			}
		case "high":
			if pivot.Price >= price && (math.IsNaN(resistance) || pivot.Price < resistance) {
				resistance = pivot.Price
			}
		}
	}
	return support, resistance
}

func findRecentSwings(pivots []Pivot) (Swing, bool, Swing, bool) {
	var up Swing
	var down Swing
	var hasUp bool
	var hasDown bool
	for i := len(pivots) - 1; i >= 1; i-- {
		prev := pivots[i-1]
		curr := pivots[i]
		if !hasUp && prev.Kind == "low" && curr.Kind == "high" && curr.Price > prev.Price {
			up = Swing{Low: prev.Price, High: curr.Price}
			hasUp = true
		}
		if !hasDown && prev.Kind == "high" && curr.Kind == "low" && curr.Price < prev.Price {
			down = Swing{Low: curr.Price, High: prev.Price}
			hasDown = true
		}
		if hasUp && hasDown {
			break
		}
	}
	return up, hasUp, down, hasDown
}

func waveBias(pivots []Pivot) (bool, bool) {
	highs := make([]float64, 0, 2)
	lows := make([]float64, 0, 2)
	for i := len(pivots) - 1; i >= 0 && (len(highs) < 2 || len(lows) < 2); i-- {
		pivot := pivots[i]
		if pivot.Kind == "high" && len(highs) < 2 {
			highs = append(highs, pivot.Price)
		}
		if pivot.Kind == "low" && len(lows) < 2 {
			lows = append(lows, pivot.Price)
		}
	}
	if len(highs) < 2 || len(lows) < 2 {
		return false, false
	}
	latestHigh := highs[0]
	priorHigh := highs[1]
	latestLow := lows[0]
	priorLow := lows[1]
	return latestHigh > priorHigh && latestLow > priorLow, latestHigh < priorHigh && latestLow < priorLow
}

func retracementLevels(high, low float64, fromUpSwing bool) []float64 {
	rangeSize := high - low
	if rangeSize <= 0 {
		return nil
	}
	ratios := []float64{0.382, 0.5, 0.618}
	levels := make([]float64, 0, len(ratios))
	for _, ratio := range ratios {
		if fromUpSwing {
			levels = append(levels, high-rangeSize*ratio)
			continue
		}
		levels = append(levels, low+rangeSize*ratio)
	}
	return levels
}

func bullishPriceAction(prev, curr Bar) bool {
	return bullishEngulfing(prev, curr) || bullishPin(curr)
}

func bearishPriceAction(prev, curr Bar) bool {
	return bearishEngulfing(prev, curr) || bearishPin(curr)
}

func bullishEngulfing(prev, curr Bar) bool {
	if prev.Close >= prev.Open || curr.Close <= curr.Open {
		return false
	}
	return curr.Open <= prev.Close && curr.Close >= prev.Open
}

func bearishEngulfing(prev, curr Bar) bool {
	if prev.Close <= prev.Open || curr.Close >= curr.Open {
		return false
	}
	return curr.Open >= prev.Close && curr.Close <= prev.Open
}

func bullishPin(bar Bar) bool {
	body := math.Abs(bar.Close - bar.Open)
	lowerWick := math.Min(bar.Close, bar.Open) - bar.Low
	upperWick := bar.High - math.Max(bar.Close, bar.Open)
	return lowerWick > body*2 && upperWick <= math.Max(body, lowerWick*0.4)
}

func bearishPin(bar Bar) bool {
	body := math.Abs(bar.Close - bar.Open)
	upperWick := bar.High - math.Max(bar.Close, bar.Open)
	lowerWick := math.Min(bar.Close, bar.Open) - bar.Low
	return upperWick > body*2 && lowerWick <= math.Max(body, upperWick*0.4)
}

func nearAny(price float64, levels []float64, tolerance float64) bool {
	for _, level := range levels {
		if nearLevel(price, level, tolerance) {
			return true
		}
	}
	return false
}

func nearLevel(price, level, tolerance float64) bool {
	if price == 0 || level == 0 {
		return false
	}
	return math.Abs(price-level)/price <= tolerance
}

func smaAt(bars []Bar, idx, window int) (float64, bool) {
	if window <= 0 || idx+1 < window {
		return 0, false
	}
	sum := 0.0
	for i := idx - window + 1; i <= idx; i++ {
		sum += bars[i].Close
	}
	return sum / float64(window), true
}

func atrAt(bars []Bar, idx, window int) float64 {
	if window <= 0 || idx < window {
		return 0
	}
	sum := 0.0
	for i := idx - window + 1; i <= idx; i++ {
		prevClose := bars[i-1].Close
		tr := math.Max(bars[i].High-bars[i].Low, math.Max(math.Abs(bars[i].High-prevClose), math.Abs(bars[i].Low-prevClose)))
		sum += tr
	}
	return sum / float64(window)
}

func JoinReasons(reasons []string) string {
	return strings.Join(reasons, ", ")
}
