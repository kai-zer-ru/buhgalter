-- +goose Up
CREATE TABLE stores (
    id              TEXT PRIMARY KEY,
    user_id         TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    name_normalized TEXT NOT NULL,
    created_at      TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(user_id, name_normalized)
);
CREATE INDEX idx_stores_user ON stores(user_id);

CREATE TABLE receipt_imports (
    id                      TEXT PRIMARY KEY,
    user_id                 TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    fiscal_drive_number     TEXT NOT NULL,
    fiscal_document_number  TEXT NOT NULL,
    fiscal_sign             TEXT NOT NULL,
    total_sum               INTEGER NOT NULL,
    store_id                TEXT REFERENCES stores(id),
    transaction_date        TEXT NOT NULL,
    payload_json            TEXT NOT NULL,
    fns_check_url           TEXT NOT NULL,
    created_at              TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(user_id, fiscal_drive_number, fiscal_document_number, fiscal_sign)
);
CREATE INDEX idx_receipt_imports_user ON receipt_imports(user_id);

ALTER TABLE transactions ADD COLUMN store_id TEXT REFERENCES stores(id);
ALTER TABLE transactions ADD COLUMN receipt_import_id TEXT REFERENCES receipt_imports(id);
ALTER TABLE transactions ADD COLUMN receipt_item_index INTEGER;
CREATE INDEX idx_tx_store ON transactions(user_id, store_id);
CREATE INDEX idx_tx_receipt_import ON transactions(receipt_import_id);

-- +goose Down
-- Forward-only migration.
