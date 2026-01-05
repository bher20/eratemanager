-- +goose Up
-- Add first_name and last_name columns to users table
ALTER TABLE users ADD COLUMN first_name TEXT DEFAULT '';
ALTER TABLE users ADD COLUMN last_name TEXT DEFAULT '';

-- +goose Down
ALTER TABLE users DROP COLUMN IF EXISTS first_name;
ALTER TABLE users DROP COLUMN IF EXISTS last_name;
