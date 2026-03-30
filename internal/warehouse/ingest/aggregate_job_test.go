package ingest

import (
	"context"
	"testing"
	"time"

	"quantlab/internal/config"
	"quantlab/internal/core"
)

func TestAggregateBarsBuildsFiveMinuteBucket(t *testing.T) {
	base := time.Date(2026, 3, 29, 12, 0, 0, 0, time.UTC)
	bars := []core.Bar{
		{Time: base, Open: 1, High: 2, Low: 0.5, Close: 1.5, Volume: 1},
		{Time: base.Add(1 * time.Minute), Open: 1.5, High: 3, Low: 1.4, Close: 2.5, Volume: 2},
		{Time: base.Add(2 * time.Minute), Open: 2.5, High: 4, Low: 2, Close: 3.5, Volume: 3},
		{Time: base.Add(3 * time.Minute), Open: 3.5, High: 4.5, Low: 3, Close: 4, Volume: 4},
		{Time: base.Add(4 * time.Minute), Open: 4, High: 5, Low: 3.5, Close: 4.5, Volume: 5},
	}
	got, err := AggregateBars(bars, "1m", "5m")
	if err != nil {
		t.Fatalf("aggregate bars: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected one aggregate bar, got %d", len(got))
	}
	if got[0].Time != base || got[0].Open != 1 || got[0].High != 5 || got[0].Low != 0.5 || got[0].Close != 4.5 || got[0].Volume != 15 {
		t.Fatalf("unexpected aggregate bar: %+v", got[0])
	}
}

func TestAggregateBarsBuildsDailyBucketUsingBitgetSessionBoundary(t *testing.T) {
	base := time.Date(2026, 3, 29, 16, 0, 0, 0, time.UTC)
	bars := make([]core.Bar, 0, 96)
	for index := 0; index < 96; index++ {
		ts := base.Add(time.Duration(index) * 15 * time.Minute)
		bars = append(bars, core.Bar{
			Time:   ts,
			Open:   float64(index + 1),
			High:   float64(index + 2),
			Low:    float64(index),
			Close:  float64(index + 1),
			Volume: 1,
		})
	}
	got, err := AggregateBars(bars, "15m", "1d")
	if err != nil {
		t.Fatalf("aggregate bars: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected one aggregate bar, got %d", len(got))
	}
	if got[0].Time != base {
		t.Fatalf("expected bucket start %s, got %s", base, got[0].Time)
	}
}

