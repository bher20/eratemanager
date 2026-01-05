-- +goose Up
CREATE TABLE IF NOT EXISTS batch_progress (
    batch_id TEXT NOT NULL,
    provider TEXT NOT NULL,
    status TEXT NOT NULL,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    error_message TEXT,
    retry_count INTEGER DEFAULT 0,
    PRIMARY KEY (batch_id, provider)
);
CREATE INDEX IF NOT EXISTS idx_batch_progress_status ON batch_progress(batch_id, status);

-- +goose Down
DROP TABLE IF EXISTS batch_progress;
