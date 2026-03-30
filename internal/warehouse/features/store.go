package features

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"quantlab/internal/config"
	"quantlab/internal/core"
	"quantlab/internal/warehouse/ingest"
)

type PostgresStore struct {
	db   *sql.DB
	bars *ingest.PostgresStore
}

func NewPostgresStore(db *sql.DB) *PostgresStore {
	return &PostgresStore{db: db, bars: ingest.NewPostgresStore(db)}
}

func (store *PostgresStore) LoadBars(ctx context.Context, spec config.DatasetConfig) ([]core.Bar, error) {
	if store.bars == nil {
		return nil, fmt.Errorf("postgres store db is nil")
	}
	return store.bars.LoadBars(ctx, spec)
}

func (store *PostgresStore) UpsertSnapshots(ctx context.Context, snapshots []Snapshot) (int, error) {
	if store.db == nil {
		return 0, fmt.Errorf("postgres store db is nil")
	}
	if len(snapshots) == 0 {
		return 0, nil
	}
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
        INSERT INTO strategy_feature_snapshots (
            strategy_id, feature_version, dataset_name, symbol, interval, bar_time, features_json, theory_fixture, generated_at
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
        ON CONFLICT (strategy_id, feature_version, dataset_name, bar_time) DO UPDATE SET
            symbol = EXCLUDED.symbol,
            interval = EXCLUDED.interval,
            features_json = EXCLUDED.features_json,
            theory_fixture = EXCLUDED.theory_fixture,
            generated_at = EXCLUDED.generated_at
    `)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	inserted := 0
	for _, snapshot := range snapshots {
		body, err := json.Marshal(snapshot.Features)
		if err != nil {
			return 0, err
		}
		result, err := stmt.ExecContext(ctx,
			snapshot.StrategyID,
			snapshot.FeatureVersion,
			snapshot.DatasetName,
			snapshot.Symbol,
			snapshot.Interval,
			snapshot.BarTime.UTC(),
			body,
			snapshot.TheoryFixture,
			snapshot.GeneratedAt.UTC(),
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
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return inserted, nil
}

func (store *PostgresStore) FetchSnapshot(ctx context.Context, query FetchSnapshotQuery) (Snapshot, error) {
	if store.db == nil {
		return Snapshot{}, fmt.Errorf("postgres store db is nil")
	}
	var snapshot Snapshot
	var featuresJSON []byte
	if err := store.db.QueryRowContext(ctx, `
        SELECT strategy_id, feature_version, dataset_name, symbol, interval, bar_time, features_json, theory_fixture, generated_at
        FROM strategy_feature_snapshots
        WHERE strategy_id = $1 AND feature_version = $2 AND dataset_name = $3 AND bar_time = $4
    `, query.StrategyID, query.FeatureVersion, query.DatasetName, query.BarTime.UTC()).Scan(
		&snapshot.StrategyID,
		&snapshot.FeatureVersion,
		&snapshot.DatasetName,
		&snapshot.Symbol,
		&snapshot.Interval,
		&snapshot.BarTime,
		&featuresJSON,
		&snapshot.TheoryFixture,
		&snapshot.GeneratedAt,
	); err != nil {
		return Snapshot{}, err
	}
	if err := json.Unmarshal(featuresJSON, &snapshot.Features); err != nil {
		return Snapshot{}, err
	}
	return snapshot, nil
}
