-- +goose Up
CREATE TABLE IF NOT EXISTS scheduled_jobs (
    name TEXT PRIMARY KEY,
    last_run_at TEXT,
    last_duration_ms INTEGER,
    last_success INTEGER,
    last_error TEXT
);

-- +goose Down
DROP TABLE IF EXISTS scheduled_jobs;
