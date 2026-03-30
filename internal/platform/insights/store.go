package insights

import (
	"context"
	"database/sql"
	"fmt"

	"quantlab/internal/core"
)

type WarehouseStore interface {
	LoadRecentBars(ctx context.Context, provider, symbol, interval string, limit int) ([]core.Bar, error)
	LoadAnomalyThresholds(ctx context.Context, provider, symbol, interval string, lookbackBars int, moveQuantile, volumeQuantile float64) (AnomalyThresholds, error)
}

type SQLStore struct {
	db *sql.DB
}

func NewSQLStore(db *sql.DB) *SQLStore {
	return &SQLStore{db: db}
}

func (store *SQLStore) LoadRecentBars(ctx context.Context, provider, symbol, interval string, limit int) ([]core.Bar, error) {
	if store == nil || store.db == nil {
		return nil, fmt.Errorf("insights sql store db is nil")
	}
	if limit <= 0 {
		limit = 256
	}
	rows, err := store.db.QueryContext(ctx, `
		SELECT open_time, open, high, low, close, volume
		FROM (
			SELECT open_time, open, high, low, close, volume
			FROM market_bars
			WHERE provider = $1 AND symbol = $2 AND interval = $3
			ORDER BY open_time DESC
			LIMIT $4
		) bars
		ORDER BY open_time ASC
	`, provider, symbol, interval, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]core.Bar, 0, limit)
	for rows.Next() {
		var bar core.Bar
		if err := rows.Scan(&bar.Time, &bar.Open, &bar.High, &bar.Low, &bar.Close, &bar.Volume); err != nil {
			return nil, err
		}
		out = append(out, bar)
	}
	return out, rows.Err()
}

func (store *SQLStore) LoadAnomalyThresholds(ctx context.Context, provider, symbol, interval string, lookbackBars int, moveQuantile, volumeQuantile float64) (AnomalyThresholds, error) {
	if store == nil || store.db == nil {
		return AnomalyThresholds{}, fmt.Errorf("insights sql store db is nil")
	}
	if lookbackBars <= 0 {
		lookbackBars = 4096
	}
	var result AnomalyThresholds
	err := store.db.QueryRowContext(ctx, `
		WITH recent AS (
			SELECT open, close, volume
			FROM market_bars
			WHERE provider = $1 AND symbol = $2 AND interval = $3
			ORDER BY open_time DESC
			LIMIT $4
		)
		SELECT
			COALESCE(percentile_cont($5) WITHIN GROUP (ORDER BY ABS((close / NULLIF(open, 0) - 1) * 100.0)), 0),
			COALESCE(percentile_cont($6) WITHIN GROUP (ORDER BY volume), 0)
		FROM recent
	`, provider, symbol, interval, lookbackBars, moveQuantile, volumeQuantile).Scan(&result.MovePct, &result.Volume)
	return result, err
}
