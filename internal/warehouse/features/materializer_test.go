package features

import (
	"context"
	"reflect"
	"testing"
	"time"

	"quantlab/internal/config"
	"quantlab/internal/core"
)

type fakeStore struct {
	bars       []core.Bar
	snapshots  []Snapshot
	fetch      Snapshot
	fetchQuery FetchSnapshotQuery
	loadSpec   config.DatasetConfig
}

func (store *fakeStore) LoadBars(_ context.Context, spec config.DatasetConfig) ([]core.Bar, error) {
	store.loadSpec = spec
	return append([]core.Bar(nil), store.bars...), nil
}

func (store *fakeStore) UpsertSnapshots(_ context.Context, snapshots []Snapshot) (int, error) {
	store.snapshots = append([]Snapshot(nil), snapshots...)
	return len(snapshots), nil
}

func (store *fakeStore) FetchSnapshot(_ context.Context, query FetchSnapshotQuery) (Snapshot, error) {
	store.fetchQuery = query
	return store.fetch, nil
}

func TestMaterializerMaterializeBuildsSnapshotsFromLoadedBars(t *testing.T) {
	bars := []core.Bar{
		{Time: time.Date(2026, 3, 29, 0, 0, 0, 0, time.UTC), Open: 100, High: 102, Low: 99, Close: 101, Volume: 10},
		{Time: time.Date(2026, 3, 29, 1, 0, 0, 0, time.UTC), Open: 101, High: 103, Low: 100, Close: 102, Volume: 11},
		{Time: time.Date(2026, 3, 29, 2, 0, 0, 0, time.UTC), Open: 102, High: 105, Low: 101, Close: 104, Volume: 12},
		{Time: time.Date(2026, 3, 29, 3, 0, 0, 0, time.UTC), Open: 104, High: 106, Low: 103, Close: 105, Volume: 13},
	}
	store := &fakeStore{bars: bars}
	now := time.Date(2026, 3, 29, 12, 0, 0, 0, time.UTC)
	materializer := NewMaterializer(Config{Store: store, Now: func() time.Time { return now }})
	request := Request{
		StrategyID:    "mstr-wave-fib",
		Dataset:       config.DatasetConfig{Name: "mstrusdt_1h", Provider: "bitget", Symbol: "MSTRUSDT", Interval: "1h"},
		Strategy:      config.StrategyConfig{FastSMA: 2, SlowSMA: 3, PivotWindow: 1, LevelLookback: 3, LevelTolerance: 0.01, FibTolerance: 0.01, ATRWindow: 2},
		TheoryFixture: "wave_structure:demo",
	}
	result, err := materializer.Materialize(context.Background(), request)
	if err != nil {
		t.Fatalf("materialize dataset: %v", err)
	}
	if store.loadSpec != request.Dataset {
		t.Fatalf("unexpected load spec: %+v", store.loadSpec)
	}
	if result.FeatureVersion != DefaultVersion().String() {
		t.Fatalf("unexpected feature version: %+v", result)
	}
	if result.Snapshots != len(bars) {
		t.Fatalf("unexpected result: %+v", result)
	}
	if len(store.snapshots) != len(bars) {
		t.Fatalf("unexpected stored snapshots: %d", len(store.snapshots))
	}
	for index, snapshot := range store.snapshots {
		if snapshot.StrategyID != request.StrategyID || snapshot.DatasetName != request.Dataset.Name || snapshot.Symbol != request.Dataset.Symbol || snapshot.Interval != request.Dataset.Interval {
			t.Fatalf("unexpected snapshot metadata: %+v", snapshot)
		}
		if !snapshot.BarTime.Equal(bars[index].Time) || !snapshot.GeneratedAt.Equal(now) {
			t.Fatalf("unexpected snapshot timing: %+v", snapshot)
		}
		if snapshot.TheoryFixture != request.TheoryFixture {
			t.Fatalf("unexpected theory fixture: %+v", snapshot)
		}
		wantFeatures := core.ExtractFeatureSet(bars, index, request.Strategy)
		if !reflect.DeepEqual(snapshot.Features, wantFeatures) {
			t.Fatalf("unexpected snapshot features at index %d\nwant=%+v\ngot=%+v", index, wantFeatures, snapshot.Features)
		}
	}
}

func TestMaterializerFetchSnapshotUsesDefaultVersion(t *testing.T) {
	want := Snapshot{
		StrategyID:     "mstr-wave-fib",
		FeatureVersion: DefaultVersion().String(),
		DatasetName:    "mstrusdt_1h",
		Symbol:         "MSTRUSDT",
		Interval:       "1h",
		BarTime:        time.Date(2026, 3, 29, 3, 0, 0, 0, time.UTC),
		GeneratedAt:    time.Date(2026, 3, 29, 12, 0, 0, 0, time.UTC),
	}
	store := &fakeStore{fetch: want}
	materializer := NewMaterializer(Config{Store: store})
	got, err := materializer.FetchSnapshot(context.Background(), FetchSnapshotQuery{
		StrategyID:  "mstr-wave-fib",
		DatasetName: "mstrusdt_1h",
		BarTime:     want.BarTime,
	})
	if err != nil {
		t.Fatalf("fetch snapshot: %v", err)
	}
	if store.fetchQuery.FeatureVersion != DefaultVersion().String() {
		t.Fatalf("unexpected fetch query: %+v", store.fetchQuery)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected snapshot: %+v", got)
	}
}
