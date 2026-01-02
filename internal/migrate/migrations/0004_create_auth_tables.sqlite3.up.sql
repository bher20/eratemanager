-- +goose Up
CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    username TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS tokens (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    token_hash TEXT NOT NULL UNIQUE,
    role TEXT NOT NULL,
    created_at TEXT NOT NULL,
    expires_at TEXT,
    last_used_at TEXT
);

-- +goose Down
DROP TABLE IF EXISTS tokens;
DROP TABLE IF EXISTS users;
