-- +goose Up
CREATE TABLE IF NOT EXISTS rates_snapshots (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    provider TEXT NOT NULL,
    payload BLOB NOT NULL,
    fetched_at TEXT NOT NULL
);

-- +goose Down
DROP TABLE IF EXISTS rates_snapshots;
