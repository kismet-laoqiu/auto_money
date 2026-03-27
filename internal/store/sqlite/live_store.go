package sqlite

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"quantlab/internal/market"
	"quantlab/internal/trader"
)

//go:embed schema.sql
var schemaSQL string

type Store struct {
	db *sql.DB
}

type LiveStore interface {
	AppendEvent(ctx context.Context, evt market.MarketEvent, raw []byte) error
	SaveCheckpoint(ctx context.Context, shard string, state trader.EngineState) error
	LoadCheckpoint(ctx context.Context, shard string) (trader.EngineState, error)
}

func NewStore(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec("PRAGMA journal_mode=WAL;"); err != nil {
		db.Close()
		return nil, err
	}
	if _, err := db.Exec(schemaSQL); err != nil {
		db.Close()
		return nil, err
	}
	return &Store{db: db}, nil
}

func (store *Store) AppendEvent(ctx context.Context, evt market.MarketEvent, raw []byte) error {
	payload := raw
	if payload == nil {
		payload = []byte("null")
	}
	_, err := store.db.ExecContext(ctx, `
        INSERT INTO market_event_log (event_id, source, symbol, event_kind, exchange_ts, received_ts, payload_json)
        VALUES (?, ?, ?, ?, ?, ?, ?)
    `, evt.EventID(), "market", evt.Symbol(), evt.Kind(), evt.EventTime().UTC().Format(time.RFC3339Nano), time.Now().UTC().Format(time.RFC3339Nano), payload)
	return err
}

func (store *Store) SaveCheckpoint(ctx context.Context, shard string, state trader.EngineState) error {
	body, err := json.Marshal(state)
	if err != nil {
		return err
	}
	_, err = store.db.ExecContext(ctx, `
        INSERT INTO engine_checkpoint (shard_key, state_json, updated_at)
        VALUES (?, ?, ?)
        ON CONFLICT(shard_key) DO UPDATE SET state_json=excluded.state_json, updated_at=excluded.updated_at
    `, shard, body, time.Now().UTC().Format(time.RFC3339Nano))
	return err
}

func (store *Store) LoadCheckpoint(ctx context.Context, shard string) (trader.EngineState, error) {
	var body []byte
	err := store.db.QueryRowContext(ctx, `SELECT state_json FROM engine_checkpoint WHERE shard_key = ?`, shard).Scan(&body)
	if err != nil {
		if err == sql.ErrNoRows {
			return trader.EngineState{}, nil
		}
		return trader.EngineState{}, err
	}
	var state trader.EngineState
	if err := json.Unmarshal(body, &state); err != nil {
		return trader.EngineState{}, fmt.Errorf("decode checkpoint: %w", err)
	}
	return state, nil
}
