package sqlite

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"quantlab/internal/trader"
)

//go:embed schema.sql
var schemaSQL string

type Store struct {
	db *sql.DB
}

type LogEvent interface {
	EventID() string
	Symbol() string
	EventTime() time.Time
	Kind() string
}

type EventEnvelope struct {
	Seq        int64
	Source     string
	EventID    string
	Symbol     string
	Kind       string
	ExchangeTS time.Time
	ReceivedTS time.Time
	Payload    json.RawMessage
}

type LiveStore interface {
	AppendEvent(ctx context.Context, source string, evt LogEvent, raw []byte) (int64, error)
	ListEventsAfter(ctx context.Context, afterSeq int64, limit int, sources ...string) ([]EventEnvelope, error)
	SaveConsumerCursor(ctx context.Context, consumer string, seq int64) error
	LoadConsumerCursor(ctx context.Context, consumer string) (int64, error)
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

func (store *Store) AppendEvent(ctx context.Context, source string, evt LogEvent, raw []byte) (int64, error) {
	payload := raw
	if payload == nil {
		payload = []byte("null")
	}
	result, err := store.db.ExecContext(ctx, `
        INSERT INTO event_log (source, event_id, symbol, event_kind, exchange_ts, received_ts, payload_json)
        VALUES (?, ?, ?, ?, ?, ?, ?)
    `, source, evt.EventID(), evt.Symbol(), evt.Kind(), evt.EventTime().UTC().Format(time.RFC3339Nano), time.Now().UTC().Format(time.RFC3339Nano), payload)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (store *Store) ListEventsAfter(ctx context.Context, afterSeq int64, limit int, sources ...string) ([]EventEnvelope, error) {
	if limit <= 0 {
		limit = 256
	}
	args := []any{afterSeq}
	query := `
        SELECT seq, source, event_id, symbol, event_kind, exchange_ts, received_ts, payload_json
        FROM event_log
        WHERE seq > ?
    `
	if len(sources) > 0 {
		placeholders := make([]string, 0, len(sources))
		for _, source := range sources {
			placeholders = append(placeholders, "?")
			args = append(args, source)
		}
		query += " AND source IN (" + strings.Join(placeholders, ", ") + ")"
	}
	query += " ORDER BY seq ASC LIMIT ?"
	args = append(args, limit)

	rows, err := store.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := make([]EventEnvelope, 0, limit)
	for rows.Next() {
		var env EventEnvelope
		var exchangeTS string
		var receivedTS string
		if err := rows.Scan(&env.Seq, &env.Source, &env.EventID, &env.Symbol, &env.Kind, &exchangeTS, &receivedTS, &env.Payload); err != nil {
			return nil, err
		}
		env.ExchangeTS, err = time.Parse(time.RFC3339Nano, exchangeTS)
		if err != nil {
			return nil, fmt.Errorf("parse exchange ts: %w", err)
		}
		env.ReceivedTS, err = time.Parse(time.RFC3339Nano, receivedTS)
		if err != nil {
			return nil, fmt.Errorf("parse received ts: %w", err)
		}
		events = append(events, env)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return events, nil
}

func (store *Store) SaveConsumerCursor(ctx context.Context, consumer string, seq int64) error {
	_, err := store.db.ExecContext(ctx, `
        INSERT INTO consumer_cursor (consumer_key, last_seq, updated_at)
        VALUES (?, ?, ?)
        ON CONFLICT(consumer_key) DO UPDATE SET last_seq=excluded.last_seq, updated_at=excluded.updated_at
    `, consumer, seq, time.Now().UTC().Format(time.RFC3339Nano))
	return err
}

func (store *Store) LoadConsumerCursor(ctx context.Context, consumer string) (int64, error) {
	var seq int64
	err := store.db.QueryRowContext(ctx, `SELECT last_seq FROM consumer_cursor WHERE consumer_key = ?`, consumer).Scan(&seq)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, err
	}
	return seq, nil
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
