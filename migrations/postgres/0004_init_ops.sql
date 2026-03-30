CREATE TABLE IF NOT EXISTS ops_job_runs (
    job_id TEXT PRIMARY KEY,
    job_kind TEXT NOT NULL,
    state TEXT NOT NULL,
    requested_by TEXT NOT NULL DEFAULT '',
    payload_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    result_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    requested_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS ops_notifications_dead_letter (
    id BIGSERIAL PRIMARY KEY,
    channel TEXT NOT NULL,
    notification_kind TEXT NOT NULL,
    title TEXT NOT NULL,
    error TEXT NOT NULL,
    payload_json JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
