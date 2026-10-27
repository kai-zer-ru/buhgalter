-- +goose Up
ALTER TABLE accounts ADD COLUMN currency TEXT NOT NULL DEFAULT 'RUB';
UPDATE accounts
SET currency = COALESCE((SELECT u.currency FROM users u WHERE u.id = accounts.user_id), 'RUB');

ALTER TABLE budgets ADD COLUMN currency TEXT NOT NULL DEFAULT 'RUB';
UPDATE budgets
SET currency = COALESCE((SELECT u.currency FROM users u WHERE u.id = budgets.user_id), 'RUB');

ALTER TABLE debts ADD COLUMN currency TEXT NOT NULL DEFAULT 'RUB';
UPDATE debts
SET currency = COALESCE(
    (SELECT a.currency
     FROM transactions t
     JOIN accounts a ON a.id = t.account_id
     WHERE t.id = debts.transaction_id),
    (SELECT u.currency FROM users u WHERE u.id = debts.user_id),
    'RUB'
);

-- +goose Down
-- Forward-only migration.
