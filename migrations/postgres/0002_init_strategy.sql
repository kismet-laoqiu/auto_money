CREATE TABLE IF NOT EXISTS strategy_registry (
    strategy_id TEXT NOT NULL,
    version TEXT NOT NULL,
    bundle_path TEXT NOT NULL,
    score_expr TEXT NOT NULL DEFAULT '',
    risk_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (strategy_id, version)
);

CREATE TABLE IF NOT EXISTS strategy_feature_snapshots (
    strategy_id TEXT NOT NULL,
    feature_version TEXT NOT NULL,
    dataset_name TEXT NOT NULL,
    symbol TEXT NOT NULL,
    interval TEXT NOT NULL,
    bar_time TIMESTAMPTZ NOT NULL,
    features_json JSONB NOT NULL,
    theory_fixture TEXT NOT NULL DEFAULT '',
    generated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (strategy_id, feature_version, dataset_name, bar_time)
);
