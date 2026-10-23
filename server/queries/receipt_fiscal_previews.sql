-- name: GetReceiptFiscalPreview :one
SELECT result_json
FROM receipt_fiscal_previews
WHERE user_id = ?
  AND fiscal_drive_number = ?
  AND fiscal_document_number = ?
  AND fiscal_sign = ?;

-- name: UpsertReceiptFiscalPreview :exec
INSERT INTO receipt_fiscal_previews (
    user_id,
    fiscal_drive_number,
    fiscal_document_number,
    fiscal_sign,
    result_json,
    fetched_at
) VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT(user_id, fiscal_drive_number, fiscal_document_number, fiscal_sign) DO UPDATE SET
    result_json = excluded.result_json,
    fetched_at = excluded.fetched_at;
