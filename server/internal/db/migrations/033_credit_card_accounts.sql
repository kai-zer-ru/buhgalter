-- +goose Up
-- +goose StatementBegin
-- CHECK (type IN …) в SQLite нельзя расширить через ALTER — пересборка таблицы.
-- Миграции идут на соединении без foreign_keys(1) в DSN (см. db.runMigrations).
PRAGMA foreign_keys=OFF;

DROP TABLE IF EXISTS accounts_new;
DROP TABLE IF EXISTS accounts_legacy;

CREATE TABLE accounts_new (
    id                  TEXT PRIMARY KEY,
    user_id             TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name                TEXT NOT NULL,
    type                TEXT NOT NULL CHECK (type IN ('cash', 'bank', 'credit_card')),
    bank_id             TEXT REFERENCES banks(id),
    initial_balance     INTEGER NOT NULL DEFAULT 0,
    current_balance     INTEGER NOT NULL DEFAULT 0,
    credit_limit        INTEGER,
    payment_account_id  TEXT REFERENCES accounts_new(id),
    status              TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'archived')),
    is_primary          INTEGER NOT NULL DEFAULT 0,
    created_at          TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at          TEXT NOT NULL DEFAULT (datetime('now'))
);

INSERT INTO accounts_new (
    id, user_id, name, type, bank_id, initial_balance, current_balance,
    credit_limit, payment_account_id, status, is_primary, created_at, updated_at
)
SELECT
    id, user_id, name, type, bank_id, initial_balance, current_balance,
    NULL, NULL, status, is_primary, created_at, updated_at
FROM accounts;

DROP TABLE accounts;
ALTER TABLE accounts_new RENAME TO accounts;
CREATE INDEX IF NOT EXISTS idx_accounts_user ON accounts(user_id);

PRAGMA foreign_keys=ON;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
PRAGMA foreign_keys=OFF;

DROP TABLE IF EXISTS accounts_old;

CREATE TABLE accounts_old (
    id              TEXT PRIMARY KEY,
    user_id         TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    type            TEXT NOT NULL CHECK (type IN ('cash', 'bank')),
    bank_id         TEXT REFERENCES banks(id),
    initial_balance INTEGER NOT NULL DEFAULT 0,
    current_balance INTEGER NOT NULL DEFAULT 0,
    status          TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'archived')),
    is_primary      INTEGER NOT NULL DEFAULT 0,
    created_at      TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at      TEXT NOT NULL DEFAULT (datetime('now'))
);

INSERT INTO accounts_old (
    id, user_id, name, type, bank_id, initial_balance, current_balance,
    status, is_primary, created_at, updated_at
)
SELECT
    id, user_id, name, type, bank_id, initial_balance, current_balance,
    status, is_primary, created_at, updated_at
FROM accounts
WHERE type IN ('cash', 'bank');

DROP TABLE accounts;
ALTER TABLE accounts_old RENAME TO accounts;
CREATE INDEX IF NOT EXISTS idx_accounts_user ON accounts(user_id);

PRAGMA foreign_keys=ON;
-- +goose StatementEnd
