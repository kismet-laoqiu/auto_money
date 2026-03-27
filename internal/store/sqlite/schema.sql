CREATE TABLE IF NOT EXISTS market_event_log (
  event_id TEXT PRIMARY KEY,
  source TEXT NOT NULL,
  symbol TEXT NOT NULL,
  event_kind TEXT NOT NULL,
  exchange_ts TEXT NOT NULL,
  received_ts TEXT NOT NULL,
  payload_json BLOB NOT NULL
);

CREATE TABLE IF NOT EXISTS engine_checkpoint (
  shard_key TEXT PRIMARY KEY,
  state_json BLOB NOT NULL,
  updated_at TEXT NOT NULL
);
