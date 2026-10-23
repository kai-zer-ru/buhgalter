-- +goose Up
CREATE TABLE receipt_import_item_categories (
    receipt_import_id   TEXT NOT NULL REFERENCES receipt_imports(id) ON DELETE CASCADE,
    item_index          INTEGER NOT NULL,
    user_id             TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    category_id         TEXT NOT NULL,
    subcategory_id      TEXT,
    PRIMARY KEY (receipt_import_id, item_index)
);
CREATE INDEX idx_receipt_item_cat_user ON receipt_import_item_categories(user_id, receipt_import_id);

-- +goose Down
-- Forward-only migration.
