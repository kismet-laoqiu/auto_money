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

func TestDefaultIntervalsIncludeWeekly(t *testing.T) {
	found := false
	for _, interval := range DefaultIntervals {
		if interval == "1w" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected 1w in default intervals")
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

func TestHistoricalJobSyncReusesEarliestBarAsNextPageCursor(t *testing.T) {
	pageOne := []core.Bar{
		{Time: time.Date(2025, 9, 29, 0, 0, 0, 0, time.UTC), Close: 1},
		{Time: time.Date(2026, 3, 28, 0, 0, 0, 0, time.UTC), Close: 2},
	}
	pageTwo := []core.Bar{
		{Time: time.Date(2025, 3, 29, 0, 0, 0, 0, time.UTC), Close: 3},
		{Time: time.Date(2025, 6, 29, 0, 0, 0, 0, time.UTC), Close: 4},
	}
	fetcher := &pagedFetcherStub{pages: [][]core.Bar{pageOne, pageTwo}}
	store := &recordingStoreStub{}
	job := NewHistoricalJob(JobConfig{
		Fetcher: fetcher,
		Store:   store,
		Now: func() time.Time {
			return time.Date(2026, 3, 29, 0, 0, 0, 0, time.UTC)
		},
	})

	result, err := job.Sync(context.Background(), Request{
		Provider:     "bitget",
		ProductType:  "USDT-FUTURES",
		Symbols:      []string{"BTCUSDT"},
		Intervals:    []string{"1d"},
		Limit:        2,
		HorizonDays:  365,
		ArtifactRoot: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("sync historical: %v", err)
	}
	if len(fetcher.specs) != 2 {
		t.Fatalf("unexpected fetch count: %d", len(fetcher.specs))
	}
	if !fetcher.specs[0].StartTime.Equal(time.Date(2025, 3, 29, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("unexpected start time: %s", fetcher.specs[0].StartTime)
	}
	if !fetcher.specs[1].EndTime.Equal(pageOne[0].Time) {
		t.Fatalf("expected second page cursor to reuse first page earliest, got %s", fetcher.specs[1].EndTime)
	}
	if len(store.lastBars) != 4 {
		t.Fatalf("unexpected stored bars: %+v", store.lastBars)
	}
	if store.lastBars[0].Time != pageTwo[0].Time || store.lastBars[1].Time != pageTwo[1].Time || store.lastBars[2].Time != pageOne[0].Time || store.lastBars[3].Time != pageOne[1].Time {
		t.Fatalf("unexpected stored bars order: %+v", store.lastBars)
	}
	if len(result.Datasets) != 1 || result.Datasets[0].RowCount != 4 {
		t.Fatalf("unexpected result: %+v", result)
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

type pagedFetcherStub struct {
	specs []config.DatasetConfig
	pages [][]core.Bar
}

func (stub *pagedFetcherStub) FetchBars(_ context.Context, spec config.DatasetConfig) ([]core.Bar, error) {
	stub.specs = append(stub.specs, spec)
	if len(stub.pages) == 0 {
		return nil, nil
	}
	page := stub.pages[0]
	stub.pages = stub.pages[1:]
	return append([]core.Bar(nil), page...), nil
}

type recordingStoreStub struct {
	lastBars []core.Bar
}

func (store *recordingStoreStub) UpsertBars(_ context.Context, _ config.DatasetConfig, bars []core.Bar) (int, error) {
	store.lastBars = append([]core.Bar(nil), bars...)
	return len(bars), nil
}

func (store *recordingStoreStub) CountBars(_ context.Context, _ config.DatasetConfig) (int, error) {
	return len(store.lastBars), nil
}
