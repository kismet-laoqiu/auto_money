package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	CacheDir            string          `yaml:"cache_dir"`
	ArtifactDir         string          `yaml:"artifact_dir"`
	StrategyBundlePath  string          `yaml:"strategy_bundle_path"`
	WatchlistPath       string          `yaml:"watchlist_path"`
	WarehouseConfigPath string          `yaml:"warehouse_config_path"`
	Objective           ObjectiveConfig `yaml:"objective"`
	Strategy            StrategyConfig  `yaml:"strategy"`
	Datasets            []DatasetConfig `yaml:"datasets"`
	Stream              StreamConfig    `yaml:"stream"`
	Notify              NotifyConfig    `yaml:"notify"`
	Insights            InsightsConfig  `yaml:"insights"`
	Live                LiveConfig      `yaml:"live"`
}

type ObjectiveConfig struct {
	RiskFreeRate         float64 `yaml:"risk_free_rate"`
	OverfitPenaltyWeight float64 `yaml:"overfit_penalty_weight"`
	DrawdownWeight       float64 `yaml:"drawdown_weight"`
	CalmarWeight         float64 `yaml:"calmar_weight"`
	PnLWeight            float64 `yaml:"pnl_weight"`
}

type StrategyConfig struct {
	FastSMA         int     `yaml:"fast_sma"`
	SlowSMA         int     `yaml:"slow_sma"`
	PivotWindow     int     `yaml:"pivot_window"`
	LevelLookback   int     `yaml:"level_lookback"`
	LevelTolerance  float64 `yaml:"level_tolerance"`
	FibTolerance    float64 `yaml:"fib_tolerance"`
	SignalThreshold float64 `yaml:"signal_threshold"`
	StopATR         float64 `yaml:"stop_atr"`
	RewardRisk      float64 `yaml:"reward_risk"`
	MaxHoldBars     int     `yaml:"max_hold_bars"`
	CooldownBars    int     `yaml:"cooldown_bars"`
	CommissionBps   float64 `yaml:"commission_bps"`
	SlippageBps     float64 `yaml:"slippage_bps"`
	ATRWindow       int     `yaml:"atr_window"`
}

type DatasetConfig struct {
	Name        string    `yaml:"name"`
	Provider    string    `yaml:"provider"`
	Symbol      string    `yaml:"symbol"`
	Interval    string    `yaml:"interval"`
	Range       string    `yaml:"range"`
	Limit       int       `yaml:"limit"`
	ProductType string    `yaml:"product_type"`
	StartTime   time.Time `yaml:"-"`
	EndTime     time.Time `yaml:"-"`
}

type StreamConfig struct {
	BitgetURL     string   `yaml:"bitget_url"`
	Symbols       []string `yaml:"symbols"`
	Interval      string   `yaml:"interval"`
	SnapshotLimit int      `yaml:"snapshot_limit"`
}

type NotifyConfig struct {
	Enable          bool   `yaml:"enable"`
	DingTalkWebhook string `yaml:"dingtalk_webhook"`
	DingTalkSecret  string `yaml:"dingtalk_secret"`
	DingTalkKeyword string `yaml:"dingtalk_keyword"`
}

type InsightsConfig struct {
	Enabled      bool                    `yaml:"enabled"`
	ScanInterval time.Duration           `yaml:"scan_interval"`
	PriceTTL     time.Duration           `yaml:"price_ttl"`
	DailySignal  DailySignalAlertConfig  `yaml:"daily_signal"`
	Anomaly      AnomalyAlertConfig      `yaml:"anomaly"`
	Dashboard    DashboardOperatorConfig `yaml:"dashboard"`
}

type DailySignalAlertConfig struct {
	Interval      string  `yaml:"interval"`
	LookbackBars  int     `yaml:"lookback_bars"`
	MinScore      float64 `yaml:"min_score"`
	ScoreQuantile float64 `yaml:"score_quantile"`
	CooldownBars  int     `yaml:"cooldown_bars"`
}

type AnomalyAlertConfig struct {
	Interval        string  `yaml:"interval"`
	LookbackBars    int     `yaml:"lookback_bars"`
	RollingWindow   int     `yaml:"rolling_window"`
	MinMovePct      float64 `yaml:"min_move_pct"`
	MoveQuantile    float64 `yaml:"move_quantile"`
	MinVolumeZScore float64 `yaml:"min_volume_zscore"`
	MinVolumeRatio  float64 `yaml:"min_volume_ratio"`
	VolumeQuantile  float64 `yaml:"volume_quantile"`
	CooldownBars    int     `yaml:"cooldown_bars"`
}

type DashboardOperatorConfig struct {
	PlatformctlPath    string `yaml:"platformctl_path"`
	MarketdRestartUnit string `yaml:"marketd_restart_unit"`
	RenderSymbolLimit  int    `yaml:"render_symbol_limit"`
}

type LiveConfig struct {
	Enabled  bool           `yaml:"enabled"`
	Runtime  RuntimeConfig  `yaml:"runtime"`
	Exchange ExchangeConfig `yaml:"exchange"`
	Risk     RiskConfig     `yaml:"risk"`
	Agent    AgentConfig    `yaml:"agent"`
}

type RuntimeConfig struct {
	ArmingState string `yaml:"arming_state"`
	StateDBPath string `yaml:"state_db_path"`
	ObserveOnly bool   `yaml:"observe_only"`
}

