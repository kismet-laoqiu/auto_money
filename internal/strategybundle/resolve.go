package strategybundle

import (
	"fmt"
	"path/filepath"

	"quantlab/internal/config"
	"quantlab/internal/watchlist"
)

var loadWatchlist = watchlist.Load

func LoadConfig(path string) (config.Config, *Bundle, error) {
	cfg, err := config.Load(path)
	if err != nil {
		return config.Config{}, nil, err
	}
	if cfg.StrategyBundlePath != "" && !filepath.IsAbs(cfg.StrategyBundlePath) {
		cfg.StrategyBundlePath = filepath.Clean(filepath.Join(filepath.Dir(path), cfg.StrategyBundlePath))
	}
	if cfg.WatchlistPath != "" && !filepath.IsAbs(cfg.WatchlistPath) {
		cfg.WatchlistPath = filepath.Clean(filepath.Join(filepath.Dir(path), cfg.WatchlistPath))
	}
	if cfg.WarehouseConfigPath != "" && !filepath.IsAbs(cfg.WarehouseConfigPath) {
		cfg.WarehouseConfigPath = filepath.Clean(filepath.Join(filepath.Dir(path), cfg.WarehouseConfigPath))
	}
	return ResolveConfig(cfg)
}

func ResolveConfig(cfg config.Config) (config.Config, *Bundle, error) {
	resolved := cfg
	var bundle *Bundle
	if cfg.StrategyBundlePath != "" {
		loaded, err := Load(cfg.StrategyBundlePath)
		if err != nil {
			return config.Config{}, nil, fmt.Errorf("load strategy bundle %s: %w", cfg.StrategyBundlePath, err)
		}
		bundle = &loaded
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
	}

	if resolved.WatchlistPath != "" {
		file, err := loadWatchlist(resolved.WatchlistPath)
		if err != nil {
			return config.Config{}, nil, fmt.Errorf("load watchlist %s: %w", resolved.WatchlistPath, err)
		}
		resolved.Live.Exchange.Symbols = append([]config.LiveSymbolConfig(nil), file.Symbols...)
		resolved.Stream.Symbols = file.SymbolNames()
		if resolved.Stream.Interval == "" {
			resolved.Stream.Interval = file.StreamInterval
		}
		if resolved.Live.Exchange.ProductType == "" {
			resolved.Live.Exchange.ProductType = file.ProductType
		}
	}
	return resolved, bundle, nil
}
