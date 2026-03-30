package features

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"quantlab/internal/config"
	"quantlab/internal/warehouse/catalog"
)

func TestMaterializerWarehouseRoundTrip(t *testing.T) {
	if testing.Short() {
		t.Skip("skip warehouse integration in short mode")
	}
	warehouseConfig, err := catalog.LoadConfig(filepath.Join("..", "..", "..", "configs", "platform", "warehouse.yaml"))
	if err != nil {
		t.Fatalf("load warehouse config: %v", err)
	}
	db, err := catalog.Open(warehouseConfig)
	if err != nil {
		t.Fatalf("open warehouse db: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	strategyID := fmt.Sprintf("p05-materializer-integration-%d", time.Now().UnixNano())
	defer db.ExecContext(context.Background(), `DELETE FROM strategy_feature_snapshots WHERE strategy_id = $1`, strategyID)

	store := NewPostgresStore(db)
	request := Request{
		StrategyID: strategyID,
		Dataset: config.DatasetConfig{
			Name:     "mstrusdt_1h",
			Provider: "bitget",
			Symbol:   "MSTRUSDT",
			Interval: "1h",
		},
		Strategy: config.StrategyConfig{
			FastSMA:        15,
			SlowSMA:        60,
			PivotWindow:    3,
			LevelLookback:  90,
			LevelTolerance: 0.012,
			FibTolerance:   0.012,
			ATRWindow:      14,
		},
	}
	generatedAt := time.Date(2026, 3, 29, 13, 15, 0, 0, time.UTC)
	materializer := NewMaterializer(Config{Store: store, Now: func() time.Time { return generatedAt }})

	bars, err := store.LoadBars(ctx, request.Dataset)
	if err != nil {
		t.Fatalf("load bars: %v", err)
	}
	if len(bars) == 0 {
		t.Fatalf("expected warehouse bars for %+v", request.Dataset)
	}
	result, err := materializer.Materialize(ctx, request)
	if err != nil {
		t.Fatalf("materialize snapshots: %v", err)
	}
	if result.LoadedBars != len(bars) || result.Snapshots == 0 {
		t.Fatalf("unexpected materialize result: %+v loaded=%d", result, len(bars))
	}
	lastBar := bars[len(bars)-1]
	snapshot, err := materializer.FetchSnapshot(ctx, FetchSnapshotQuery{
		StrategyID:     strategyID,
		FeatureVersion: result.FeatureVersion,
		DatasetName:    request.Dataset.Name,
		BarTime:        lastBar.Time,
	})
	if err != nil {
		t.Fatalf("fetch snapshot: %v", err)
	}
	if snapshot.StrategyID != strategyID || snapshot.FeatureVersion != result.FeatureVersion {
		t.Fatalf("unexpected snapshot identity: %+v", snapshot)
	}
	if snapshot.DatasetName != request.Dataset.Name || snapshot.Symbol != request.Dataset.Symbol || snapshot.Interval != request.Dataset.Interval {
		t.Fatalf("unexpected snapshot metadata: %+v", snapshot)
	}
	if !snapshot.BarTime.Equal(lastBar.Time.UTC()) || !snapshot.GeneratedAt.Equal(generatedAt) {
		t.Fatalf("unexpected snapshot timing: %+v last_bar=%s", snapshot, lastBar.Time.UTC())
	}
}