func TestAggregateJobStoresDerivedIntervals(t *testing.T) {
	base := time.Date(2026, 3, 29, 0, 0, 0, 0, time.UTC)
	store := &aggregateStoreStub{
		bars: map[string][]core.Bar{
			"bitget:BTCUSDT:1m": {
				{Time: base, Open: 1, High: 2, Low: 0.5, Close: 1.5, Volume: 1},
				{Time: base.Add(1 * time.Minute), Open: 1.5, High: 3, Low: 1.4, Close: 2.5, Volume: 2},
				{Time: base.Add(2 * time.Minute), Open: 2.5, High: 4, Low: 2, Close: 3.5, Volume: 3},
				{Time: base.Add(3 * time.Minute), Open: 3.5, High: 4.5, Low: 3, Close: 4, Volume: 4},
				{Time: base.Add(4 * time.Minute), Open: 4, High: 5, Low: 3.5, Close: 4.5, Volume: 5},
			},
			"bitget:BTCUSDT:15m": {
				{Time: base, Open: 10, High: 12, Low: 9, Close: 11, Volume: 10},
				{Time: base.Add(15 * time.Minute), Open: 11, High: 13, Low: 10, Close: 12, Volume: 11},
				{Time: base.Add(30 * time.Minute), Open: 12, High: 14, Low: 11, Close: 13, Volume: 12},
				{Time: base.Add(45 * time.Minute), Open: 13, High: 15, Low: 12, Close: 14, Volume: 13},
				{Time: base.Add(1 * time.Hour), Open: 14, High: 16, Low: 13, Close: 15, Volume: 14},
				{Time: base.Add(75 * time.Minute), Open: 15, High: 17, Low: 14, Close: 16, Volume: 15},
				{Time: base.Add(90 * time.Minute), Open: 16, High: 18, Low: 15, Close: 17, Volume: 16},
				{Time: base.Add(105 * time.Minute), Open: 17, High: 19, Low: 16, Close: 18, Volume: 17},
				{Time: base.Add(2 * time.Hour), Open: 18, High: 20, Low: 17, Close: 19, Volume: 18},
				{Time: base.Add(135 * time.Minute), Open: 19, High: 21, Low: 18, Close: 20, Volume: 19},
				{Time: base.Add(150 * time.Minute), Open: 20, High: 22, Low: 19, Close: 21, Volume: 20},
				{Time: base.Add(165 * time.Minute), Open: 21, High: 23, Low: 20, Close: 22, Volume: 21},
				{Time: base.Add(3 * time.Hour), Open: 22, High: 24, Low: 21, Close: 23, Volume: 22},
				{Time: base.Add(195 * time.Minute), Open: 23, High: 25, Low: 22, Close: 24, Volume: 23},
				{Time: base.Add(210 * time.Minute), Open: 24, High: 26, Low: 23, Close: 25, Volume: 24},
				{Time: base.Add(225 * time.Minute), Open: 25, High: 27, Low: 24, Close: 26, Volume: 25},
			},
		},
	}
	job := NewAggregateJob(AggregateJobConfig{
		Store: store,
		Now:   func() time.Time { return time.Date(2026, 3, 29, 12, 45, 0, 0, time.UTC) },
	})
	result, err := job.Run(context.Background(), AggregateRequest{
		Provider:  "bitget",
		Symbols:   []string{"BTCUSDT"},
		Intervals: []string{"5m", "4h"},
	})
	if err != nil {
		t.Fatalf("run aggregate job: %v", err)
	}
	if len(result.Datasets) != 2 {
		t.Fatalf("unexpected aggregate datasets: %+v", result.Datasets)
	}
	if len(store.upserts) != 2 {
		t.Fatalf("unexpected upserts: %+v", store.upserts)
	}
	if store.upserts[0].spec.Interval != "5m" || store.upserts[1].spec.Interval != "4h" {
		t.Fatalf("unexpected target intervals: %+v", store.upserts)
	}
	if result.Datasets[1].SourceInterval != "15m" || result.Datasets[1].SourceRows != 16 {
		t.Fatalf("unexpected aggregate source: %+v", result.Datasets[1])
	}
}

