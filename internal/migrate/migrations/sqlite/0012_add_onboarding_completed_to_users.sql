-- +goose Up
ALTER TABLE users ADD COLUMN onboarding_completed BOOLEAN DEFAULT 0;

-- +goose Down
ALTER TABLE users DROP COLUMN onboarding_completed;
