package ingest

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"quantlab/internal/config"
	"quantlab/internal/core"
)

var DefaultIntervals = []string{"1m", "5m", "15m", "1h", "4h", "1d", "1w"}

type Fetcher interface {
	FetchBars(ctx context.Context, spec config.DatasetConfig) ([]core.Bar, error)
}

type Store interface {
	UpsertBars(ctx context.Context, spec config.DatasetConfig, bars []core.Bar) (int, error)
	CountBars(ctx context.Context, spec config.DatasetConfig) (int, error)
}

type JobConfig struct {
	Fetcher Fetcher
	Store   Store
	Now     func() time.Time
}

type HistoricalJob struct {
	cfg JobConfig
}

type Request struct {
	Provider     string   `json:"provider"`
	ProductType  string   `json:"product_type"`
	Symbols      []string `json:"symbols"`
	Intervals    []string `json:"intervals"`
	Limit        int      `json:"limit"`
	HorizonDays  int      `json:"horizon_days"`
	ArtifactRoot string   `json:"artifact_root"`
}

type DatasetReport struct {
	Provider string `json:"provider"`
	Symbol   string `json:"symbol"`
	Interval string `json:"interval"`
	Inserted int    `json:"inserted"`
	RowCount int    `json:"row_count"`
	Checksum string `json:"checksum"`
	GapCount int    `json:"gap_count"`
}

type SyncResult struct {
	GeneratedAt time.Time       `json:"generated_at"`
	ArtifactDir string          `json:"artifact_dir"`
	Request     Request         `json:"request"`
	Datasets    []DatasetReport `json:"datasets"`
}

func NewHistoricalJob(cfg JobConfig) *HistoricalJob {
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	return &HistoricalJob{cfg: cfg}
}

func (job *HistoricalJob) Sync(ctx context.Context, request Request) (SyncResult, error) {
	if job.cfg.Fetcher == nil {
		return SyncResult{}, fmt.Errorf("historical fetcher is nil")
	}
	if job.cfg.Store == nil {
		return SyncResult{}, fmt.Errorf("historical store is nil")
	}
	if len(request.Symbols) == 0 {
		return SyncResult{}, fmt.Errorf("at least one symbol is required")
	}
	if request.Provider == "" {
		request.Provider = "bitget"
	}
	if request.ArtifactRoot == "" {
		request.ArtifactRoot = filepath.Join("artifacts", "platform", "historical-sync")
	}
	generatedAt := job.cfg.Now().UTC()
	artifactDir := filepath.Join(request.ArtifactRoot, generatedAt.Format("20060102T150405Z"))
	normalizedIntervals := normalizeIntervals(request.Intervals)
	result := SyncResult{
		GeneratedAt: generatedAt,
		ArtifactDir: artifactDir,
		Request: Request{
			Provider:     request.Provider,
			ProductType:  request.ProductType,
			Symbols:      append([]string(nil), request.Symbols...),
			Intervals:    append([]string(nil), normalizedIntervals...),
			Limit:        request.Limit,
			HorizonDays:  request.HorizonDays,
			ArtifactRoot: request.ArtifactRoot,
		},
		Datasets: make([]DatasetReport, 0, len(request.Symbols)*len(normalizedIntervals)),
	}
	for _, symbol := range request.Symbols {
		for _, interval := range normalizedIntervals {
			spec := config.DatasetConfig{
				Name:        datasetName(symbol, interval),
				Provider:    request.Provider,
				Symbol:      symbol,
				Interval:    interval,
				Limit:       request.Limit,
				ProductType: request.ProductType,
			}
			bars, err := job.fetchBars(ctx, spec, generatedAt, request.HorizonDays)
			if err != nil {
				return SyncResult{}, err
			}
			inserted, err := job.cfg.Store.UpsertBars(ctx, spec, bars)
			if err != nil {
				return SyncResult{}, err
			}
			rowCount, err := job.cfg.Store.CountBars(ctx, spec)
			if err != nil {
				return SyncResult{}, err
			}
			result.Datasets = append(result.Datasets, DatasetReport{
				Provider: request.Provider,
				Symbol:   symbol,
				Interval: interval,
				Inserted: inserted,
				RowCount: rowCount,
				Checksum: checksumBars(bars),
				GapCount: gapCount(bars, interval),
			})
		}
	}
	if err := writeArtifacts(result); err != nil {
		return SyncResult{}, err
	}
	return result, nil
}

