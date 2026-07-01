-- Snapshot for sqlc (source of truth: internal/db/migrations/*.sql via goose).
-- After each migration, update this file before `make sqlc`.

-- SQLite catalog (sqlc schema stub; system table at runtime).
CREATE TABLE sqlite_master (
    type TEXT,
    name TEXT,
    tbl_name TEXT,
    rootpage INTEGER,
    sql TEXT
);

CREATE TABLE system_settings (
    id              INTEGER PRIMARY KEY CHECK (id = 1),
    is_configured   INTEGER NOT NULL DEFAULT 0,
    db_path         TEXT NOT NULL DEFAULT '',
    external_url    TEXT,
    notification_secret_key TEXT NOT NULL DEFAULT '',
    app_version     TEXT NOT NULL DEFAULT '',
    previous_app_version TEXT,
    registration_enabled INTEGER NOT NULL DEFAULT 0,
    backup_enabled  INTEGER NOT NULL DEFAULT 0,
    backup_time     TEXT DEFAULT '03:00',
    backup_retention INTEGER NOT NULL DEFAULT 7,
    created_at      TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at      TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE users (
    id              TEXT PRIMARY KEY,
    login           TEXT NOT NULL UNIQUE,
    password_hash   TEXT NOT NULL,
    display_name    TEXT,
    is_admin        INTEGER NOT NULL DEFAULT 0,
    language        TEXT NOT NULL DEFAULT 'ru',
    currency        TEXT NOT NULL DEFAULT 'RUB',
    timezone        TEXT NOT NULL DEFAULT 'Europe/Moscow',
    theme           TEXT NOT NULL DEFAULT 'light',
    status          TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'pending', 'banned')),
    created_at      TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at      TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE sessions (
    id              TEXT PRIMARY KEY,
    user_id         TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash      TEXT NOT NULL UNIQUE,
    last_activity   TEXT NOT NULL DEFAULT (datetime('now')),
    expires_at      TEXT NOT NULL,
    ip_address      TEXT,
    user_agent      TEXT,
    created_at      TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX idx_sessions_user ON sessions(user_id);
CREATE INDEX idx_sessions_token ON sessions(token_hash);

CREATE TABLE api_tokens (
    id              TEXT PRIMARY KEY,
    user_id         TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    token_hash      TEXT NOT NULL UNIQUE,
    token_prefix    TEXT NOT NULL,
    expires_at      TEXT,
    last_used_at    TEXT,
    created_at      TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX idx_api_tokens_user ON api_tokens(user_id);

CREATE TABLE banks (
    id              TEXT PRIMARY KEY,
    name            TEXT NOT NULL,
    bic             TEXT,
    icon_path       TEXT NOT NULL,
    sort_order      INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE accounts (
    id                  TEXT PRIMARY KEY,
    user_id             TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name                TEXT NOT NULL,
    type                TEXT NOT NULL CHECK (type IN ('cash', 'bank', 'credit_card')),
    bank_id             TEXT REFERENCES banks(id),
    initial_balance     INTEGER NOT NULL DEFAULT 0,
    current_balance     INTEGER NOT NULL DEFAULT 0,
    credit_limit        INTEGER,
    payment_account_id  TEXT REFERENCES accounts(id),
    status              TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'archived')),
    is_primary          INTEGER NOT NULL DEFAULT 0,
    created_at          TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at          TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX idx_accounts_user ON accounts(user_id);

-- Staging table for migration 033 rebuild (sqlc schema; ephemeral at runtime).
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

CREATE TABLE categories (
    id              TEXT PRIMARY KEY,
    user_id         TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    type            TEXT NOT NULL CHECK (type IN ('income', 'expense')),
    icon            TEXT NOT NULL DEFAULT 'default',
    sort_order      INTEGER NOT NULL DEFAULT 0,
    is_primary      INTEGER NOT NULL DEFAULT 0,
    is_system       INTEGER NOT NULL DEFAULT 0,
    created_at      TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX idx_categories_user ON categories(user_id);

CREATE TABLE subcategories (
    id              TEXT PRIMARY KEY,
    category_id     TEXT NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    icon            TEXT NOT NULL DEFAULT 'default',
    sort_order      INTEGER NOT NULL DEFAULT 0,
    created_at      TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(category_id, name)
);
CREATE INDEX idx_subcategories_category ON subcategories(category_id);

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
CREATE INDEX idx_tx_user_account_balance ON transactions(user_id, account_id, kind, type, transaction_date);

CREATE TABLE recurring_operations (
    id              TEXT PRIMARY KEY,
    user_id         TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type            TEXT NOT NULL CHECK (type IN ('income', 'expense')),
    amount          INTEGER NOT NULL CHECK (amount > 0),
    description     TEXT,
    account_id      TEXT NOT NULL REFERENCES accounts(id),
    category_id     TEXT NOT NULL REFERENCES categories(id),
    subcategory_id  TEXT REFERENCES subcategories(id),
    period          TEXT NOT NULL CHECK (period IN ('week', 'two_weeks', 'month', 'year')),
    weekday         INTEGER CHECK (weekday BETWEEN 1 AND 7),
    day_of_month    INTEGER CHECK (day_of_month BETWEEN 1 AND 31),
    start_date      TEXT NOT NULL,
    time_local      TEXT NOT NULL DEFAULT '00:00',
    next_run_at     TEXT NOT NULL,
    last_run_at     TEXT,
    active          INTEGER NOT NULL DEFAULT 1,
    created_at      TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at      TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX idx_recurring_ops_user ON recurring_operations(user_id);
CREATE INDEX idx_recurring_ops_due ON recurring_operations(user_id, active, next_run_at);

CREATE TABLE debtors (
    id              TEXT PRIMARY KEY,
    user_id         TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name            TEXT NOT NULL COLLATE NOCASE,
    created_at      TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(user_id, name)
);

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

CREATE TABLE debt_transactions (
    debt_id         TEXT NOT NULL REFERENCES debts(id) ON DELETE CASCADE,
    transaction_id  TEXT NOT NULL REFERENCES transactions(id),
    role            TEXT NOT NULL CHECK (role IN ('open', 'settle')),
    PRIMARY KEY (debt_id, transaction_id)
);
CREATE UNIQUE INDEX idx_debt_transactions_tx ON debt_transactions(transaction_id);

CREATE TABLE credits (
    id                  TEXT PRIMARY KEY,
    user_id             TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name                TEXT,
    credit_kind         TEXT NOT NULL DEFAULT 'consumer'
                        CHECK (credit_kind IN ('consumer', 'mortgage')),
    principal_amount    INTEGER NOT NULL,
    property_price      INTEGER,
    down_payment        INTEGER NOT NULL DEFAULT 0,
    down_payment_affects_balance INTEGER NOT NULL DEFAULT 0,
    down_payment_transaction_id TEXT REFERENCES transactions(id),
    principal_affects_balance INTEGER NOT NULL DEFAULT 0,
    principal_transaction_id TEXT REFERENCES transactions(id),
    issue_date          TEXT NOT NULL,
    term_months         INTEGER NOT NULL,
    interest_rate       REAL NOT NULL DEFAULT 0,
    payment_interval    TEXT NOT NULL DEFAULT 'month'
                        CHECK (payment_interval IN ('month', 'week', 'two_weeks', 'manual')),
    paid_amount         INTEGER NOT NULL DEFAULT 0,
    monthly_payment     INTEGER NOT NULL,
    debit_account_id    TEXT NOT NULL REFERENCES accounts(id),
    debit_time_local    TEXT,
    bank_id             TEXT REFERENCES banks(id),
    bank_id_locked      INTEGER NOT NULL DEFAULT 0,
    added_retroactively INTEGER NOT NULL DEFAULT 0,
    recorded_at         TEXT NOT NULL DEFAULT (datetime('now')),
    status              TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'closed')),
    closed_at           TEXT,
    created_at          TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at          TEXT NOT NULL DEFAULT (datetime('now'))
);

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
CREATE INDEX idx_credits_user ON credits(user_id);
CREATE INDEX idx_credits_status ON credits(user_id, status);
CREATE INDEX idx_credit_payments_credit ON credit_payments(credit_id);
CREATE INDEX idx_credit_payments_date ON credit_payments(payment_date);
CREATE INDEX idx_credit_payments_applied ON credit_payments(credit_id, is_applied);

CREATE TABLE import_idempotency (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    idempotency_key TEXT NOT NULL,
    response_json TEXT NOT NULL,
    created_at TEXT NOT NULL,
    UNIQUE (user_id, idempotency_key)
);
CREATE INDEX idx_import_idempotency_user_key ON import_idempotency (user_id, idempotency_key);

CREATE TABLE password_reset_requests (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    created_at TEXT NOT NULL,
    dismissed_at TEXT
);
CREATE INDEX idx_password_reset_requests_pending ON password_reset_requests(user_id)
WHERE dismissed_at IS NULL;

CREATE TABLE import_jobs (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    filename TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('queued', 'running', 'done', 'failed')),
    error_message TEXT,
    report_json TEXT,
    created_at TEXT NOT NULL,
    started_at TEXT,
    finished_at TEXT,
    updated_at TEXT NOT NULL
);
CREATE INDEX idx_import_jobs_user_created ON import_jobs (user_id, created_at DESC);

CREATE TABLE notification_settings (
    user_id             TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    telegram_enabled    INTEGER NOT NULL DEFAULT 0,
    telegram_bot_token  TEXT,
    telegram_chat_id    TEXT,
    max_enabled         INTEGER NOT NULL DEFAULT 0,
    max_provider        TEXT CHECK (max_provider IN ('a161', 'official')),
    max_token           TEXT,
    max_user_id         INTEGER,
    max_recipient_id    INTEGER,
    trigger_debt        INTEGER NOT NULL DEFAULT 1,
    trigger_credit      INTEGER NOT NULL DEFAULT 1,
    trigger_planned     INTEGER NOT NULL DEFAULT 1,
    trigger_negative_balance INTEGER NOT NULL DEFAULT 1,
    trigger_user_registration INTEGER NOT NULL DEFAULT 1,
    trigger_password_reset INTEGER NOT NULL DEFAULT 1,
    debt_days_before    INTEGER NOT NULL DEFAULT 1,
    my_debt_overdue_days_limit INTEGER NOT NULL DEFAULT 7,
    owed_debt_overdue_start_after_days INTEGER NOT NULL DEFAULT 0,
    owed_debt_overdue_days_limit INTEGER NOT NULL DEFAULT 7,
    credit_days_before  INTEGER NOT NULL DEFAULT 1,
    notification_time_local TEXT NOT NULL DEFAULT '00:00',
    updated_at          TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE notification_log (
    id              TEXT PRIMARY KEY,
    user_id         TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    trigger_type    TEXT NOT NULL,
    channel         TEXT NOT NULL,
    entity_id       TEXT,
    dedup_date      TEXT,
    status          TEXT NOT NULL,
    message         TEXT,
    created_at      TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX idx_notification_log_dedup
    ON notification_log (user_id, trigger_type, channel, entity_id, dedup_date);

CREATE TABLE notification_templates (
    user_id         TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    trigger_type    TEXT NOT NULL CHECK (trigger_type IN (
                        'debt_overdue', 'debt_due_soon', 'credit_payment', 'planned_operation',
                        'balance_shortfall',
                        'user_registration', 'password_reset', 'test'
                    )),
    template        TEXT NOT NULL,
    updated_at      TEXT NOT NULL DEFAULT (datetime('now')),
    PRIMARY KEY (user_id, trigger_type)
);

