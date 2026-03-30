package strategybundle

import "quantlab/internal/config"

type Bundle struct {
	RootPath    string
	StrategyID  string
	Version     string
	Description string
	Strategy    config.StrategyConfig
	Objective   config.ObjectiveConfig
	Universe    Universe
	Risk        Risk
	ScoreCEL    string
	GatesCEL    string
}

type Universe struct {
	Datasets []config.DatasetConfig `yaml:"datasets"`
	Symbols  []string               `yaml:"symbols"`
}

type Risk struct {
	MaxLeverage int                       `yaml:"max_leverage"`
	MarginMode  string                    `yaml:"margin_mode"`
	ProductType string                    `yaml:"product_type"`
	Symbols     []config.LiveSymbolConfig `yaml:"symbols"`
}