func (job *HistoricalJob) fetchBars(ctx context.Context, spec config.DatasetConfig, generatedAt time.Time, horizonDays int) ([]core.Bar, error) {
	if horizonDays <= 0 {
		return job.cfg.Fetcher.FetchBars(ctx, spec)
	}
	horizonStart := generatedAt.AddDate(0, 0, -horizonDays)
	cursor := generatedAt
	seen := map[int64]bool{}
	bars := make([]core.Bar, 0, max(spec.Limit, 256))
	for {
		pageSpec := spec
		pageSpec.StartTime = horizonStart
		pageSpec.EndTime = cursor
		page, err := job.cfg.Fetcher.FetchBars(ctx, pageSpec)
		if err != nil {
			return nil, err
		}
		if len(page) == 0 {
			break
		}
		for _, bar := range page {
			if bar.Time.Before(horizonStart) {
				continue
			}
			ts := bar.Time.UTC().UnixMilli()
			if seen[ts] {
				continue
			}
			seen[ts] = true
			bars = append(bars, bar)
		}
		earliest := page[0].Time.UTC()
		if !earliest.After(horizonStart) {
			break
		}
		cursor = earliest.Add(-time.Millisecond)
	}
	sortBars(bars)
	return bars, nil
}

func writeArtifacts(result SyncResult) error {
	if err := os.MkdirAll(result.ArtifactDir, 0o755); err != nil {
		return err
	}
	body, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(result.ArtifactDir, "summary.json"), body, 0o644); err != nil {
		return err
	}
	type gapReport struct {
		Symbol   string `json:"symbol"`
		Interval string `json:"interval"`
		GapCount int    `json:"gap_count"`
	}
	gaps := make([]gapReport, 0, len(result.Datasets))
	for _, dataset := range result.Datasets {
		gaps = append(gaps, gapReport{Symbol: dataset.Symbol, Interval: dataset.Interval, GapCount: dataset.GapCount})
	}
	gapBody, err := json.MarshalIndent(gaps, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(result.ArtifactDir, "gap-report.json"), gapBody, 0o644)
}

func normalizeIntervals(intervals []string) []string {
	if len(intervals) == 0 {
		return append([]string(nil), DefaultIntervals...)
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

func datasetName(symbol, interval string) string {
	return strings.ToLower(strings.TrimSpace(symbol)) + "_" + strings.ToLower(strings.TrimSpace(interval))
}

func checksumBars(bars []core.Bar) string {
	body, _ := json.Marshal(bars)
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

func gapCount(bars []core.Bar, interval string) int {
	step, ok := intervalDuration(interval)
	if !ok || len(bars) < 2 {
		return 0
	}
	gaps := 0
	for index := 1; index < len(bars); index++ {
		if bars[index].Time.Sub(bars[index-1].Time) > step {
			gaps++
		}
	}
	return gaps
}

func intervalDuration(interval string) (time.Duration, bool) {
	switch strings.ToLower(strings.TrimSpace(interval)) {
	case "1m":
		return time.Minute, true
	case "5m":
		return 5 * time.Minute, true
	case "15m":
		return 15 * time.Minute, true
	case "1h":
		return time.Hour, true
	case "4h":
		return 4 * time.Hour, true
	case "1d":
		return 24 * time.Hour, true
	case "1w":
		return 7 * 24 * time.Hour, true
	default:
		return 0, false
	}
}

func sortBars(bars []core.Bar) {
	for i := 1; i < len(bars); i++ {
		for j := i; j > 0 && bars[j].Time.Before(bars[j-1].Time); j-- {
			bars[j], bars[j-1] = bars[j-1], bars[j]
		}
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
