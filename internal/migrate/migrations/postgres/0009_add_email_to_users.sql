-- +goose Up
ALTER TABLE users ADD COLUMN email TEXT;
CREATE UNIQUE INDEX idx_users_email ON users(email);

-- +goose Down
DROP INDEX idx_users_email;
ALTER TABLE users DROP COLUMN email;