func TestAggregateJobReplacesDerivedIntervalRows(t *testing.T) {
	base := time.Date(2026, 3, 29, 0, 0, 0, 0, time.UTC)
	store := &aggregateStoreStub{
		bars: map[string][]core.Bar{
			"bitget:BTCUSDT:15m": {
				{Time: base, Open: 10, High: 12, Low: 9, Close: 11, Volume: 10},
				{Time: base.Add(15 * time.Minute), Open: 11, High: 13, Low: 10, Close: 12, Volume: 11},
				{Time: base.Add(30 * time.Minute), Open: 12, High: 14, Low: 11, Close: 13, Volume: 12},
				{Time: base.Add(45 * time.Minute), Open: 13, High: 15, Low: 12, Close: 14, Volume: 13},
				{Time: base.Add(1 * time.Hour), Open: 14, High: 16, Low: 13, Close: 15, Volume: 14},
				{Time: base.Add(75 * time.Minute), Open: 15, High: 17, Low: 14, Close: 16, Volume: 15},
				{Time: base.Add(90 * time.Minute), Open: 16, High: 18, Low: 15, Close: 17, Volume: 16},
				{Time: base.Add(105 * time.Minute), Open: 17, High: 19, Low: 16, Close: 18, Volume: 17},
				{Time: base.Add(2 * time.Hour), Open: 18, High: 20, Low: 17, Close: 19, Volume: 18},
				{Time: base.Add(135 * time.Minute), Open: 19, High: 21, Low: 18, Close: 20, Volume: 19},
				{Time: base.Add(150 * time.Minute), Open: 20, High: 22, Low: 19, Close: 21, Volume: 20},
				{Time: base.Add(165 * time.Minute), Open: 21, High: 23, Low: 20, Close: 22, Volume: 21},
				{Time: base.Add(3 * time.Hour), Open: 22, High: 24, Low: 21, Close: 23, Volume: 22},
				{Time: base.Add(195 * time.Minute), Open: 23, High: 25, Low: 22, Close: 24, Volume: 23},
				{Time: base.Add(210 * time.Minute), Open: 24, High: 26, Low: 23, Close: 25, Volume: 24},
				{Time: base.Add(225 * time.Minute), Open: 25, High: 27, Low: 24, Close: 26, Volume: 25},
			},
			"bitget:BTCUSDT:4h": {
				{Time: base.Add(-4 * time.Hour), Open: 1, High: 2, Low: 0.5, Close: 1.5, Volume: 99},
			},
		},
	}
	job := NewAggregateJob(AggregateJobConfig{
		Store: store,
		Now:   func() time.Time { return time.Date(2026, 3, 29, 12, 45, 0, 0, time.UTC) },
	})
	result, err := job.Run(context.Background(), AggregateRequest{
		Provider:  "bitget",
		Symbols:   []string{"BTCUSDT"},
		Intervals: []string{"4h"},
	})
	if err != nil {
		t.Fatalf("run aggregate job: %v", err)
	}
	if len(result.Datasets) != 1 {
		t.Fatalf("unexpected aggregate datasets: %+v", result.Datasets)
	}
	if result.Datasets[0].RowCount != 1 {
		t.Fatalf("expected derived dataset to be replaced, got row_count=%d", result.Datasets[0].RowCount)
	}
	target := store.bars["bitget:BTCUSDT:4h"]
	if len(target) != 1 || target[0].Time != base {
		t.Fatalf("unexpected replaced target bars: %+v", target)
	}
}

type aggregateStoreStub struct {
	bars    map[string][]core.Bar
	upserts []aggregateUpsert
}

type aggregateUpsert struct {
	spec config.DatasetConfig
	bars []core.Bar
}

func (store *aggregateStoreStub) LoadBars(_ context.Context, spec config.DatasetConfig) ([]core.Bar, error) {
	key := spec.Provider + ":" + spec.Symbol + ":" + spec.Interval
	return append([]core.Bar(nil), store.bars[key]...), nil
}

func (store *aggregateStoreStub) UpsertBars(_ context.Context, spec config.DatasetConfig, bars []core.Bar) (int, error) {
	key := spec.Provider + ":" + spec.Symbol + ":" + spec.Interval
	store.bars[key] = append(append([]core.Bar(nil), store.bars[key]...), bars...)
	store.upserts = append(store.upserts, aggregateUpsert{spec: spec, bars: append([]core.Bar(nil), bars...)})
	return len(bars), nil
}

func (store *aggregateStoreStub) ReplaceBars(_ context.Context, spec config.DatasetConfig, bars []core.Bar) (int, error) {
	key := spec.Provider + ":" + spec.Symbol + ":" + spec.Interval
	store.bars[key] = append([]core.Bar(nil), bars...)
	store.upserts = append(store.upserts, aggregateUpsert{spec: spec, bars: append([]core.Bar(nil), bars...)})
	return len(bars), nil
}

func (store *aggregateStoreStub) CountBars(_ context.Context, spec config.DatasetConfig) (int, error) {
	key := spec.Provider + ":" + spec.Symbol + ":" + spec.Interval
	return len(store.bars[key]), nil
}
