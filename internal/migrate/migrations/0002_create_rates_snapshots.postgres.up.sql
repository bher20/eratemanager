-- +goose Up
CREATE TABLE IF NOT EXISTS rates_snapshots (
    id SERIAL PRIMARY KEY,
    provider TEXT NOT NULL,
    payload BYTEA NOT NULL,
    fetched_at TIMESTAMPTZ NOT NULL
);

-- +goose Down
DROP TABLE IF EXISTS rates_snapshots;
