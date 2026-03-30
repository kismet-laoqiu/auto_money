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
			"bitget:BTCUSDT:1h": {
				{Time: base, Open: 10, High: 12, Low: 9, Close: 11, Volume: 10},
				{Time: base.Add(1 * time.Hour), Open: 11, High: 13, Low: 10, Close: 12, Volume: 11},
				{Time: base.Add(2 * time.Hour), Open: 12, High: 14, Low: 11, Close: 13, Volume: 12},
				{Time: base.Add(3 * time.Hour), Open: 13, High: 15, Low: 12, Close: 14, Volume: 13},
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
	store.upserts = append(store.upserts, aggregateUpsert{spec: spec, bars: append([]core.Bar(nil), bars...)})
	return len(bars), nil
}

func (store *aggregateStoreStub) CountBars(_ context.Context, spec config.DatasetConfig) (int, error) {
	for _, upsert := range store.upserts {
		if upsert.spec.Provider == spec.Provider && upsert.spec.Symbol == spec.Symbol && upsert.spec.Interval == spec.Interval {
			return len(upsert.bars), nil
		}
	}
	return 0, nil
}
