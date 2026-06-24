-- +goose Up
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
    debt_days_before    INTEGER NOT NULL DEFAULT 1,
    my_debt_overdue_days_limit INTEGER NOT NULL DEFAULT 7,
    owed_debt_overdue_start_after_days INTEGER NOT NULL DEFAULT 0,
    owed_debt_overdue_days_limit INTEGER NOT NULL DEFAULT 7,
    credit_days_before  INTEGER NOT NULL DEFAULT 1,
    notification_time_local TEXT NOT NULL DEFAULT '00:00',
    updated_at          TEXT NOT NULL DEFAULT (datetime('now'))
);

-- +goose Down
DROP TABLE notification_settings;
