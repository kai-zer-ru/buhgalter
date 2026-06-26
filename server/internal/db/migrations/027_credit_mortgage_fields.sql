-- +goose Up
ALTER TABLE credits ADD COLUMN credit_kind TEXT NOT NULL DEFAULT 'consumer'
    CHECK (credit_kind IN ('consumer', 'mortgage'));
ALTER TABLE credits ADD COLUMN property_price INTEGER;
ALTER TABLE credits ADD COLUMN down_payment INTEGER NOT NULL DEFAULT 0;
ALTER TABLE credits ADD COLUMN down_payment_affects_balance INTEGER NOT NULL DEFAULT 0;
ALTER TABLE credits ADD COLUMN down_payment_transaction_id TEXT REFERENCES transactions(id);

-- +goose Down
-- SQLite does not support dropping columns directly; keep schema forward-only.
