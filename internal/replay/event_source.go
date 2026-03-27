package replay

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"quantlab/internal/adapters"
	"quantlab/internal/config"
	"quantlab/internal/market"
)

func BuildReplaySpecs(cfg config.Config) []config.DatasetConfig {
	if len(cfg.Datasets) > 0 {
		specs := make([]config.DatasetConfig, len(cfg.Datasets))
		copy(specs, cfg.Datasets)
		return specs
	}
	specs := make([]config.DatasetConfig, 0, len(cfg.Live.Exchange.Symbols))
	for _, symbol := range cfg.Live.Exchange.Symbols {
		specs = append(specs, config.DatasetConfig{
			Name:        strings.ToLower(symbol.Symbol) + "_replay",
			Provider:    "bitget",
			Symbol:      symbol.Symbol,
			Interval:    "1m",
			Limit:       240,
			ProductType: cfg.Live.Exchange.ProductType,
		})
	}
	return specs
}

func LoadEvents(ctx context.Context, cacheDir string, specs []config.DatasetConfig, refresh bool) ([]market.MarketEvent, error) {
	client := adapters.NewClient()
	events := make([]market.MarketEvent, 0)
	for _, spec := range specs {
		dataset, err := client.EnsureDataset(ctx, cacheDir, spec, refresh)
		if err != nil {
			fallback, ok := fallbackReplaySpec(spec)
			if !ok {
				return nil, err
			}
			dataset, err = client.EnsureDataset(ctx, cacheDir, fallback, refresh)
			if err != nil {
				return nil, fmt.Errorf("load replay dataset %s: %w", spec.Symbol, err)
			}
		}
		interval := dataset.Interval
		if interval == "" {
			interval = spec.Interval
		}
		for _, bar := range dataset.Bars {
			tsMillis := bar.Time.UTC().UnixMilli()
			events = append(events, market.BarClosedEvent{
				EventIDValue: fmt.Sprintf("replay:%s:%s:%d", dataset.Symbol, interval, tsMillis),
				SymbolValue:  dataset.Symbol,
				Interval:     interval,
				Ts:           bar.Time.UTC(),
				Open:         bar.Open,
				High:         bar.High,
				Low:          bar.Low,
				Close:        bar.Close,
				Volume:       bar.Volume,
			})
		}
	}
	sort.Slice(events, func(i, j int) bool {
		ti := events[i].EventTime()
		tj := events[j].EventTime()
		if ti.Equal(tj) {
			return events[i].Symbol() < events[j].Symbol()
		}
		return ti.Before(tj)
	})
	return events, nil
}

func fallbackReplaySpec(spec config.DatasetConfig) (config.DatasetConfig, bool) {
	if !strings.EqualFold(spec.Provider, "bitget") {
		return config.DatasetConfig{}, false
	}
	fallback := spec
	fallback.Provider = "binance"
	if fallback.Interval == "" {
		fallback.Interval = "1d"
	}
	if fallback.Limit == 0 {
		fallback.Limit = 1000
	}
	return fallback, true
}
