package insights

import (
	"context"
	"testing"
	"time"

	"quantlab/internal/config"
	"quantlab/internal/core"
	"quantlab/internal/watchlist"
)

func TestServiceDetectAlertsTriggersDailySignalAboveHistoryThreshold(t *testing.T) {
	store := &storeStub{
		bars: map[string][]core.Bar{
			"bitget:BTCUSDT:1d":  buildBars(8, time.Unix(1710000000, 0).UTC(), time.Hour*24, 100, 10),
			"bitget:BTCUSDT:15m": buildBars(32, time.Unix(1710000000, 0).UTC(), 15*time.Minute, 100, 10),
		},
		thresholds: map[string]AnomalyThresholds{
			"bitget:BTCUSDT:15m": {MovePct: 50, Volume: 500},
		},
	}
	signals := map[int]core.Signal{
		3: {Side: core.Long, Score: 3.8},
		4: {Side: core.Long, Score: 4.1},
		5: {Side: core.Long, Score: 4.3},
		6: {Side: core.Long, Score: 4.5},
		7: {Side: core.Long, Score: 4.9, Reasons: []string{"trend up", "break retest"}},
	}
	service := NewService(Config{
		WatchlistPath: "configs/platform/watchlist.yaml",
		Strategy:      config.StrategyConfig{FastSMA: 2, SlowSMA: 3, ATRWindow: 2, LevelLookback: 2, PivotWindow: 1},
		Insights: config.InsightsConfig{
			DailySignal: config.DailySignalAlertConfig{Interval: "1d", LookbackBars: 8, MinScore: 4.25, ScoreQuantile: 0.75, CooldownBars: 2},
			Anomaly:     config.AnomalyAlertConfig{Interval: "15m", RollingWindow: 5, CooldownBars: 4, MinMovePct: 20, MinVolumeRatio: 5, MinVolumeZScore: 10},
		},
		Store: store,
		LoadWatchlist: func(string) (watchlist.File, error) {
			return watchlist.File{Provider: "bitget", ProductType: "USDT-FUTURES", Symbols: []config.LiveSymbolConfig{{Symbol: "BTCUSDT"}}}, nil
		},
		EvaluateSignal: func(_ []core.Bar, idx int, _ config.StrategyConfig) core.Signal {
			if signal, ok := signals[idx]; ok {
				return signal
			}
			return core.Signal{Side: core.Flat}
		},
		ExtractFeatures: func(_ []core.Bar, _ int, _ config.StrategyConfig) core.FeatureSet {
			return core.FeatureSet{Trigger: core.PriceActionTriggerFeatures{RSI14: 26}, Volume: core.VolumeConfirmationFeatures{VolumeZScore: 2.1, RelativeVolumeRatio: 1.4}}
		},
		Now: func() time.Time { return time.Unix(1710000000, 0).UTC() },
	})

	alerts, err := service.DetectAlerts(context.Background())
	if err != nil {
		t.Fatalf("detect alerts: %v", err)
	}
	if len(alerts) != 1 {
		t.Fatalf("expected one alert, got %+v", alerts)
	}
	if alerts[0].AlertType != AlertTypeDailySignal || alerts[0].Direction != "long" {
		t.Fatalf("unexpected alert: %+v", alerts[0])
	}
	if alerts[0].Threshold < 4.25 {
		t.Fatalf("unexpected threshold: %+v", alerts[0])
	}
}

