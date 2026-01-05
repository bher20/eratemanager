-- +goose Up
CREATE TABLE IF NOT EXISTS casbin_rules (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    ptype TEXT,
    v0 TEXT,
    v1 TEXT,
    v2 TEXT,
    v3 TEXT,
    v4 TEXT,
    v5 TEXT
);

-- +goose Down
DROP TABLE IF EXISTS casbin_rules;
