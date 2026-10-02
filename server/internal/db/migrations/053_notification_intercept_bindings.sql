-- +goose Up
CREATE TABLE notification_intercept_bank_bindings (
    user_id       TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    account_id    TEXT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    bank_id       TEXT NOT NULL REFERENCES banks(id),
    package_name  TEXT NOT NULL,
    PRIMARY KEY (user_id, account_id)
);

CREATE TABLE notification_intercept_card_bindings (
    user_id     TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    bank_id     TEXT NOT NULL REFERENCES banks(id),
    last4       TEXT NOT NULL CHECK (length(last4) = 4 AND last4 GLOB '[0-9][0-9][0-9][0-9]'),
    account_id  TEXT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, bank_id, last4)
);

CREATE INDEX idx_notification_intercept_card_bindings_user
    ON notification_intercept_card_bindings (user_id);

-- +goose Down
DROP TABLE notification_intercept_card_bindings;
DROP TABLE notification_intercept_bank_bindings;
