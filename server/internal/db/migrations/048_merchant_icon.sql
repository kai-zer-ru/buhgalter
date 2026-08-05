-- +goose Up
ALTER TABLE merchants ADD COLUMN icon TEXT NOT NULL DEFAULT 'default';

-- +goose Down
-- SQLite: column left in place on down.
