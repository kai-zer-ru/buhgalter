-- +goose Up
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

-- +goose Down
DROP TABLE subcategories;
