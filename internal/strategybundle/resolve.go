package strategybundle

import (
	"fmt"
	"path/filepath"

	"quantlab/internal/config"
)

func LoadConfig(path string) (config.Config, *Bundle, error) {
	cfg, err := config.Load(path)
	if err != nil {
		return config.Config{}, nil, err
	}
	if cfg.StrategyBundlePath != "" && !filepath.IsAbs(cfg.StrategyBundlePath) {
		cfg.StrategyBundlePath = filepath.Clean(filepath.Join(filepath.Dir(path), cfg.StrategyBundlePath))
	}
	return ResolveConfig(cfg)
}

func ResolveConfig(cfg config.Config) (config.Config, *Bundle, error) {
	if cfg.StrategyBundlePath == "" {
		return cfg, nil, nil
	}

	bundle, err := Load(cfg.StrategyBundlePath)
	if err != nil {
		return config.Config{}, nil, fmt.Errorf("load strategy bundle %s: %w", cfg.StrategyBundlePath, err)
	}

	resolved := cfg
	resolved.Strategy = bundle.Strategy
	resolved.Objective = bundle.Objective
	if len(bundle.Universe.Datasets) > 0 {
		resolved.Datasets = append([]config.DatasetConfig(nil), bundle.Universe.Datasets...)
	}
	if bundle.Risk.MaxLeverage > 0 {
		resolved.Live.Risk.MaxLeverage = bundle.Risk.MaxLeverage
	}
	if bundle.Risk.ProductType != "" {
		resolved.Live.Exchange.ProductType = bundle.Risk.ProductType
	}
	if bundle.Risk.MarginMode != "" {
		resolved.Live.Exchange.MarginMode = bundle.Risk.MarginMode
	}
	if len(bundle.Risk.Symbols) > 0 {
		resolved.Live.Exchange.Symbols = append([]config.LiveSymbolConfig(nil), bundle.Risk.Symbols...)
	}
	return resolved, &bundle, nil
}
