package insights

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"quantlab/internal/config"
	"quantlab/internal/core"
	"quantlab/internal/watchlist"
)

type PriceReader interface {
	FetchTickerPrice(ctx context.Context, symbol, productType string) (float64, error)
}

type EvaluateSignalFunc func([]core.Bar, int, config.StrategyConfig) core.Signal
type ExtractFeaturesFunc func([]core.Bar, int, config.StrategyConfig) core.FeatureSet

type Config struct {
	WatchlistPath   string
	Provider        string
	ProductType     string
	Strategy        config.StrategyConfig
	Insights        config.InsightsConfig
	Store           WarehouseStore
	PriceReader     PriceReader
	LoadWatchlist   func(string) (watchlist.File, error)
	EvaluateSignal  EvaluateSignalFunc
	ExtractFeatures ExtractFeaturesFunc
	Now             func() time.Time
}

type Service struct {
	cfg Config

	mu            sync.Mutex
	threshold     map[string]cachedThreshold
	marketFetcher marketContextFetcher
	marketCache   cachedMarketContext
}

type cachedThreshold struct {
	At         time.Time
	Thresholds AnomalyThresholds
}

type cachedMarketContext struct {
	At       time.Time
	Snapshot *MarketContextSnapshot
}

func NewService(cfg Config) *Service {
	if cfg.LoadWatchlist == nil {
		cfg.LoadWatchlist = watchlist.Load
	}
	if cfg.EvaluateSignal == nil {
		cfg.EvaluateSignal = core.EvaluateSignal
	}
	if cfg.ExtractFeatures == nil {
		cfg.ExtractFeatures = core.ExtractFeatureSet
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	return &Service{
		cfg:           cfg,
		threshold:     map[string]cachedThreshold{},
		marketFetcher: newMarketContextHTTPFetcher(cfg.Now),
	}
}

func (service *Service) DetectAlerts(ctx context.Context) ([]Event, error) {
	file, err := service.loadWatchlist()
	if err != nil {
		return nil, err
	}
	alerts := make([]Event, 0, len(file.Symbols))
	for _, item := range file.Symbols {
		snapshot, symbolAlerts, err := service.evaluateSymbol(ctx, file, strings.TrimSpace(item.Symbol))
		if err != nil {
			return nil, err
		}
		_ = snapshot
		alerts = append(alerts, symbolAlerts...)
	}
	sort.Slice(alerts, func(i, j int) bool {
		if alerts[i].Ts.Equal(alerts[j].Ts) {
			return alerts[i].SymbolValue < alerts[j].SymbolValue
		}
		return alerts[i].Ts.Before(alerts[j].Ts)
	})
	return alerts, nil
}

func (service *Service) BuildDashboard(ctx context.Context) (DashboardReport, error) {
	file, err := service.loadWatchlist()
	if err != nil {
		return DashboardReport{}, err
	}
	report := DashboardReport{
		GeneratedAt:   service.cfg.Now().UTC(),
		Watchlist:     service.cfg.WatchlistPath,
		Symbols:       make([]SymbolSnapshot, 0, len(file.Symbols)),
		MarketContext: &MarketContextSnapshot{},
	}
	for _, item := range file.Symbols {
		snapshot, alerts, err := service.evaluateSymbol(ctx, file, strings.TrimSpace(item.Symbol))
		if err != nil {
			return DashboardReport{}, err
		}
		report.Symbols = append(report.Symbols, snapshot)
		report.Alerts = append(report.Alerts, alerts...)
	}
	sort.Slice(report.Symbols, func(i, j int) bool { return report.Symbols[i].Symbol < report.Symbols[j].Symbol })
	sort.Slice(report.Alerts, func(i, j int) bool {
		if report.Alerts[i].Ts.Equal(report.Alerts[j].Ts) {
			return report.Alerts[i].SymbolValue < report.Alerts[j].SymbolValue
		}
		return report.Alerts[i].Ts.After(report.Alerts[j].Ts)
	})
	report.MarketContext = service.buildMarketContext(ctx, file)
	return report, nil
}

func (service *Service) evaluateSymbol(ctx context.Context, file watchlist.File, symbol string) (SymbolSnapshot, []Event, error) {
	if symbol == "" {
		return SymbolSnapshot{}, nil, nil
	}
	dailyBars, err := service.cfg.Store.LoadRecentBars(ctx, file.Provider, symbol, service.cfg.Insights.DailySignal.Interval, service.cfg.Insights.DailySignal.LookbackBars)
	if err != nil {
		return SymbolSnapshot{}, nil, err
	}
	anomalyLimit := maxInt(service.cfg.Insights.Anomaly.RollingWindow+service.cfg.Insights.Anomaly.CooldownBars+2, 32)
	anomalyBars, err := service.cfg.Store.LoadRecentBars(ctx, file.Provider, symbol, service.cfg.Insights.Anomaly.Interval, anomalyLimit)
	if err != nil {
		return SymbolSnapshot{}, nil, err
	}

	daily := service.analyzeDailySignal(dailyBars)
	anomaly, err := service.analyzeAnomaly(ctx, file.Provider, symbol, anomalyBars)
	if err != nil {
		return SymbolSnapshot{}, nil, err
	}

	latestPrice := 0.0
	if service.cfg.PriceReader != nil {
		price, priceErr := service.cfg.PriceReader.FetchTickerPrice(ctx, symbol, file.ProductType)
		if priceErr == nil {
			latestPrice = price
		}
	}
	if latestPrice == 0 {
		if len(anomalyBars) > 0 {
			latestPrice = anomalyBars[len(anomalyBars)-1].Close
		} else if len(dailyBars) > 0 {
			latestPrice = dailyBars[len(dailyBars)-1].Close
		}
	}

	snapshot := SymbolSnapshot{
		Symbol:      symbol,
		LatestPrice: latestPrice,
	}
	if len(anomalyBars) > 0 {
		snapshot.Latest15mCloseAt = anomalyBars[len(anomalyBars)-1].Time.UTC()
	}
	if len(dailyBars) > 0 {
		snapshot.Latest1dCloseAt = dailyBars[len(dailyBars)-1].Time.UTC()
	}
	snapshot.DailyFeatures = featureSnapshot(daily.Features)

	alerts := make([]Event, 0, 2)
	if daily.Qualified && len(dailyBars) > 0 {
		signal := daily.Signal
		bar := dailyBars[len(dailyBars)-1]
		snapshot.DailySignal = &SignalSnapshot{
			Side:      string(signal.Side),
			Score:     signal.Score,
			Threshold: daily.Threshold,
			BarTime:   bar.Time.UTC(),
			Reasons:   append([]string(nil), signal.Reasons...),
		}
		title := fmt.Sprintf("%s 日线%s", symbol, mapSideLabel(signal.Side))
		event := Event{
			EventIDValue: fmt.Sprintf("insight:daily_signal:%s:%s:%d:%s", symbol, service.cfg.Insights.DailySignal.Interval, bar.Time.UTC().Unix(), signal.Side),
			SymbolValue:  symbol,
			Ts:           bar.Time.UTC(),
			AlertType:    AlertTypeDailySignal,
			Interval:     service.cfg.Insights.DailySignal.Interval,
			Title:        title,
			Summary:      fmt.Sprintf("score=%.2f threshold=%.2f close=%.4f", signal.Score, daily.Threshold, bar.Close),
			Details: []string{
				fmt.Sprintf("reasons=%s", strings.Join(signal.Reasons, ", ")),
				fmt.Sprintf("rsi14=%.2f volume_zscore=%.2f", daily.Features.Trigger.RSI14, daily.Features.Volume.VolumeZScore),
				fmt.Sprintf("relative_volume_ratio=%.2f breakout_confirmed=%t", daily.Features.Volume.RelativeVolumeRatio, daily.Features.Volume.BreakoutVolumeConfirmed),
			},
			Direction: string(signal.Side),
			Score:     signal.Score,
			Threshold: daily.Threshold,
		}
		alerts = append(alerts, event)
	}

	if anomaly.Qualified && len(anomalyBars) > 0 {
		bar := anomalyBars[len(anomalyBars)-1]
		snapshot.LatestMarketAlert = &AnomalySnapshot{
			Signal:         anomaly.Signal,
			MovePct:        anomaly.MovePct,
			MoveThreshold:  anomaly.MoveLimit,
			Volume:         anomaly.Volume,
			VolumeBaseline: anomaly.VolumeLimit,
			VolumeRatio:    anomaly.VolumeRatio,
			VolumeZScore:   anomaly.VolumeZScore,
			BarTime:        bar.Time.UTC(),
		}
		title, summary, details := marketAlertNotification(symbol, bar, anomaly, daily.Features)
		event := Event{
			EventIDValue: fmt.Sprintf("insight:market_alert:%s:%s:%d:%s", symbol, service.cfg.Insights.Anomaly.Interval, bar.Time.UTC().Unix(), anomaly.Signal),
			SymbolValue:  symbol,
			Ts:           bar.Time.UTC(),
			AlertType:    AlertTypeMarketAlert,
			Interval:     service.cfg.Insights.Anomaly.Interval,
			Title:        title,
			Summary:      summary,
			Details:      details,
			Signal:       anomaly.Signal,
			Threshold:    maxFloat(anomaly.MoveLimit, anomaly.VolumeLimit),
		}
		alerts = append(alerts, event)
	}

	return snapshot, alerts, nil
}

func (service *Service) analyzeDailySignal(bars []core.Bar) DailyAnalysis {
	if len(bars) == 0 {
		return DailyAnalysis{}
	}
	index := len(bars) - 1
	features := service.cfg.ExtractFeatures(bars, index, service.cfg.Strategy)
	signal := service.cfg.EvaluateSignal(bars, index, service.cfg.Strategy)
	if signal.Side == core.Flat {
		return DailyAnalysis{Features: features}
	}

	warmup := core.WarmupBars(service.cfg.Strategy)
	scores := make([]float64, 0, len(bars))
	for i := warmup; i < len(bars)-1; i++ {
		candidate := service.cfg.EvaluateSignal(bars[:i+1], i, service.cfg.Strategy)
		if candidate.Side == core.Flat {
			continue
		}
		scores = append(scores, candidate.Score)
	}
	threshold := maxFloat(service.cfg.Insights.DailySignal.MinScore, quantile(scores, service.cfg.Insights.DailySignal.ScoreQuantile))
	qualified := signal.Score >= threshold
	if qualified && service.hasRecentDailySignal(bars, threshold, signal.Side) {
		qualified = false
	}
	return DailyAnalysis{
		Signal:    signal,
		Threshold: threshold,
		Features:  features,
		Qualified: qualified,
	}
}

func (service *Service) hasRecentDailySignal(bars []core.Bar, threshold float64, side core.Side) bool {
	warmup := core.WarmupBars(service.cfg.Strategy)
	start := len(bars) - 1 - service.cfg.Insights.DailySignal.CooldownBars
	if start < warmup {
		start = warmup
	}
	for i := start; i < len(bars)-1; i++ {
		candidate := service.cfg.EvaluateSignal(bars[:i+1], i, service.cfg.Strategy)
		if candidate.Side == side && candidate.Score >= threshold {
			return true
		}
	}
	return false
}

func (service *Service) analyzeAnomaly(ctx context.Context, provider, symbol string, bars []core.Bar) (AnomalyAnalysis, error) {
	if len(bars) < 2 {
		return AnomalyAnalysis{}, nil
	}
	thresholds, err := service.loadAnomalyThresholds(ctx, provider, symbol)
	if err != nil {
		return AnomalyAnalysis{}, err
	}
	last := len(bars) - 1
	current := bars[last]
	movePct := math.Abs((current.Close/current.Open - 1) * 100)
	moveLimit := maxFloat(service.cfg.Insights.Anomaly.MinMovePct, thresholds.MovePct)

	window := service.cfg.Insights.Anomaly.RollingWindow
	if window > last {
		window = last
	}
	recent := bars[last-window : last]
	mean := averageVolume(recent)
	std := stddevVolume(recent, mean)
	volumeRatio := 0.0
	if mean > 0 {
		volumeRatio = current.Volume / mean
	}
	volumeZScore := 0.0
	if std > 0 {
		volumeZScore = (current.Volume - mean) / std
	} else if mean > 0 && current.Volume > mean {
		volumeZScore = math.Inf(1)
	}
	volumeQualified := current.Volume >= thresholds.Volume &&
		volumeRatio >= service.cfg.Insights.Anomaly.MinVolumeRatio &&
		volumeZScore >= service.cfg.Insights.Anomaly.MinVolumeZScore

	moveQualified := movePct >= moveLimit
	if moveQualified && service.hasRecentMoveAlert(bars, moveLimit) {
		moveQualified = false
	}
	if volumeQualified && service.hasRecentVolumeAlert(bars, thresholds.Volume) {
		volumeQualified = false
	}

	signal := MarketAlertSignal("")
	switch {
	case moveQualified && volumeQualified:
		signal = MarketAlertCombo
	case current.Close > current.Open && moveQualified:
		signal = MarketAlertSurge
	case current.Close < current.Open && moveQualified:
		signal = MarketAlertDump
	case volumeQualified:
		signal = MarketAlertVolumeSpike
	}
	return AnomalyAnalysis{
		Signal:       signal,
		MovePct:      movePct,
		MoveLimit:    moveLimit,
		Volume:       current.Volume,
		VolumeLimit:  thresholds.Volume,
		VolumeRatio:  volumeRatio,
		VolumeZScore: volumeZScore,
		Qualified:    signal != "",
	}, nil
}

func (service *Service) hasRecentMoveAlert(bars []core.Bar, limit float64) bool {
	start := len(bars) - 1 - service.cfg.Insights.Anomaly.CooldownBars
	if start < 0 {
		start = 0
	}
	for i := start; i < len(bars)-1; i++ {
		movePct := math.Abs((bars[i].Close/bars[i].Open - 1) * 100)
		if movePct >= limit {
			return true
		}
	}
	return false
}

func (service *Service) hasRecentVolumeAlert(bars []core.Bar, limit float64) bool {
	start := len(bars) - 1 - service.cfg.Insights.Anomaly.CooldownBars
	if start < 1 {
		start = 1
	}
	window := service.cfg.Insights.Anomaly.RollingWindow
	for i := start; i < len(bars)-1; i++ {
		left := i - window
		if left < 0 {
			left = 0
		}
		recent := bars[left:i]
		mean := averageVolume(recent)
		std := stddevVolume(recent, mean)
		ratio := 0.0
		if mean > 0 {
			ratio = bars[i].Volume / mean
		}
		zscore := 0.0
		if std > 0 {
			zscore = (bars[i].Volume - mean) / std
		} else if mean > 0 && bars[i].Volume > mean {
			zscore = math.Inf(1)
		}
		if bars[i].Volume >= limit &&
			ratio >= service.cfg.Insights.Anomaly.MinVolumeRatio &&
			zscore >= service.cfg.Insights.Anomaly.MinVolumeZScore {
			return true
		}
	}
	return false
}

func (service *Service) loadAnomalyThresholds(ctx context.Context, provider, symbol string) (AnomalyThresholds, error) {
	key := provider + ":" + symbol + ":" + service.cfg.Insights.Anomaly.Interval
	now := service.cfg.Now().UTC()
	service.mu.Lock()
	cached, ok := service.threshold[key]
	if ok && now.Sub(cached.At) <= 6*time.Hour {
		service.mu.Unlock()
		return cached.Thresholds, nil
	}
	service.mu.Unlock()

	thresholds, err := service.cfg.Store.LoadAnomalyThresholds(
		ctx,
		provider,
		symbol,
		service.cfg.Insights.Anomaly.Interval,
		service.cfg.Insights.Anomaly.LookbackBars,
		service.cfg.Insights.Anomaly.MoveQuantile,
		service.cfg.Insights.Anomaly.VolumeQuantile,
	)
	if err != nil {
		return AnomalyThresholds{}, err
	}
	service.mu.Lock()
	service.threshold[key] = cachedThreshold{At: now, Thresholds: thresholds}
	service.mu.Unlock()
	return thresholds, nil
}

func (service *Service) loadWatchlist() (watchlist.File, error) {
	if service.cfg.LoadWatchlist == nil {
		return watchlist.File{}, fmt.Errorf("watchlist loader is nil")
	}
	if strings.TrimSpace(service.cfg.WatchlistPath) == "" {
		return watchlist.File{}, fmt.Errorf("watchlist path is empty")
	}
	return service.cfg.LoadWatchlist(service.cfg.WatchlistPath)
}

func featureSnapshot(features core.FeatureSet) FeatureSnapshot {
	return FeatureSnapshot{
		RSI14:                   features.Trigger.RSI14,
		NeedleDropPct:           features.Trigger.NeedleDropPct,
		ReclaimPct:              features.Trigger.ReclaimPct,
		VolumeZScore:            features.Volume.VolumeZScore,
		RelativeVolumeRatio:     features.Volume.RelativeVolumeRatio,
		BreakoutVolumeConfirmed: features.Volume.BreakoutVolumeConfirmed,
		TrendUp:                 features.Regime.TrendUpFlag,
		TrendDown:               features.Regime.TrendDownFlag,
		Range:                   features.Regime.RangeFlag,
		HighVol:                 features.Regime.HighVolFlag,
	}
}

func marketAlertNotification(symbol string, bar core.Bar, anomaly AnomalyAnalysis, features core.FeatureSet) (string, string, []string) {
	title := marketAlertTitle(symbol, bar, anomaly)
	summary := marketAlertSummary(bar, anomaly)
	details := []string{
		fmt.Sprintf("成交量 %.4f（阈值 %.4f）", anomaly.Volume, anomaly.VolumeLimit),
		fmt.Sprintf("日线背景：%s趋势，RSI14 %.2f", marketAlertTrendLabel(features), features.Trigger.RSI14),
	}
	return title, summary, details
}

func marketAlertTitle(symbol string, bar core.Bar, anomaly AnomalyAnalysis) string {
	switch anomaly.Signal {
	case MarketAlertCombo:
		direction := "放量上涨"
		if bar.Close < bar.Open {
			direction = "放量下跌"
		}
		return fmt.Sprintf("%s 15m %s", symbol, direction)
	case MarketAlertSurge:
		return fmt.Sprintf("%s 15m 暴涨", symbol)
	case MarketAlertDump:
		return fmt.Sprintf("%s 15m 暴跌", symbol)
	default:
		return fmt.Sprintf("%s 15m 异常放量", symbol)
	}
}

func marketAlertSummary(bar core.Bar, anomaly AnomalyAnalysis) string {
	switch anomaly.Signal {
	case MarketAlertCombo:
		return fmt.Sprintf(
			"现价 %.4f，15m %s %.2f%%（阈值 %.2f%%），成交量放大到基线 %.2f 倍",
			bar.Close,
			marketAlertMoveLabel(bar),
			anomaly.MovePct,
			anomaly.MoveLimit,
			anomaly.VolumeRatio,
		)
	case MarketAlertSurge, MarketAlertDump:
		return fmt.Sprintf(
			"现价 %.4f，15m %s %.2f%%（阈值 %.2f%%）",
			bar.Close,
			marketAlertMoveLabel(bar),
			anomaly.MovePct,
			anomaly.MoveLimit,
		)
	default:
		return fmt.Sprintf(
			"现价 %.4f，15m 波动 %.2f%%，成交量放大到基线 %.2f 倍",
			bar.Close,
			anomaly.MovePct,
			anomaly.VolumeRatio,
		)
	}
}

func marketAlertMoveLabel(bar core.Bar) string {
	if bar.Close < bar.Open {
		return "跌幅"
	}
	return "涨幅"
}

func marketAlertTrendLabel(features core.FeatureSet) string {
	switch {
	case features.Regime.TrendUpFlag:
		return "上涨"
	case features.Regime.TrendDownFlag:
		return "下跌"
	case features.Regime.RangeFlag:
		return "震荡"
	default:
		return "中性"
	}
}

func mapSideLabel(side core.Side) string {
	switch side {
	case core.Long:
		return "买点"
	case core.Short:
		return "卖点"
	default:
		return "信号"
	}
}

func quantile(values []float64, q float64) float64 {
	if len(values) == 0 {
		return 0
	}
	clamped := q
	if clamped < 0 {
		clamped = 0
	}
	if clamped > 1 {
		clamped = 1
	}
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	if len(sorted) == 1 {
		return sorted[0]
	}
	position := clamped * float64(len(sorted)-1)
	lower := int(math.Floor(position))
	upper := int(math.Ceil(position))
	if lower == upper {
		return sorted[lower]
	}
	weight := position - float64(lower)
	return sorted[lower] + (sorted[upper]-sorted[lower])*weight
}

func averageVolume(bars []core.Bar) float64 {
	if len(bars) == 0 {
		return 0
	}
	total := 0.0
	for _, bar := range bars {
		total += bar.Volume
	}
	return total / float64(len(bars))
}

func stddevVolume(bars []core.Bar, mean float64) float64 {
	if len(bars) == 0 {
		return 0
	}
	sum := 0.0
	for _, bar := range bars {
		delta := bar.Volume - mean
		sum += delta * delta
	}
	return math.Sqrt(sum / float64(len(bars)))
}

func maxFloat(left, right float64) float64 {
	if left > right {
		return left
	}
	return right
}

func maxInt(left, right int) int {
	if left > right {
		return left
	}
	return right
}
