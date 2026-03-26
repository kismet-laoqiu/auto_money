package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	CacheDir    string          `yaml:"cache_dir"`
	ArtifactDir string          `yaml:"artifact_dir"`
	Objective   ObjectiveConfig `yaml:"objective"`
	Strategy    StrategyConfig  `yaml:"strategy"`
	Datasets    []DatasetConfig `yaml:"datasets"`
	Stream      StreamConfig    `yaml:"stream"`
	Notify      NotifyConfig    `yaml:"notify"`
}

type ObjectiveConfig struct {
	RiskFreeRate         float64 `yaml:"risk_free_rate"`
	OverfitPenaltyWeight float64 `yaml:"overfit_penalty_weight"`
	DrawdownWeight       float64 `yaml:"drawdown_weight"`
	CalmarWeight         float64 `yaml:"calmar_weight"`
	AnnualReturnWeight   float64 `yaml:"annual_return_weight"`
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
	Name        string `yaml:"name"`
	Provider    string `yaml:"provider"`
	Symbol      string `yaml:"symbol"`
	Interval    string `yaml:"interval"`
	Range       string `yaml:"range"`
	Limit       int    `yaml:"limit"`
	ProductType string `yaml:"product_type"`
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
	return cfg, nil
}
