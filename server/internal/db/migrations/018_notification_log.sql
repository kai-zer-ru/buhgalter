-- +goose Up
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

-- +goose Down
DROP TABLE notification_log;
