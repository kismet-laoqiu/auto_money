CREATE TABLE IF NOT EXISTS event_log (
  seq INTEGER PRIMARY KEY AUTOINCREMENT,
  source TEXT NOT NULL,
  event_id TEXT NOT NULL UNIQUE,
  symbol TEXT NOT NULL,
  event_kind TEXT NOT NULL,
  exchange_ts TEXT NOT NULL,
  received_ts TEXT NOT NULL,
  payload_json BLOB NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_event_log_source_seq ON event_log(source, seq);
CREATE INDEX IF NOT EXISTS idx_event_log_kind_seq ON event_log(event_kind, seq);

CREATE TABLE IF NOT EXISTS engine_checkpoint (
  shard_key TEXT PRIMARY KEY,
  state_json BLOB NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS consumer_cursor (
  consumer_key TEXT PRIMARY KEY,
  last_seq INTEGER NOT NULL,
  updated_at TEXT NOT NULL
);
