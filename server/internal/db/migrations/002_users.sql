-- +goose Up
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
    created_at      TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at      TEXT NOT NULL DEFAULT (datetime('now'))
);

-- +goose Down
DROP TABLE users;
