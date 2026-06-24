-- +goose Up
CREATE TABLE transactions (
    id                  TEXT PRIMARY KEY,
    user_id             TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    account_id          TEXT NOT NULL REFERENCES accounts(id),
    type                TEXT NOT NULL CHECK (type IN ('income', 'expense', 'transfer')),
    kind                TEXT NOT NULL DEFAULT 'manual' CHECK (kind IN ('manual', 'future')),
    amount              INTEGER NOT NULL CHECK (amount > 0),
    description         TEXT,
    category_id         TEXT REFERENCES categories(id),
    subcategory_id      TEXT REFERENCES subcategories(id),
    transfer_group_id   TEXT,
    transfer_account_id TEXT REFERENCES accounts(id),
    transaction_date    TEXT NOT NULL,
    affects_balance     INTEGER NOT NULL DEFAULT 1,
    created_at          TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at          TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX idx_tx_user ON transactions(user_id);
CREATE INDEX idx_tx_account ON transactions(account_id);
CREATE INDEX idx_tx_date ON transactions(transaction_date);
CREATE INDEX idx_tx_transfer_group ON transactions(transfer_group_id);
CREATE INDEX idx_tx_kind ON transactions(kind);

-- +goose Down
DROP TABLE transactions;
