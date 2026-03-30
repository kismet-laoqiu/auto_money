package backtest

import (
	"context"

	"quantlab/internal/adapters"
	"quantlab/internal/config"
	"quantlab/internal/core"
	"quantlab/internal/strategybundle"
)

type RunnerConfig struct {
	Loader Loader
	Writer ArtifactWriter
}

type Run struct {
	Config   config.Config
	Bundle   *strategybundle.Bundle
	Datasets []core.Dataset
	Reports  []core.Report
}

type Runner struct {
	cfg RunnerConfig
}

func NewRunner(cfg RunnerConfig) *Runner {
	if cfg.Loader == nil {
		cfg.Loader = adapters.NewClient()
	}
	if cfg.Writer == nil {
		cfg.Writer = reportArtifactWriter{}
	}
	return &Runner{cfg: cfg}
}

func (runner *Runner) RunConfig(ctx context.Context, configPath string, refresh bool) (Run, error) {
	cfg, bundle, err := strategybundle.LoadConfig(configPath)
	if err != nil {
		return Run{}, err
	}
	return runner.runResolved(ctx, cfg, bundle, refresh)
}

func (runner *Runner) Run(ctx context.Context, cfg config.Config, refresh bool) (Run, error) {
	return runner.runResolved(ctx, cfg, nil, refresh)
}

func (runner *Runner) runResolved(ctx context.Context, cfg config.Config, bundle *strategybundle.Bundle, refresh bool) (Run, error) {
	reports := make([]core.Report, 0, len(cfg.Datasets))
	datasets := make([]core.Dataset, 0, len(cfg.Datasets))
	for _, spec := range cfg.Datasets {
		dataset, err := runner.cfg.Loader.EnsureDataset(ctx, cfg.CacheDir, spec, refresh)
		if err != nil {
			return Run{}, err
		}
		report := core.BacktestDataset(dataset, cfg.Strategy, cfg.Objective)
		if err := runner.cfg.Writer.WriteReportArtifacts(cfg.ArtifactDir, report); err != nil {
			return Run{}, err
		}
		datasets = append(datasets, dataset)
		reports = append(reports, report)
	}
	return Run{Config: cfg, Bundle: bundle, Datasets: datasets, Reports: reports}, nil
}

type reportArtifactWriter struct{}

func (reportArtifactWriter) WriteReportArtifacts(root string, report core.Report) error {
	return adapters.WriteReportArtifacts(root, report)
}