func TestServiceDetectAlertsTriggersVolumeSpike(t *testing.T) {
	bars := buildBars(30, time.Unix(1710000000, 0).UTC(), 15*time.Minute, 100, 10)
	bars[len(bars)-1].Volume = 160
	store := &storeStub{
		bars: map[string][]core.Bar{
			"bitget:BTCUSDT:1d":  buildBars(16, time.Unix(1710000000, 0).UTC(), time.Hour*24, 100, 10),
			"bitget:BTCUSDT:15m": bars,
		},
		thresholds: map[string]AnomalyThresholds{
			"bitget:BTCUSDT:15m": {MovePct: 50, Volume: 100},
		},
	}
	service := NewService(Config{
		WatchlistPath: "configs/platform/watchlist.yaml",
		Strategy:      config.StrategyConfig{FastSMA: 2, SlowSMA: 3, ATRWindow: 2, LevelLookback: 2, PivotWindow: 1},
		Insights: config.InsightsConfig{
			DailySignal: config.DailySignalAlertConfig{Interval: "1d", LookbackBars: 16, MinScore: 100, ScoreQuantile: 0.99, CooldownBars: 2},
			Anomaly:     config.AnomalyAlertConfig{Interval: "15m", RollingWindow: 20, CooldownBars: 4, MinMovePct: 20, MinVolumeRatio: 3, MinVolumeZScore: 3.5},
		},
		Store: store,
		LoadWatchlist: func(string) (watchlist.File, error) {
			return watchlist.File{Provider: "bitget", ProductType: "USDT-FUTURES", Symbols: []config.LiveSymbolConfig{{Symbol: "BTCUSDT"}}}, nil
		},
		EvaluateSignal: func(_ []core.Bar, _ int, _ config.StrategyConfig) core.Signal { return core.Signal{Side: core.Flat} },
		ExtractFeatures: func(_ []core.Bar, _ int, _ config.StrategyConfig) core.FeatureSet {
			return core.FeatureSet{
				Trigger: core.PriceActionTriggerFeatures{RSI14: 61.4},
				Regime:  core.RegimeTags{TrendUpFlag: true},
			}
		},
	})

	alerts, err := service.DetectAlerts(context.Background())
	if err != nil {
		t.Fatalf("detect alerts: %v", err)
	}
	if len(alerts) != 1 {
		t.Fatalf("expected one alert, got %+v", alerts)
	}
	if alerts[0].Signal != MarketAlertVolumeSpike {
		t.Fatalf("unexpected volume alert: %+v", alerts[0])
	}
	if alerts[0].Title != "BTCUSDT 15m 异常放量" {
		t.Fatalf("unexpected title: %q", alerts[0].Title)
	}
	if alerts[0].Summary != "现价 129.2000，15m 波动 0.16%，成交量放大到基线 16.00 倍" {
		t.Fatalf("unexpected summary: %q", alerts[0].Summary)
	}
	wantDetails := []string{
		"成交量 160.0000（阈值 100.0000）",
		"日线背景：上涨趋势，RSI14 61.40",
	}
	if len(alerts[0].Details) != len(wantDetails) {
		t.Fatalf("unexpected details: %+v", alerts[0].Details)
	}
	for i, want := range wantDetails {
		if alerts[0].Details[i] != want {
			t.Fatalf("unexpected detail[%d]: want %q, got %q", i, want, alerts[0].Details[i])
		}
	}
}

func TestServiceDetectAlertsTriggersSurgeReadableSummary(t *testing.T) {
	bars := buildBars(30, time.Unix(1710000000, 0).UTC(), 15*time.Minute, 100, 10)
	bars[len(bars)-1].Close = 132.87
	bars[len(bars)-1].High = 133.10
	store := &storeStub{
		bars: map[string][]core.Bar{
			"bitget:BTCUSDT:1d":  buildBars(16, time.Unix(1710000000, 0).UTC(), time.Hour*24, 100, 10),
			"bitget:BTCUSDT:15m": bars,
		},
		thresholds: map[string]AnomalyThresholds{
			"bitget:BTCUSDT:15m": {MovePct: 2.5, Volume: 1000},
		},
	}
	service := NewService(Config{
		WatchlistPath: "configs/platform/watchlist.yaml",
		Strategy:      config.StrategyConfig{FastSMA: 2, SlowSMA: 3, ATRWindow: 2, LevelLookback: 2, PivotWindow: 1},
		Insights: config.InsightsConfig{
			DailySignal: config.DailySignalAlertConfig{Interval: "1d", LookbackBars: 16, MinScore: 100, ScoreQuantile: 0.99, CooldownBars: 2},
			Anomaly:     config.AnomalyAlertConfig{Interval: "15m", RollingWindow: 20, CooldownBars: 4, MinMovePct: 2.5, MinVolumeRatio: 30, MinVolumeZScore: 30},
		},
		Store: store,
		LoadWatchlist: func(string) (watchlist.File, error) {
			return watchlist.File{Provider: "bitget", ProductType: "USDT-FUTURES", Symbols: []config.LiveSymbolConfig{{Symbol: "BTCUSDT"}}}, nil
		},
		EvaluateSignal: func(_ []core.Bar, _ int, _ config.StrategyConfig) core.Signal { return core.Signal{Side: core.Flat} },
		ExtractFeatures: func(_ []core.Bar, _ int, _ config.StrategyConfig) core.FeatureSet {
			return core.FeatureSet{
				Trigger: core.PriceActionTriggerFeatures{RSI14: 58.2},
				Regime:  core.RegimeTags{TrendUpFlag: true},
			}
		},
	})

	alerts, err := service.DetectAlerts(context.Background())
	if err != nil {
		t.Fatalf("detect alerts: %v", err)
	}
	if len(alerts) != 1 {
		t.Fatalf("expected one alert, got %+v", alerts)
	}
	if alerts[0].Signal != MarketAlertSurge {
		t.Fatalf("unexpected surge alert: %+v", alerts[0])
	}
	if alerts[0].Title != "BTCUSDT 15m 暴涨" {
		t.Fatalf("unexpected title: %q", alerts[0].Title)
	}
	if alerts[0].Summary != "现价 132.8700，15m 涨幅 3.00%（阈值 2.50%）" {
		t.Fatalf("unexpected summary: %q", alerts[0].Summary)
	}
	wantDetails := []string{
		"成交量 10.0000（阈值 1000.0000）",
		"日线背景：上涨趋势，RSI14 58.20",
	}
	if len(alerts[0].Details) != len(wantDetails) {
		t.Fatalf("unexpected details: %+v", alerts[0].Details)
	}
	for i, want := range wantDetails {
		if alerts[0].Details[i] != want {
			t.Fatalf("unexpected detail[%d]: want %q, got %q", i, want, alerts[0].Details[i])
		}
	}
}

