package ingest

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"quantlab/internal/config"
	"quantlab/internal/core"
)

func TestHistoricalJobSyncUsesDefaultIntervalsAndWritesArtifacts(t *testing.T) {
	fetcher := &fetcherStub{bars: sampleBars()}
	store := &storeStub{}
	job := NewHistoricalJob(JobConfig{
		Fetcher: fetcher,
		Store:   store,
		Now: func() time.Time {
			return time.Date(2026, 3, 29, 12, 0, 0, 0, time.UTC)
		},
	})

	result, err := job.Sync(context.Background(), Request{
		Provider:     "bitget",
		ProductType:  "USDT-FUTURES",
		Symbols:      []string{"MSTRUSDT"},
		ArtifactRoot: t.TempDir(),
		Limit:        3,
	})
	if err != nil {
		t.Fatalf("sync historical: %v", err)
	}
	if len(fetcher.specs) != len(DefaultIntervals) {
		t.Fatalf("unexpected fetch calls: %+v", fetcher.specs)
	}
	if len(result.Datasets) != len(DefaultIntervals) {
		t.Fatalf("unexpected dataset reports: %+v", result)
	}
	for _, report := range result.Datasets {
		if report.Checksum == "" || report.RowCount != 3 {
			t.Fatalf("unexpected dataset report: %+v", report)
		}
	}
	body, err := os.ReadFile(filepath.Join(result.ArtifactDir, "summary.json"))
	if err != nil {
		t.Fatalf("read summary artifact: %v", err)
	}
	var summary SyncResult
	if err := json.Unmarshal(body, &summary); err != nil {
		t.Fatalf("decode summary artifact: %v", err)
	}
	if summary.ArtifactDir != result.ArtifactDir || len(summary.Datasets) != len(DefaultIntervals) {
		t.Fatalf("unexpected summary artifact: %+v", summary)
	}
}

func TestHistoricalJobSyncCapturesRequestMetadata(t *testing.T) {
	fetcher := &fetcherStub{bars: sampleBars()}
	store := &storeStub{}
	job := NewHistoricalJob(JobConfig{
		Fetcher: fetcher,
		Store:   store,
		Now: func() time.Time {
			return time.Date(2026, 3, 29, 12, 30, 0, 0, time.UTC)
		},
	})

	result, err := job.Sync(context.Background(), Request{
		Provider:     "bitget",
		ProductType:  "USDT-FUTURES",
		Symbols:      []string{"BTCUSDT", "ETHUSDT"},
		Intervals:    []string{"1h", "4h"},
		Limit:        55,
		ArtifactRoot: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("sync historical: %v", err)
	}
	if result.Request.Provider != "bitget" || result.Request.ProductType != "USDT-FUTURES" {
		t.Fatalf("unexpected request metadata: %+v", result.Request)
	}
	if len(result.Request.Symbols) != 2 || result.Request.Symbols[0] != "BTCUSDT" || len(result.Request.Intervals) != 2 || result.Request.Limit != 55 {
		t.Fatalf("unexpected request metadata: %+v", result.Request)
	}
}

type fetcherStub struct {
	specs []config.DatasetConfig
	bars  []core.Bar
}

func (stub *fetcherStub) FetchBars(_ context.Context, spec config.DatasetConfig) ([]core.Bar, error) {
	stub.specs = append(stub.specs, spec)
	return append([]core.Bar(nil), stub.bars...), nil
}

type storeStub struct{}

func (store *storeStub) UpsertBars(_ context.Context, _ config.DatasetConfig, bars []core.Bar) (int, error) {
	return len(bars), nil
}

func (store *storeStub) CountBars(_ context.Context, _ config.DatasetConfig) (int, error) {
	return 3, nil
}

func sampleBars() []core.Bar {
	start := time.Date(2026, 3, 29, 0, 0, 0, 0, time.UTC)
	return []core.Bar{
		{Time: start, Open: 1, High: 2, Low: 0.5, Close: 1.5, Volume: 10},
		{Time: start.Add(time.Minute), Open: 1.5, High: 2.5, Low: 1, Close: 2, Volume: 12},
		{Time: start.Add(2 * time.Minute), Open: 2, High: 3, Low: 1.5, Close: 2.5, Volume: 15},
	}
}
