-- +goose Up
-- Cached ParsedReceipt from proverkacheka (per user + fiscal key) to avoid repeat API calls.
CREATE TABLE receipt_fiscal_previews (
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    fiscal_drive_number TEXT NOT NULL,
    fiscal_document_number TEXT NOT NULL,
    fiscal_sign TEXT NOT NULL,
    result_json TEXT NOT NULL,
    fetched_at TEXT NOT NULL,
    PRIMARY KEY (user_id, fiscal_drive_number, fiscal_document_number, fiscal_sign)
);
CREATE INDEX idx_receipt_fiscal_previews_user_fetched ON receipt_fiscal_previews (user_id, fetched_at DESC);

-- +goose Down
DROP TABLE receipt_fiscal_previews;
