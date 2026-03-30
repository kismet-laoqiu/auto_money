CREATE TABLE IF NOT EXISTS backtest_runs (
    run_id TEXT PRIMARY KEY,
    strategy_id TEXT NOT NULL,
    bundle_version TEXT NOT NULL DEFAULT '',
    feature_version TEXT NOT NULL,
    config_path TEXT NOT NULL,
    dataset_hash TEXT NOT NULL,
    metric_name TEXT NOT NULL,
    objective_score DOUBLE PRECISION NOT NULL,
    final_score DOUBLE PRECISION NOT NULL,
    artifact_dir TEXT NOT NULL,
    generated_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS backtest_run_datasets (
    run_id TEXT NOT NULL REFERENCES backtest_runs(run_id) ON DELETE CASCADE,
    dataset_name TEXT NOT NULL,
    provider TEXT NOT NULL,
    symbol TEXT NOT NULL,
    interval TEXT NOT NULL,
    bars INTEGER NOT NULL,
    dataset_hash TEXT NOT NULL,
    PRIMARY KEY (run_id, dataset_name)
);