func TestServiceBuildDashboardIncludesLatestPriceAndFeatures(t *testing.T) {
	store := &storeStub{
		bars: map[string][]core.Bar{
			"bitget:BTCUSDT:1d":  buildBars(10, time.Unix(1710000000, 0).UTC(), time.Hour*24, 100, 10),
			"bitget:BTCUSDT:15m": buildBars(32, time.Unix(1710000000, 0).UTC(), 15*time.Minute, 100, 10),
		},
		thresholds: map[string]AnomalyThresholds{
			"bitget:BTCUSDT:15m": {MovePct: 80, Volume: 1000},
		},
	}
	service := NewService(Config{
		WatchlistPath: "configs/platform/watchlist.yaml",
		Strategy:      config.StrategyConfig{FastSMA: 2, SlowSMA: 3, ATRWindow: 2, LevelLookback: 2, PivotWindow: 1},
		Insights: config.InsightsConfig{
			DailySignal: config.DailySignalAlertConfig{Interval: "1d", LookbackBars: 10, MinScore: 100, ScoreQuantile: 0.99, CooldownBars: 2},
			Anomaly:     config.AnomalyAlertConfig{Interval: "15m", RollingWindow: 20, CooldownBars: 4, MinMovePct: 20, MinVolumeRatio: 10, MinVolumeZScore: 10},
		},
		Store:       store,
		PriceReader: priceReaderStub{price: 12345.6},
		LoadWatchlist: func(string) (watchlist.File, error) {
			return watchlist.File{Provider: "bitget", ProductType: "USDT-FUTURES", Symbols: []config.LiveSymbolConfig{{Symbol: "BTCUSDT"}}}, nil
		},
		EvaluateSignal: func(_ []core.Bar, _ int, _ config.StrategyConfig) core.Signal { return core.Signal{Side: core.Flat} },
		ExtractFeatures: func(_ []core.Bar, _ int, _ config.StrategyConfig) core.FeatureSet {
			return core.FeatureSet{
				Trigger: core.PriceActionTriggerFeatures{RSI14: 31.5, NeedleDropPct: 0.8, ReclaimPct: 0.2},
				Volume:  core.VolumeConfirmationFeatures{VolumeZScore: 1.7, RelativeVolumeRatio: 1.2, BreakoutVolumeConfirmed: true},
				Regime:  core.RegimeTags{TrendUpFlag: true},
			}
		},
		Now: func() time.Time { return time.Unix(1710000000, 0).UTC() },
	})
	service.marketFetcher = marketContextFetcherStub{}

	report, err := service.BuildDashboard(context.Background())
	if err != nil {
		t.Fatalf("build dashboard: %v", err)
	}
	if len(report.Symbols) != 1 {
		t.Fatalf("unexpected report symbols: %+v", report)
	}
	if report.Symbols[0].LatestPrice != 12345.6 {
		t.Fatalf("unexpected latest price: %+v", report.Symbols[0])
	}
	if report.Symbols[0].DailyFeatures.RSI14 != 31.5 || !report.Symbols[0].DailyFeatures.BreakoutVolumeConfirmed {
		t.Fatalf("unexpected features: %+v", report.Symbols[0].DailyFeatures)
	}
}

