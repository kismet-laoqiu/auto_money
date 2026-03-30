package ingest

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"quantlab/internal/config"
	"quantlab/internal/core"
)

type PostgresStore struct {
	db *sql.DB
}

func NewPostgresStore(db *sql.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

func (store *PostgresStore) UpsertBars(ctx context.Context, spec config.DatasetConfig, bars []core.Bar) (int, error) {
	if store.db == nil {
		return 0, fmt.Errorf("postgres store db is nil")
	}
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	inserted, err := store.upsertBarsTx(ctx, tx, spec, bars)
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return inserted, nil
}

func (store *PostgresStore) ReplaceBars(ctx context.Context, spec config.DatasetConfig, bars []core.Bar) (int, error) {
	if store.db == nil {
		return 0, fmt.Errorf("postgres store db is nil")
	}
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `
        DELETE FROM market_bars
        WHERE provider = $1 AND symbol = $2 AND interval = $3
    `, spec.Provider, spec.Symbol, spec.Interval); err != nil {
		return 0, err
	}
	inserted, err := store.upsertBarsTx(ctx, tx, spec, bars)
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return inserted, nil
}

func (store *PostgresStore) upsertBarsTx(ctx context.Context, tx *sql.Tx, spec config.DatasetConfig, bars []core.Bar) (int, error) {
	stmt, err := tx.PrepareContext(ctx, `
        INSERT INTO market_bars (
            provider, symbol, interval, open_time, close_time, open, high, low, close, volume, trade_count
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
        ON CONFLICT (provider, symbol, interval, open_time) DO NOTHING
    `)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()
	step, ok := intervalDuration(spec.Interval)
	if !ok {
		step = time.Minute
	}
	inserted := 0
	for _, bar := range bars {
		result, err := stmt.ExecContext(ctx,
			spec.Provider,
			spec.Symbol,
			spec.Interval,
			bar.Time.UTC(),
			bar.Time.UTC().Add(step),
			bar.Open,
			bar.High,
			bar.Low,
			bar.Close,
			bar.Volume,
			0,
		)
		if err != nil {
			return 0, err
		}
		rows, err := result.RowsAffected()
		if err != nil {
			return 0, err
		}
		inserted += int(rows)
	}
	return inserted, nil
}

func (store *PostgresStore) CountBars(ctx context.Context, spec config.DatasetConfig) (int, error) {
	if store.db == nil {
		return 0, fmt.Errorf("postgres store db is nil")
	}
	var count int
	if err := store.db.QueryRowContext(ctx, `
        SELECT COUNT(*)
        FROM market_bars
        WHERE provider = $1 AND symbol = $2 AND interval = $3
    `, spec.Provider, spec.Symbol, spec.Interval).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (store *PostgresStore) LoadBars(ctx context.Context, spec config.DatasetConfig) ([]core.Bar, error) {
	if store.db == nil {
		return nil, fmt.Errorf("postgres store db is nil")
	}
	rows, err := store.db.QueryContext(ctx, `
        SELECT open_time, open, high, low, close, volume
        FROM market_bars
        WHERE provider = $1 AND symbol = $2 AND interval = $3
        ORDER BY open_time ASC
    `, spec.Provider, spec.Symbol, spec.Interval)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	bars := make([]core.Bar, 0, 256)
	for rows.Next() {
		var bar core.Bar
		if err := rows.Scan(&bar.Time, &bar.Open, &bar.High, &bar.Low, &bar.Close, &bar.Volume); err != nil {
			return nil, err
		}
		bars = append(bars, bar)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return bars, nil
}
