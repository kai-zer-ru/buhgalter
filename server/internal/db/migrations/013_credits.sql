-- +goose Up
CREATE TABLE credits (
    id                  TEXT PRIMARY KEY,
    user_id             TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name                TEXT,
    principal_amount    INTEGER NOT NULL,
    issue_date          TEXT NOT NULL,
    term_months         INTEGER NOT NULL,
    interest_rate       REAL NOT NULL DEFAULT 0,
    payment_interval    TEXT NOT NULL DEFAULT 'month'
                        CHECK (payment_interval IN ('month', 'week', 'two_weeks', 'manual')),
    paid_amount         INTEGER NOT NULL DEFAULT 0,
    monthly_payment     INTEGER NOT NULL,
    debit_account_id    TEXT NOT NULL REFERENCES accounts(id),
    added_retroactively INTEGER NOT NULL DEFAULT 0,
    recorded_at         TEXT NOT NULL DEFAULT (datetime('now')),
    status              TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'closed')),
    closed_at           TEXT,
    created_at          TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at          TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX idx_credits_user ON credits(user_id);
CREATE INDEX idx_credits_status ON credits(user_id, status);

-- +goose Down
DROP TABLE credits;