type ExchangeConfig struct {
	Venue         string             `yaml:"venue"`
	ProductType   string             `yaml:"product_type"`
	RESTBaseURL   string             `yaml:"rest_base_url"`
	PublicWSURL   string             `yaml:"public_ws_url"`
	PrivateWSURL  string             `yaml:"private_ws_url"`
	APIKeyEnv     string             `yaml:"api_key_env"`
	APISecretEnv  string             `yaml:"api_secret_env"`
	PassphraseEnv string             `yaml:"passphrase_env"`
	MarginMode    string             `yaml:"margin_mode"`
	PositionMode  string             `yaml:"position_mode"`
	Symbols       []LiveSymbolConfig `yaml:"symbols"`
}

type LiveSymbolConfig struct {
	Symbol      string  `yaml:"symbol"`
	MaxNotional float64 `yaml:"max_notional"`
	MaxTranches int     `yaml:"max_tranches"`
}

type RiskConfig struct {
	MaxLeverage int `yaml:"max_leverage"`
}

type AgentConfig struct {
	Enabled      bool `yaml:"enabled"`
	AdvisoryOnly bool `yaml:"advisory_only"`
}

func Load(path string) (Config, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config %s: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(content, &cfg); err != nil {
		return Config{}, fmt.Errorf("decode config %s: %w", path, err)
	}
	if cfg.CacheDir == "" {
		cfg.CacheDir = "data"
	}
	if cfg.ArtifactDir == "" {
		cfg.ArtifactDir = "artifacts"
	}
	applyInsightsDefaults(&cfg.Insights)
	applyLiveDefaults(&cfg.Live)
	return cfg, nil
}

func applyInsightsDefaults(cfg *InsightsConfig) {
	if cfg.ScanInterval <= 0 {
		cfg.ScanInterval = time.Minute
	}
	if cfg.PriceTTL <= 0 {
		cfg.PriceTTL = 30 * time.Second
	}
	if cfg.DailySignal.Interval == "" {
		cfg.DailySignal.Interval = "1d"
	}
	if cfg.DailySignal.LookbackBars <= 0 {
		cfg.DailySignal.LookbackBars = 1095
	}
	if cfg.DailySignal.MinScore <= 0 {
		cfg.DailySignal.MinScore = 4.25
	}
	if cfg.DailySignal.ScoreQuantile <= 0 {
		cfg.DailySignal.ScoreQuantile = 0.985
	}
	if cfg.DailySignal.CooldownBars <= 0 {
		cfg.DailySignal.CooldownBars = 5
	}
	if cfg.Anomaly.Interval == "" {
		cfg.Anomaly.Interval = "15m"
	}
	if cfg.Anomaly.LookbackBars <= 0 {
		cfg.Anomaly.LookbackBars = 105120
	}
	if cfg.Anomaly.RollingWindow <= 0 {
		cfg.Anomaly.RollingWindow = 20
	}
	if cfg.Anomaly.MinMovePct <= 0 {
		cfg.Anomaly.MinMovePct = 2.5
	}
	if cfg.Anomaly.MoveQuantile <= 0 {
		cfg.Anomaly.MoveQuantile = 0.995
	}
	if cfg.Anomaly.MinVolumeZScore <= 0 {
		cfg.Anomaly.MinVolumeZScore = 3.5
	}
	if cfg.Anomaly.MinVolumeRatio <= 0 {
		cfg.Anomaly.MinVolumeRatio = 3.0
	}
	if cfg.Anomaly.VolumeQuantile <= 0 {
		cfg.Anomaly.VolumeQuantile = 0.995
	}
	if cfg.Anomaly.CooldownBars <= 0 {
		cfg.Anomaly.CooldownBars = 8
	}
	if cfg.Dashboard.PlatformctlPath == "" {
		cfg.Dashboard.PlatformctlPath = "./bin/platformctl"
	}
	if cfg.Dashboard.MarketdRestartUnit == "" {
		cfg.Dashboard.MarketdRestartUnit = "quantlab-marketd.service"
	}
	if cfg.Dashboard.RenderSymbolLimit <= 0 {
		cfg.Dashboard.RenderSymbolLimit = 24
	}
}

func applyLiveDefaults(cfg *LiveConfig) {
	if cfg.Runtime.ArmingState == "" {
		cfg.Runtime.ArmingState = "safe"
	}
	if cfg.Runtime.StateDBPath == "" {
		cfg.Runtime.StateDBPath = "var/live-state.db"
	}
	if cfg.Exchange.Venue == "" {
		cfg.Exchange.Venue = "bitget"
	}
	if cfg.Exchange.ProductType == "" {
		cfg.Exchange.ProductType = "USDT-FUTURES"
	}
	if cfg.Exchange.RESTBaseURL == "" {
		cfg.Exchange.RESTBaseURL = "https://api.bitget.com"
	}
	if cfg.Exchange.PublicWSURL == "" {
		cfg.Exchange.PublicWSURL = "wss://ws.bitget.com/v2/ws/public"
	}
	if cfg.Exchange.PrivateWSURL == "" {
		cfg.Exchange.PrivateWSURL = "wss://ws.bitget.com/v2/ws/private"
	}
	if cfg.Exchange.APIKeyEnv == "" {
		cfg.Exchange.APIKeyEnv = "BITGET_API_KEY"
	}
	if cfg.Exchange.APISecretEnv == "" {
		cfg.Exchange.APISecretEnv = "BITGET_API_SECRET"
	}
	if cfg.Exchange.PassphraseEnv == "" {
		cfg.Exchange.PassphraseEnv = "BITGET_PASSPHRASE"
	}
	if cfg.Exchange.MarginMode == "" {
		cfg.Exchange.MarginMode = "isolated"
	}
	if cfg.Exchange.PositionMode == "" {
		cfg.Exchange.PositionMode = "one_way_mode"
	}
	if cfg.Risk.MaxLeverage == 0 {
		cfg.Risk.MaxLeverage = 3
	}
}
