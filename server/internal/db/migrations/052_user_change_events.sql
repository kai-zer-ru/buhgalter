-- +goose Up

CREATE TABLE user_change_events (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id      TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    entity_type  TEXT NOT NULL CHECK (entity_type IN ('transaction')),
    entity_id    TEXT NOT NULL,
    action       TEXT NOT NULL CHECK (action IN ('created', 'updated', 'deleted')),
    occurred_at  TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX idx_user_change_events_user_id ON user_change_events(user_id, id);

-- +goose StatementBegin
CREATE TRIGGER trg_transactions_change_insert
AFTER INSERT ON transactions
BEGIN
    INSERT INTO user_change_events (user_id, entity_type, entity_id, action, occurred_at)
    VALUES (NEW.user_id, 'transaction', NEW.id, 'created', datetime('now'));
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_transactions_change_update
AFTER UPDATE ON transactions
BEGIN
    INSERT INTO user_change_events (user_id, entity_type, entity_id, action, occurred_at)
    VALUES (NEW.user_id, 'transaction', NEW.id, 'updated', datetime('now'));
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_transactions_change_delete
AFTER DELETE ON transactions
BEGIN
    INSERT INTO user_change_events (user_id, entity_type, entity_id, action, occurred_at)
    VALUES (OLD.user_id, 'transaction', OLD.id, 'deleted', datetime('now'));
END;
-- +goose StatementEnd

-- +goose Down

DROP TRIGGER IF EXISTS trg_transactions_change_delete;
DROP TRIGGER IF EXISTS trg_transactions_change_update;
DROP TRIGGER IF EXISTS trg_transactions_change_insert;
DROP TABLE IF EXISTS user_change_events;
