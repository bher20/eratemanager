-- 002_create_rates_snapshots.sql
CREATE TABLE IF NOT EXISTS rates_snapshots (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    provider TEXT NOT NULL,
    payload BLOB NOT NULL,
    fetched_at TEXT NOT NULL
);
