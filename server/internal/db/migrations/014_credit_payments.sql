-- +goose Up
CREATE TABLE credit_payments (
    id                  TEXT PRIMARY KEY,
    credit_id           TEXT NOT NULL REFERENCES credits(id) ON DELETE CASCADE,
    transaction_id      TEXT REFERENCES transactions(id),
    amount              INTEGER NOT NULL,
    payment_date        TEXT NOT NULL,
    kind                TEXT NOT NULL DEFAULT 'scheduled'
                        CHECK (kind IN ('scheduled', 'early', 'auto', 'retroactive')),
    is_applied          INTEGER NOT NULL DEFAULT 0,
    exclude_from_stats  INTEGER NOT NULL DEFAULT 0,
    created_at          TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX idx_credit_payments_credit ON credit_payments(credit_id);
CREATE INDEX idx_credit_payments_date ON credit_payments(payment_date);
CREATE INDEX idx_credit_payments_applied ON credit_payments(credit_id, is_applied);

-- +goose Down
DROP TABLE credit_payments;
