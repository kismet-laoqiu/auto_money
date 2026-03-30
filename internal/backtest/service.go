package backtest

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	"quantlab/internal/adapters"
	"quantlab/internal/config"
	"quantlab/internal/core"
	sqlitepkg "quantlab/internal/store/sqlite"
)

type Loader interface {
	EnsureDataset(ctx context.Context, cacheDir string, spec config.DatasetConfig, refresh bool) (core.Dataset, error)
}

type ArtifactWriter interface {
	WriteReportArtifacts(root string, report core.Report) error
}

type ArchiveStore interface {
	AppendEvent(ctx context.Context, source string, evt sqlitepkg.LogEvent, raw []byte) (int64, error)
}

type Config struct {
	Loader         Loader
	Writer         ArtifactWriter
	Store          ArchiveStore
	Source         string
	Now            func() time.Time
	FeatureVersion string
}

type Service struct {
	runner         *Runner
	reports        *ReportWriter
	store          ArchiveStore
	source         string
	featureVersion string
}

type Result struct {
	GeneratedAt    time.Time          `json:"generated_at"`
	MetricName     string             `json:"metric_name"`
	ObjectiveScore float64            `json:"objective_score"`
	FinalScore     float64            `json:"final_score"`
	Robust         core.RobustScore   `json:"robust"`
	Aggregate      map[string]float64 `json:"aggregate"`
	Reports        []core.Report      `json:"reports"`
}

type RunArchiveEvent struct {
	EventIDValue   string            `json:"event_id"`
	RunID          string            `json:"run_id"`
	StrategyID     string            `json:"strategy_id"`
	ConfigPath     string            `json:"config_path"`
	GeneratedAt    time.Time         `json:"generated_at"`
	MetricName     string            `json:"metric_name"`
	ObjectiveScore float64           `json:"objective_score"`
	FinalScore     float64           `json:"final_score"`
	FeatureVersion string            `json:"feature_version"`
	BundleVersion  string            `json:"bundle_version,omitempty"`
	BundleRootPath string            `json:"bundle_root_path,omitempty"`
	ArtifactDir    string            `json:"artifact_dir"`
	DatasetHash    string            `json:"dataset_hash"`
	Datasets       []ArchivedDataset `json:"datasets"`
}

type ArchivedDataset struct {
	Name     string `json:"name"`
	Provider string `json:"provider"`
	Symbol   string `json:"symbol"`
	Interval string `json:"interval"`
	Bars     int    `json:"bars"`
	Hash     string `json:"hash"`
}

func NewService(cfg Config) *Service {
	if cfg.Loader == nil {
		cfg.Loader = adapters.NewClient()
	}
	if cfg.Writer == nil {
		cfg.Writer = reportArtifactWriter{}
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	if cfg.Source == "" {
		cfg.Source = "backtest"
	}
	if cfg.FeatureVersion == "" {
		cfg.FeatureVersion = core.FeatureSetVersion
	}
	return &Service{
		runner:         NewRunner(RunnerConfig{Loader: cfg.Loader, Writer: cfg.Writer}),
		reports:        NewReportWriter(ReportWriterConfig{Now: cfg.Now}),
		store:          cfg.Store,
		source:         cfg.Source,
		featureVersion: cfg.FeatureVersion,
	}
}

func (service *Service) RunConfig(ctx context.Context, configPath string, refresh bool) (Result, error) {
	run, err := service.runner.RunConfig(ctx, configPath, refresh)
	if err != nil {
		return Result{}, err
	}
	result := service.reports.Build(run.Reports)
	if err := service.reports.WriteArtifacts(run.Config.ArtifactDir, result); err != nil {
		return Result{}, err
	}
	if err := service.archive(ctx, configPath, run, result); err != nil {
		return Result{}, err
	}
	return result, nil
}

func (service *Service) archive(ctx context.Context, configPath string, run Run, result Result) error {
	if service.store == nil {
		return nil
	}
	event, err := buildRunArchiveEvent(configPath, run, result, service.featureVersion)
	if err != nil {
		return err
	}
	body, err := json.Marshal(event)
	if err != nil {
		return err
	}
	_, err = service.store.AppendEvent(ctx, service.source, event, body)
	return err
}

func buildRunArchiveEvent(configPath string, run Run, result Result, featureVersion string) (RunArchiveEvent, error) {
	datasets, datasetHash, err := buildArchivedDatasets(run.Datasets)
	if err != nil {
		return RunArchiveEvent{}, err
	}
	strategyID := archiveStrategyID(configPath, run)
	runID := fmt.Sprintf("backtest:%s:%d", strategyID, result.GeneratedAt.UnixNano())
	event := RunArchiveEvent{
		EventIDValue:   runID,
		RunID:          runID,
		StrategyID:     strategyID,
		ConfigPath:     configPath,
		GeneratedAt:    result.GeneratedAt,
		MetricName:     result.MetricName,
		ObjectiveScore: result.ObjectiveScore,
		FinalScore:     result.FinalScore,
		FeatureVersion: featureVersion,
		ArtifactDir:    run.Config.ArtifactDir,
		DatasetHash:    datasetHash,
		Datasets:       datasets,
	}
	if run.Bundle != nil {
		event.BundleVersion = run.Bundle.Version
		event.BundleRootPath = run.Bundle.RootPath
	}
	return event, nil
}

func buildArchivedDatasets(datasets []core.Dataset) ([]ArchivedDataset, string, error) {
	archived := make([]ArchivedDataset, 0, len(datasets))
	aggregate := sha256.New()
	for _, dataset := range datasets {
		body, err := json.Marshal(dataset)
		if err != nil {
			return nil, "", err
		}
		digest := digestHex(body)
		archived = append(archived, ArchivedDataset{
			Name:     dataset.Name,
			Provider: dataset.Provider,
			Symbol:   dataset.Symbol,
			Interval: dataset.Interval,
			Bars:     len(dataset.Bars),
			Hash:     digest,
		})
		_, _ = aggregate.Write([]byte(digest))
		_, _ = aggregate.Write([]byte{'\n'})
	}
	return archived, hex.EncodeToString(aggregate.Sum(nil)), nil
}

func digestHex(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

func archiveStrategyID(configPath string, run Run) string {
	if run.Bundle != nil && run.Bundle.StrategyID != "" {
		return run.Bundle.StrategyID
	}
	name := filepath.Base(configPath)
	ext := filepath.Ext(name)
	return name[:len(name)-len(ext)]
}

func (event RunArchiveEvent) EventID() string      { return event.EventIDValue }
func (event RunArchiveEvent) Symbol() string       { return event.StrategyID }
func (event RunArchiveEvent) EventTime() time.Time { return event.GeneratedAt }
func (event RunArchiveEvent) Kind() string         { return "backtest.run.completed" }
