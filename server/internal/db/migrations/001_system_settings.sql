-- +goose Up
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

INSERT INTO system_settings (id, is_configured, db_path)
VALUES (1, 0, '')
ON CONFLICT(id) DO NOTHING;

-- +goose Down
DROP TABLE system_settings;
