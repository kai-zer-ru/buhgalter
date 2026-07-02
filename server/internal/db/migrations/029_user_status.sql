-- +goose Up
ALTER TABLE users ADD COLUMN status TEXT NOT NULL DEFAULT 'active'
    CHECK (status IN ('active', 'pending', 'banned'));

-- +goose Down
-- SQLite does not support DROP COLUMN in older versions; recreate would be destructive.
-- For dev rollback, manual intervention is required.
