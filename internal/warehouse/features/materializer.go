package features

import (
	"context"
	"fmt"
	"strings"
	"time"

	"quantlab/internal/config"
	"quantlab/internal/core"
)

type Store interface {
	LoadBars(ctx context.Context, spec config.DatasetConfig) ([]core.Bar, error)
	UpsertSnapshots(ctx context.Context, snapshots []Snapshot) (int, error)
	FetchSnapshot(ctx context.Context, query FetchSnapshotQuery) (Snapshot, error)
}

type Config struct {
	Store Store
	Now   func() time.Time
}

type Materializer struct {
	store Store
	now   func() time.Time
}

type Request struct {
	StrategyID     string                `json:"strategy_id"`
	Dataset        config.DatasetConfig  `json:"dataset"`
	Strategy       config.StrategyConfig `json:"strategy"`
	FeatureVersion string                `json:"feature_version"`
	TheoryFixture  string                `json:"theory_fixture,omitempty"`
}

type Result struct {
	GeneratedAt    time.Time `json:"generated_at"`
	StrategyID     string    `json:"strategy_id"`
	DatasetName    string    `json:"dataset_name"`
	FeatureVersion string    `json:"feature_version"`
	LoadedBars     int       `json:"loaded_bars"`
	Snapshots      int       `json:"snapshots"`
}

type Snapshot struct {
	StrategyID     string          `json:"strategy_id"`
	FeatureVersion string          `json:"feature_version"`
	DatasetName    string          `json:"dataset_name"`
	Symbol         string          `json:"symbol"`
	Interval       string          `json:"interval"`
	BarTime        time.Time       `json:"bar_time"`
	Features       core.FeatureSet `json:"features"`
	TheoryFixture  string          `json:"theory_fixture,omitempty"`
	GeneratedAt    time.Time       `json:"generated_at"`
}

type FetchSnapshotQuery struct {
	StrategyID     string    `json:"strategy_id"`
	FeatureVersion string    `json:"feature_version"`
	DatasetName    string    `json:"dataset_name"`
	BarTime        time.Time `json:"bar_time"`
}

func NewMaterializer(cfg Config) *Materializer {
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	return &Materializer{store: cfg.Store, now: cfg.Now}
}

func (materializer *Materializer) Materialize(ctx context.Context, request Request) (Result, error) {
	if materializer.store == nil {
		return Result{}, fmt.Errorf("feature store is nil")
	}
	if request.StrategyID == "" {
		return Result{}, fmt.Errorf("strategy_id is required")
	}
	request.Dataset.Name = featureDatasetName(request.Dataset)
	if request.Dataset.Provider == "" || request.Dataset.Symbol == "" || request.Dataset.Interval == "" {
		return Result{}, fmt.Errorf("dataset provider, symbol, and interval are required")
	}
	featureVersion := request.FeatureVersion
	if featureVersion == "" {
		featureVersion = DefaultVersion().String()
	}
	bars, err := materializer.store.LoadBars(ctx, request.Dataset)
	if err != nil {
		return Result{}, err
	}
	generatedAt := materializer.now().UTC()
	snapshots := make([]Snapshot, 0, len(bars))
	for index := range bars {
		snapshots = append(snapshots, Snapshot{
			StrategyID:     request.StrategyID,
			FeatureVersion: featureVersion,
			DatasetName:    request.Dataset.Name,
			Symbol:         request.Dataset.Symbol,
			Interval:       request.Dataset.Interval,
			BarTime:        bars[index].Time.UTC(),
			Features:       core.ExtractFeatureSet(bars, index, request.Strategy),
			TheoryFixture:  request.TheoryFixture,
			GeneratedAt:    generatedAt,
		})
	}
	inserted, err := materializer.store.UpsertSnapshots(ctx, snapshots)
	if err != nil {
		return Result{}, err
	}
	return Result{
		GeneratedAt:    generatedAt,
		StrategyID:     request.StrategyID,
		DatasetName:    request.Dataset.Name,
		FeatureVersion: featureVersion,
		LoadedBars:     len(bars),
		Snapshots:      inserted,
	}, nil
}

func (materializer *Materializer) FetchSnapshot(ctx context.Context, query FetchSnapshotQuery) (Snapshot, error) {
	if materializer.store == nil {
		return Snapshot{}, fmt.Errorf("feature store is nil")
	}
	if query.StrategyID == "" || query.DatasetName == "" || query.BarTime.IsZero() {
		return Snapshot{}, fmt.Errorf("strategy_id, dataset_name, and bar_time are required")
	}
	if query.FeatureVersion == "" {
		query.FeatureVersion = DefaultVersion().String()
	}
	query.BarTime = query.BarTime.UTC()
	return materializer.store.FetchSnapshot(ctx, query)
}

func featureDatasetName(spec config.DatasetConfig) string {
	if spec.Name != "" {
		return spec.Name
	}
	symbol := strings.ToLower(strings.TrimSpace(spec.Symbol))
	interval := strings.ToLower(strings.TrimSpace(spec.Interval))
	if symbol == "" || interval == "" {
		return ""
	}
	return symbol + "_" + interval
}
