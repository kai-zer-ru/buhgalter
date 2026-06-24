-- +goose Up
CREATE TABLE debts (
    id              TEXT PRIMARY KEY,
    user_id         TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    debtor_id       TEXT NOT NULL REFERENCES debtors(id),
    direction       TEXT NOT NULL CHECK (direction IN ('lent', 'borrowed')),
    amount          INTEGER NOT NULL CHECK (amount > 0),
    affects_balance INTEGER NOT NULL DEFAULT 1,
    debt_date       TEXT NOT NULL,
    due_date        TEXT NOT NULL,
    description     TEXT,
    transaction_id  TEXT REFERENCES transactions(id),
    is_settled      INTEGER NOT NULL DEFAULT 0,
    settled_at      TEXT,
    created_at      TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX idx_debts_user ON debts(user_id);
CREATE INDEX idx_debts_debtor ON debts(debtor_id);
CREATE INDEX idx_debts_due ON debts(due_date);

-- +goose Down
DROP TABLE debts;