func TestServiceBuildDashboardIncludesMarketContext(t *testing.T) {
	store := &storeStub{
		bars: map[string][]core.Bar{
			"bitget:BTCUSDT:1d":  buildBars(120, time.Unix(1710000000, 0).UTC(), time.Hour*24, 100, 10),
			"bitget:BTCUSDT:15m": buildBars(32, time.Unix(1710000000, 0).UTC(), 15*time.Minute, 100, 10),
			"bitget:ETHUSDT:1d":  buildBars(120, time.Unix(1710000000, 0).UTC(), time.Hour*24, 200, 10),
			"bitget:ETHUSDT:15m": buildBars(32, time.Unix(1710000000, 0).UTC(), 15*time.Minute, 200, 10),
		},
		thresholds: map[string]AnomalyThresholds{
			"bitget:BTCUSDT:15m": {MovePct: 80, Volume: 1000},
			"bitget:ETHUSDT:15m": {MovePct: 80, Volume: 1000},
		},
	}
	service := NewService(Config{
		WatchlistPath: "configs/platform/watchlist.yaml",
		Strategy:      config.StrategyConfig{FastSMA: 2, SlowSMA: 3, ATRWindow: 2, LevelLookback: 2, PivotWindow: 1},
		Insights: config.InsightsConfig{
			DailySignal: config.DailySignalAlertConfig{Interval: "1d", LookbackBars: 120, MinScore: 100, ScoreQuantile: 0.99, CooldownBars: 2},
			Anomaly:     config.AnomalyAlertConfig{Interval: "15m", RollingWindow: 20, CooldownBars: 4, MinMovePct: 20, MinVolumeRatio: 10, MinVolumeZScore: 10},
		},
		Store: store,
		LoadWatchlist: func(string) (watchlist.File, error) {
			return watchlist.File{
				Provider:    "bitget",
				ProductType: "USDT-FUTURES",
				Symbols: []config.LiveSymbolConfig{
					{Symbol: "BTCUSDT"},
					{Symbol: "ETHUSDT"},
				},
			}, nil
		},
		EvaluateSignal: func(_ []core.Bar, _ int, _ config.StrategyConfig) core.Signal { return core.Signal{Side: core.Flat} },
		ExtractFeatures: func(_ []core.Bar, _ int, _ config.StrategyConfig) core.FeatureSet {
			return core.FeatureSet{}
		},
		Now: func() time.Time { return time.Unix(1710000000, 0).UTC() },
	})
	service.marketFetcher = marketContextFetcherStub{
		snapshot: marketContextExternalSnapshot{
			WMA200: 250,
			FearGreed: &FearGreedSnapshot{
				Value:          11,
				Classification: "Extreme Fear",
			},
		},
	}

	report, err := service.BuildDashboard(context.Background())
	if err != nil {
		t.Fatalf("build dashboard: %v", err)
	}
	if report.MarketContext == nil {
		t.Fatalf("expected market context, got %+v", report)
	}
	if report.MarketContext.BTC == nil || report.MarketContext.BTC.WMA200 != 250 {
		t.Fatalf("expected btc market context, got %+v", report.MarketContext)
	}
	if report.MarketContext.FearGreed == nil || report.MarketContext.FearGreed.Value != 11 {
		t.Fatalf("expected fear and greed context, got %+v", report.MarketContext)
	}
}

type storeStub struct {
	bars       map[string][]core.Bar
	thresholds map[string]AnomalyThresholds
}

func (store *storeStub) LoadRecentBars(_ context.Context, provider, symbol, interval string, _ int) ([]core.Bar, error) {
	key := provider + ":" + symbol + ":" + interval
	return append([]core.Bar(nil), store.bars[key]...), nil
}

func (store *storeStub) LoadAnomalyThresholds(_ context.Context, provider, symbol, interval string, _ int, _, _ float64) (AnomalyThresholds, error) {
	key := provider + ":" + symbol + ":" + interval
	return store.thresholds[key], nil
}

type priceReaderStub struct {
	price float64
}

func (reader priceReaderStub) FetchTickerPrice(context.Context, string, string) (float64, error) {
	return reader.price, nil
}

type marketContextFetcherStub struct {
	snapshot marketContextExternalSnapshot
	err      error
}

func (stub marketContextFetcherStub) Fetch(context.Context) (marketContextExternalSnapshot, error) {
	return stub.snapshot, stub.err
}

func buildBars(count int, start time.Time, step time.Duration, basePrice, baseVolume float64) []core.Bar {
	out := make([]core.Bar, 0, count)
	for i := 0; i < count; i++ {
		open := basePrice + float64(i)
		close := open + 0.2
		out = append(out, core.Bar{
			Time:   start.Add(step * time.Duration(i)).UTC(),
			Open:   open,
			High:   close + 0.3,
			Low:    open - 0.3,
			Close:  close,
			Volume: baseVolume,
		})
	}
	return out
}
