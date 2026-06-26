-- +goose Up
ALTER TABLE credits ADD COLUMN debit_time_local TEXT;
ALTER TABLE credits ADD COLUMN bank_id TEXT REFERENCES banks(id);
ALTER TABLE credits ADD COLUMN bank_id_locked INTEGER NOT NULL DEFAULT 0;

-- +goose Down
-- SQLite does not support dropping columns directly; keep schema forward-only.
