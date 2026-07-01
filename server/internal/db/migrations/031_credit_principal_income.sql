-- +goose Up
ALTER TABLE credits ADD COLUMN principal_affects_balance INTEGER NOT NULL DEFAULT 0;
ALTER TABLE credits ADD COLUMN principal_transaction_id TEXT REFERENCES transactions(id);

-- +goose Down
ALTER TABLE credits DROP COLUMN principal_transaction_id;
ALTER TABLE credits DROP COLUMN principal_affects_balance;
