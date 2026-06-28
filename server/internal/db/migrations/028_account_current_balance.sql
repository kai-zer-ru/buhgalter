-- +goose Up
ALTER TABLE accounts ADD COLUMN current_balance INTEGER NOT NULL DEFAULT 0;

UPDATE accounts SET current_balance = initial_balance;

CREATE INDEX IF NOT EXISTS idx_tx_user_account_balance
    ON transactions(user_id, account_id, kind, type, transaction_date);

-- +goose Down
DROP INDEX IF EXISTS idx_tx_user_account_balance;

-- SQLite cannot drop column in older versions; recreate pattern omitted for forward-only deploys.
