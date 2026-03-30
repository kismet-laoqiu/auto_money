package ingest

import (
	"context"
	"fmt"
	"strings"
	"time"

	"quantlab/internal/config"
	"quantlab/internal/core"
)

type AggregateStore interface {
	LoadBars(ctx context.Context, spec config.DatasetConfig) ([]core.Bar, error)
	UpsertBars(ctx context.Context, spec config.DatasetConfig, bars []core.Bar) (int, error)
	CountBars(ctx context.Context, spec config.DatasetConfig) (int, error)
}

type AggregateJobConfig struct {
	Store AggregateStore
	Now   func() time.Time
}

type AggregateJob struct {
	cfg AggregateJobConfig
}

type AggregateRequest struct {
	Provider    string   `json:"provider"`
	ProductType string   `json:"product_type"`
	Symbols     []string `json:"symbols"`
	Intervals   []string `json:"intervals"`
}

type AggregateDatasetReport struct {
	Provider       string `json:"provider"`
	Symbol         string `json:"symbol"`
	Interval       string `json:"interval"`
	SourceInterval string `json:"source_interval"`
	SourceRows     int    `json:"source_rows"`
	Inserted       int    `json:"inserted"`
	RowCount       int    `json:"row_count"`
}

type AggregateResult struct {
	GeneratedAt time.Time                `json:"generated_at"`
	Datasets    []AggregateDatasetReport `json:"datasets"`
}

func NewAggregateJob(cfg AggregateJobConfig) *AggregateJob {
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	return &AggregateJob{cfg: cfg}
}

func (job *AggregateJob) Run(ctx context.Context, request AggregateRequest) (AggregateResult, error) {
	if job.cfg.Store == nil {
		return AggregateResult{}, fmt.Errorf("aggregate store is nil")
	}
	if len(request.Symbols) == 0 {
		return AggregateResult{}, fmt.Errorf("at least one symbol is required")
	}
	if request.Provider == "" {
		request.Provider = "bitget"
	}
	intervals := normalizeAggregateIntervals(request.Intervals)
	result := AggregateResult{
		GeneratedAt: job.cfg.Now().UTC(),
		Datasets:    make([]AggregateDatasetReport, 0, len(request.Symbols)*len(intervals)),
	}
	for _, symbol := range request.Symbols {
		for _, interval := range intervals {
			sourceInterval, ok := aggregateSourceInterval(interval)
			if !ok {
				return AggregateResult{}, fmt.Errorf("unsupported aggregate interval %s", interval)
			}
			sourceSpec := config.DatasetConfig{
				Provider:    request.Provider,
				ProductType: request.ProductType,
				Symbol:      symbol,
				Interval:    sourceInterval,
			}
			sourceBars, err := job.cfg.Store.LoadBars(ctx, sourceSpec)
			if err != nil {
				return AggregateResult{}, err
			}
			aggregatedBars, err := AggregateBars(sourceBars, sourceInterval, interval)
			if err != nil {
				return AggregateResult{}, err
			}
			targetSpec := sourceSpec
			targetSpec.Interval = interval
			inserted, err := job.cfg.Store.UpsertBars(ctx, targetSpec, aggregatedBars)
			if err != nil {
				return AggregateResult{}, err
			}
			rowCount, err := job.cfg.Store.CountBars(ctx, targetSpec)
			if err != nil {
				return AggregateResult{}, err
			}
			result.Datasets = append(result.Datasets, AggregateDatasetReport{
				Provider:       request.Provider,
				Symbol:         symbol,
				Interval:       interval,
				SourceInterval: sourceInterval,
				SourceRows:     len(sourceBars),
				Inserted:       inserted,
				RowCount:       rowCount,
			})
		}
	}
	return result, nil
}

func AggregateBars(bars []core.Bar, sourceInterval, targetInterval string) ([]core.Bar, error) {
	sourceStep, ok := intervalDuration(sourceInterval)
	if !ok {
		return nil, fmt.Errorf("unsupported source interval %s", sourceInterval)
	}
	targetStep, ok := intervalDuration(targetInterval)
	if !ok {
		return nil, fmt.Errorf("unsupported target interval %s", targetInterval)
	}
	if targetStep <= sourceStep || targetStep%sourceStep != 0 {
		return nil, fmt.Errorf("cannot aggregate %s into %s", sourceInterval, targetInterval)
	}
	expected := int(targetStep / sourceStep)
	if len(bars) == 0 {
		return nil, nil
	}
	out := make([]core.Bar, 0, len(bars)/expected)
	bucketStart := bars[0].Time.UTC().Truncate(targetStep)
	bucket := make([]core.Bar, 0, expected)
	flush := func() {
		if len(bucket) != expected || !isContiguousBucket(bucket, sourceStep) {
			return
		}
		out = append(out, aggregateBucket(bucket, bucketStart))
	}
	for _, bar := range bars {
		current := bar.Time.UTC().Truncate(targetStep)
		if !current.Equal(bucketStart) {
			flush()
			bucketStart = current
			bucket = bucket[:0]
		}
		bucket = append(bucket, bar)
	}
	flush()
	return out, nil
}

func aggregateBucket(bars []core.Bar, bucketStart time.Time) core.Bar {
	aggregated := core.Bar{
		Time:   bucketStart,
		Open:   bars[0].Open,
		High:   bars[0].High,
		Low:    bars[0].Low,
		Close:  bars[len(bars)-1].Close,
		Volume: 0,
	}
	for _, bar := range bars {
		if bar.High > aggregated.High {
			aggregated.High = bar.High
		}
		if bar.Low < aggregated.Low {
			aggregated.Low = bar.Low
		}
		aggregated.Volume += bar.Volume
	}
	return aggregated
}

func isContiguousBucket(bars []core.Bar, step time.Duration) bool {
	for index := 1; index < len(bars); index++ {
		if bars[index].Time.UTC().Sub(bars[index-1].Time.UTC()) != step {
			return false
		}
	}
	return true
}

func aggregateSourceInterval(interval string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(interval)) {
	case "5m", "15m", "1h":
		return "1m", true
	case "4h", "1d":
		return "1h", true
	default:
		return "", false
	}
}

func normalizeAggregateIntervals(intervals []string) []string {
	if len(intervals) == 0 {
		return []string{"5m", "15m", "1h", "4h", "1d"}
	}
	out := make([]string, 0, len(intervals))
	seen := map[string]bool{}
	for _, interval := range intervals {
		normalized := strings.ToLower(strings.TrimSpace(interval))
		if normalized == "" || seen[normalized] {
			continue
		}
		seen[normalized] = true
		out = append(out, normalized)
	}
	return out
}
