package insights

import (
	"time"

	"quantlab/internal/core"
)

type AlertType string

const (
	AlertTypeDailySignal AlertType = "daily_signal"
	AlertTypeMarketAlert AlertType = "market_alert"
)

type MarketAlertSignal string

const (
	MarketAlertVolumeSpike MarketAlertSignal = "volume_spike"
	MarketAlertSurge       MarketAlertSignal = "surge"
	MarketAlertDump        MarketAlertSignal = "dump"
	MarketAlertCombo       MarketAlertSignal = "combo"
)

type Event struct {
	EventIDValue string            `json:"event_id"`
	SymbolValue  string            `json:"symbol"`
	Ts           time.Time         `json:"ts"`
	AlertType    AlertType         `json:"alert_type"`
	Interval     string            `json:"interval"`
	Title        string            `json:"title"`
	Summary      string            `json:"summary"`
	Details      []string          `json:"details"`
	Direction    string            `json:"direction,omitempty"`
	Score        float64           `json:"score,omitempty"`
	Threshold    float64           `json:"threshold,omitempty"`
	Signal       MarketAlertSignal `json:"signal,omitempty"`
}

func (event Event) EventID() string      { return event.EventIDValue }
func (event Event) Symbol() string       { return event.SymbolValue }
func (event Event) EventTime() time.Time { return event.Ts }
func (event Event) Kind() string         { return "insight.alert" }

type DashboardReport struct {
	GeneratedAt   time.Time              `json:"generated_at"`
	Watchlist     string                 `json:"watchlist"`
	Symbols       []SymbolSnapshot       `json:"symbols"`
	Alerts        []Event                `json:"alerts"`
	MarketContext *MarketContextSnapshot `json:"market_context,omitempty"`
}

type MarketContextSnapshot struct {
	BTC              *MarketAssetSnapshot      `json:"btc,omitempty"`
	ETH              *MarketAssetSnapshot      `json:"eth,omitempty"`
	RelativeStrength *RelativeStrengthSnapshot `json:"relative_strength,omitempty"`
	FearGreed        *FearGreedSnapshot        `json:"fear_greed,omitempty"`
	Hashrate         *HashrateSnapshot         `json:"hashrate,omitempty"`
	Halving          *HalvingSnapshot          `json:"halving,omitempty"`
	BalancedPrice    *MarketLevelSnapshot      `json:"balanced_price,omitempty"`
	MVRV             *MarketLevelSnapshot      `json:"mvrv,omitempty"`
	Mnav             *TreasuryPremiumSnapshot  `json:"mnav,omitempty"`
}

type MarketAssetSnapshot struct {
	Symbol        string    `json:"symbol"`
	CurrentPrice  float64   `json:"current_price"`
	Return7d      float64   `json:"return_7d"`
	Return30d     float64   `json:"return_30d"`
	Return90d     float64   `json:"return_90d"`
	WMA200        float64   `json:"wma_200,omitempty"`
	PriceToWMA200 float64   `json:"price_to_wma_200,omitempty"`
	UpdatedAt     time.Time `json:"updated_at,omitempty"`
}

type RelativeStrengthSnapshot struct {
	BTCMinusETH7d  float64 `json:"btc_minus_eth_7d"`
	BTCMinusETH30d float64 `json:"btc_minus_eth_30d"`
	BTCMinusETH90d float64 `json:"btc_minus_eth_90d"`
}

type FearGreedSnapshot struct {
	Value          int       `json:"value"`
	Classification string    `json:"classification"`
	UpdatedAt      time.Time `json:"updated_at,omitempty"`
}

type HashrateSnapshot struct {
	CurrentEH float64   `json:"current_eh"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}

type HalvingSnapshot struct {
	CurrentBlock    int64     `json:"current_block"`
	TargetBlock     int64     `json:"target_block"`
	BlocksRemaining int64     `json:"blocks_remaining"`
	DaysRemaining   float64   `json:"days_remaining"`
	CurrentReward   float64   `json:"current_reward"`
	NextReward      float64   `json:"next_reward"`
	EstimatedAt     time.Time `json:"estimated_at,omitempty"`
}

type MarketLevelSnapshot struct {
	Value     float64   `json:"value"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}

type TreasuryPremiumSnapshot struct {
	ETHPrice float64                  `json:"eth_price,omitempty"`
	MSTR     *TreasuryCompanySnapshot `json:"mstr,omitempty"`
	BMNR     *TreasuryCompanySnapshot `json:"bmnr,omitempty"`
}

type TreasuryCompanySnapshot struct {
	Symbol          string  `json:"symbol"`
	Holdings        float64 `json:"holdings,omitempty"`
	StockPrice      float64 `json:"stock_price"`
	Ratio           float64 `json:"ratio,omitempty"`
	BasicRatio      float64 `json:"basic_ratio,omitempty"`
	EnterpriseRatio float64 `json:"enterprise_ratio,omitempty"`
}

type SymbolSnapshot struct {
	Symbol            string           `json:"symbol"`
	LatestPrice       float64          `json:"latest_price"`
	Latest15mCloseAt  time.Time        `json:"latest_15m_close_at"`
	Latest1dCloseAt   time.Time        `json:"latest_1d_close_at"`
	DailySignal       *SignalSnapshot  `json:"daily_signal,omitempty"`
	DailyFeatures     FeatureSnapshot  `json:"daily_features"`
	LatestMarketAlert *AnomalySnapshot `json:"latest_market_alert,omitempty"`
}

type SignalSnapshot struct {
	Side      string    `json:"side"`
	Score     float64   `json:"score"`
	Threshold float64   `json:"threshold"`
	BarTime   time.Time `json:"bar_time"`
	Reasons   []string  `json:"reasons"`
}

type FeatureSnapshot struct {
	RSI14                   float64 `json:"rsi14"`
	NeedleDropPct           float64 `json:"needle_drop_pct"`
	ReclaimPct              float64 `json:"reclaim_pct"`
	VolumeZScore            float64 `json:"volume_zscore"`
	RelativeVolumeRatio     float64 `json:"relative_volume_ratio"`
	BreakoutVolumeConfirmed bool    `json:"breakout_volume_confirmed"`
	TrendUp                 bool    `json:"trend_up"`
	TrendDown               bool    `json:"trend_down"`
	Range                   bool    `json:"range"`
	HighVol                 bool    `json:"high_vol"`
}

type AnomalySnapshot struct {
	Signal         MarketAlertSignal `json:"signal"`
	MovePct        float64           `json:"move_pct"`
	MoveThreshold  float64           `json:"move_threshold"`
	Volume         float64           `json:"volume"`
	VolumeBaseline float64           `json:"volume_baseline"`
	VolumeRatio    float64           `json:"volume_ratio"`
	VolumeZScore   float64           `json:"volume_zscore"`
	BarTime        time.Time         `json:"bar_time"`
}

type AnomalyThresholds struct {
	MovePct float64
	Volume  float64
}

type DailyAnalysis struct {
	Signal    core.Signal
	Threshold float64
	Features  core.FeatureSet
	Qualified bool
}

type AnomalyAnalysis struct {
	Signal       MarketAlertSignal
	MovePct      float64
	MoveLimit    float64
	Volume       float64
	VolumeLimit  float64
	VolumeRatio  float64
	VolumeZScore float64
	Qualified    bool
}
