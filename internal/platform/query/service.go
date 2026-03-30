package query

import (
	"context"
	"fmt"

	"quantlab/internal/adapters"
	"quantlab/internal/config"
	"quantlab/internal/core"
	"quantlab/internal/strategybundle"
)

type Loader interface {
	EnsureDataset(ctx context.Context, cacheDir string, spec config.DatasetConfig, refresh bool) (core.Dataset, error)
}

type Config struct {
	Loader Loader
}

type Service struct {
	cfg Config
}

type BarsResult struct {
	Name     string     `json:"name"`
	Provider string     `json:"provider"`
	Symbol   string     `json:"symbol"`
	Interval string     `json:"interval"`
	Bars     []core.Bar `json:"bars"`
}

type FeaturesResult struct {
	Name     string          `json:"name"`
	Provider string          `json:"provider"`
	Symbol   string          `json:"symbol"`
	Interval string          `json:"interval"`
	Index    int             `json:"index"`
	Bar      core.Bar        `json:"bar"`
	Features core.FeatureSet `json:"features"`
}

func NewService(cfg Config) *Service {
	if cfg.Loader == nil {
		cfg.Loader = adapters.NewClient()
	}
	return &Service{cfg: cfg}
}

func (service *Service) BarsFromConfig(ctx context.Context, configPath, datasetName string, refresh bool) (BarsResult, error) {
	_, dataset, _, err := service.resolveDataset(ctx, configPath, datasetName, refresh)
	if err != nil {
		return BarsResult{}, err
	}
	return BarsResult{
		Name:     dataset.Name,
		Provider: dataset.Provider,
		Symbol:   dataset.Symbol,
		Interval: dataset.Interval,
		Bars:     append([]core.Bar(nil), dataset.Bars...),
	}, nil
}

func (service *Service) FeaturesFromConfig(ctx context.Context, configPath, datasetName string, refresh bool, offset int) (FeaturesResult, error) {
	cfg, dataset, _, err := service.resolveDataset(ctx, configPath, datasetName, refresh)
	if err != nil {
		return FeaturesResult{}, err
	}
	if len(dataset.Bars) == 0 {
		return FeaturesResult{}, fmt.Errorf("dataset %s has no bars", dataset.Name)
	}
	index := len(dataset.Bars) - 1 - offset
	if index < 0 || index >= len(dataset.Bars) {
		return FeaturesResult{}, fmt.Errorf("offset %d is out of range for dataset %s", offset, dataset.Name)
	}
	return FeaturesResult{
		Name:     dataset.Name,
		Provider: dataset.Provider,
		Symbol:   dataset.Symbol,
		Interval: dataset.Interval,
		Index:    index,
		Bar:      dataset.Bars[index],
		Features: core.ExtractFeatureSet(dataset.Bars, index, cfg.Strategy),
	}, nil
}

func (service *Service) resolveDataset(ctx context.Context, configPath, datasetName string, refresh bool) (config.Config, core.Dataset, config.DatasetConfig, error) {
	cfg, _, err := strategybundle.LoadConfig(configPath)
	if err != nil {
		return config.Config{}, core.Dataset{}, config.DatasetConfig{}, err
	}
	if len(cfg.Datasets) == 0 {
		return config.Config{}, core.Dataset{}, config.DatasetConfig{}, fmt.Errorf("config %s has no datasets", configPath)
	}
	spec, err := selectDataset(cfg.Datasets, datasetName)
	if err != nil {
		return config.Config{}, core.Dataset{}, config.DatasetConfig{}, err
	}
	dataset, err := service.cfg.Loader.EnsureDataset(ctx, cfg.CacheDir, spec, refresh)
	if err != nil {
		return config.Config{}, core.Dataset{}, config.DatasetConfig{}, err
	}
	return cfg, dataset, spec, nil
}

func selectDataset(specs []config.DatasetConfig, name string) (config.DatasetConfig, error) {
	if name == "" {
		return specs[0], nil
	}
	for _, spec := range specs {
		if spec.Name == name {
			return spec, nil
		}
	}
	return config.DatasetConfig{}, fmt.Errorf("dataset %s not found", name)
}
