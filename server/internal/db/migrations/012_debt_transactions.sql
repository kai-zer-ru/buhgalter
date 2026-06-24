-- +goose Up
CREATE TABLE debt_transactions (
    debt_id         TEXT NOT NULL REFERENCES debts(id) ON DELETE CASCADE,
    transaction_id  TEXT NOT NULL REFERENCES transactions(id),
    role            TEXT NOT NULL CHECK (role IN ('open', 'settle')),
    PRIMARY KEY (debt_id, transaction_id)
);
CREATE UNIQUE INDEX idx_debt_transactions_tx ON debt_transactions(transaction_id);

-- +goose Down
DROP TABLE debt_transactions;
