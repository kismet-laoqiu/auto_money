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

type ConsumerCursorRow struct {
	ConsumerKey string    `json:"consumer_key"`
	LastSeq     int64     `json:"last_seq"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type PositionRow struct {
	Seq        int64     `json:"seq"`
	Source     string    `json:"source"`
	EventID    string    `json:"event_id"`
	Symbol     string    `json:"symbol"`
	Kind       string    `json:"kind"`
	ExchangeTS time.Time `json:"exchange_ts"`
	Qty        float64   `json:"qty"`
}

type OrderRow struct {
	Seq        int64     `json:"seq"`
	Source     string    `json:"source"`
	EventID    string    `json:"event_id"`
	Symbol     string    `json:"symbol"`
	Kind       string    `json:"kind"`
	ExchangeTS time.Time `json:"exchange_ts"`
	OrderID    string    `json:"order_id"`
	ClientOID  string    `json:"client_oid"`
	Status     string    `json:"status"`
	Size       float64   `json:"size"`
	Price      float64   `json:"price"`
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
		var payload any
		if err := rows.Scan(&env.Seq, &env.Source, &env.EventID, &env.Symbol, &env.Kind, &exchangeTS, &receivedTS, &payload); err != nil {
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
		env.Payload, err = normalizeJSONPayload(payload)
		if err != nil {
			return nil, err
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

func (store *Store) LastSeq(ctx context.Context) (int64, error) {
	var seq sql.NullInt64
	if err := store.db.QueryRowContext(ctx, `SELECT MAX(seq) FROM event_log`).Scan(&seq); err != nil {
		return 0, err
	}
	if !seq.Valid {
		return 0, nil
	}
	return seq.Int64, nil
}

func (store *Store) ListConsumerCursors(ctx context.Context) ([]ConsumerCursorRow, error) {
	rows, err := store.db.QueryContext(ctx, `
		SELECT consumer_key, last_seq, updated_at
		FROM consumer_cursor
		ORDER BY consumer_key ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]ConsumerCursorRow, 0)
	for rows.Next() {
		var row ConsumerCursorRow
		var updatedAt string
		if err := rows.Scan(&row.ConsumerKey, &row.LastSeq, &updatedAt); err != nil {
			return nil, err
		}
		row.UpdatedAt, err = time.Parse(time.RFC3339Nano, updatedAt)
		if err != nil {
			return nil, fmt.Errorf("parse cursor updated_at: %w", err)
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (store *Store) ListRecentEvents(ctx context.Context, limit int) ([]EventEnvelope, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := store.db.QueryContext(ctx, `
		SELECT seq, source, event_id, symbol, event_kind, exchange_ts, received_ts, payload_json
		FROM event_log
		ORDER BY seq DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanEventEnvelopes(rows)
}

func (store *Store) ListLatestPositions(ctx context.Context) ([]PositionRow, error) {
	rows, err := store.db.QueryContext(ctx, `
		SELECT e.seq, e.source, e.event_id, e.symbol, e.event_kind, e.exchange_ts, e.payload_json
		FROM event_log e
		INNER JOIN (
			SELECT symbol, MAX(seq) AS max_seq
			FROM event_log
			WHERE event_kind IN ('position_snapshot', 'position_update')
			GROUP BY symbol
		) latest ON latest.symbol = e.symbol AND latest.max_seq = e.seq
		ORDER BY e.symbol ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]PositionRow, 0)
	for rows.Next() {
		var row PositionRow
		var exchangeTS string
		var payload any
		if err := rows.Scan(&row.Seq, &row.Source, &row.EventID, &row.Symbol, &row.Kind, &exchangeTS, &payload); err != nil {
			return nil, err
		}
		row.ExchangeTS, err = time.Parse(time.RFC3339Nano, exchangeTS)
		if err != nil {
			return nil, fmt.Errorf("parse position exchange ts: %w", err)
		}
		var event struct {
			Qty float64 `json:"Qty"`
			Alt float64 `json:"qty"`
		}
		body, err := normalizeJSONPayload(payload)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(body, &event); err != nil {
			return nil, fmt.Errorf("decode position payload: %w", err)
		}
		row.Qty = event.Qty
		if row.Qty == 0 {
			row.Qty = event.Alt
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (store *Store) ListLatestOrders(ctx context.Context, limit int) ([]OrderRow, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := store.db.QueryContext(ctx, `
		SELECT seq, source, event_id, symbol, event_kind, exchange_ts, payload_json
		FROM event_log
		WHERE event_kind IN ('order_update', 'order_fill')
		ORDER BY seq DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]OrderRow, 0, limit)
	for rows.Next() {
		var row OrderRow
		var exchangeTS string
		var payload any
		if err := rows.Scan(&row.Seq, &row.Source, &row.EventID, &row.Symbol, &row.Kind, &exchangeTS, &payload); err != nil {
			return nil, err
		}
		row.ExchangeTS, err = time.Parse(time.RFC3339Nano, exchangeTS)
		if err != nil {
			return nil, fmt.Errorf("parse order exchange ts: %w", err)
		}
		var event struct {
			OrderID      string  `json:"OrderID"`
			OrderIDAlt   string  `json:"orderId"`
			ClientOID    string  `json:"ClientOID"`
			ClientOIDAlt string  `json:"clientOid"`
			Status       string  `json:"Status"`
			StatusAlt    string  `json:"status"`
			Size         float64 `json:"Size"`
			SizeAlt      float64 `json:"size"`
			Price        float64 `json:"Price"`
			PriceAlt     float64 `json:"price"`
		}
		body, err := normalizeJSONPayload(payload)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(body, &event); err != nil {
			return nil, fmt.Errorf("decode order payload: %w", err)
		}
		row.OrderID = firstNonEmpty(event.OrderID, event.OrderIDAlt)
		row.ClientOID = firstNonEmpty(event.ClientOID, event.ClientOIDAlt)
		row.Status = firstNonEmpty(event.Status, event.StatusAlt)
		row.Size = firstNonZero(event.Size, event.SizeAlt)
		row.Price = firstNonZero(event.Price, event.PriceAlt)
		out = append(out, row)
	}
	return out, rows.Err()
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

func scanEventEnvelopes(rows *sql.Rows) ([]EventEnvelope, error) {
	out := make([]EventEnvelope, 0)
	for rows.Next() {
		var env EventEnvelope
		var exchangeTS string
		var receivedTS string
		var payload any
		if err := rows.Scan(&env.Seq, &env.Source, &env.EventID, &env.Symbol, &env.Kind, &exchangeTS, &receivedTS, &payload); err != nil {
			return nil, err
		}
		var err error
		env.ExchangeTS, err = time.Parse(time.RFC3339Nano, exchangeTS)
		if err != nil {
			return nil, fmt.Errorf("parse exchange ts: %w", err)
		}
		env.ReceivedTS, err = time.Parse(time.RFC3339Nano, receivedTS)
		if err != nil {
			return nil, fmt.Errorf("parse received ts: %w", err)
		}
		env.Payload, err = normalizeJSONPayload(payload)
		if err != nil {
			return nil, err
		}
		out = append(out, env)
	}
	return out, rows.Err()
}

func normalizeJSONPayload(value any) (json.RawMessage, error) {
	switch payload := value.(type) {
	case []byte:
		return append(json.RawMessage(nil), payload...), nil
	case string:
		return json.RawMessage(payload), nil
	default:
		return nil, fmt.Errorf("unsupported payload type %T", value)
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func firstNonZero(values ...float64) float64 {
	for _, value := range values {
		if value != 0 {
			return value
		}
	}
	return 0
}
