CREATE EXTENSION IF NOT EXISTS timescaledb;

CREATE TABLE IF NOT EXISTS market_bars (
    provider TEXT NOT NULL,
    symbol TEXT NOT NULL,
    interval TEXT NOT NULL,
    open_time TIMESTAMPTZ NOT NULL,
    close_time TIMESTAMPTZ NOT NULL,
    open DOUBLE PRECISION NOT NULL,
    high DOUBLE PRECISION NOT NULL,
    low DOUBLE PRECISION NOT NULL,
    close DOUBLE PRECISION NOT NULL,
    volume DOUBLE PRECISION NOT NULL,
    trade_count BIGINT NOT NULL DEFAULT 0,
    source_checksum TEXT NOT NULL DEFAULT '',
    ingested_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (provider, symbol, interval, open_time)
);

SELECT create_hypertable('market_bars', 'open_time', if_not_exists => TRUE, migrate_data => TRUE);

CREATE INDEX IF NOT EXISTS market_bars_symbol_interval_time_idx
    ON market_bars (symbol, interval, open_time DESC);
