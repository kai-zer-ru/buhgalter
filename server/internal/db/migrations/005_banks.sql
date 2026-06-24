-- +goose Up
CREATE TABLE banks (
    id              TEXT PRIMARY KEY,
    name            TEXT NOT NULL,
    bic             TEXT,
    icon_path       TEXT NOT NULL,
    sort_order      INTEGER NOT NULL DEFAULT 0
);

-- +goose Down
DROP TABLE banks;
