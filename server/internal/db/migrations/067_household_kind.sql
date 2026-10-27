-- +goose Up
ALTER TABLE households ADD COLUMN kind TEXT NOT NULL DEFAULT 'family';

-- +goose Down
-- Forward-only migration.
