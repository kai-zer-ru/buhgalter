-- name: GetReceiptImportByID :one
SELECT
    ri.id,
    ri.user_id,
    ri.fiscal_drive_number,
    ri.fiscal_document_number,
    ri.fiscal_sign,
    ri.total_sum,
    ri.store_id,
    ri.transaction_date,
    ri.payload_json,
    ri.fns_check_url,
    ri.created_at,
    s.name AS store_name
FROM receipt_imports ri
LEFT JOIN stores s ON s.id = ri.store_id
WHERE ri.id = ? AND ri.user_id = ?;

-- name: GetReceiptImportByFiscalKey :one
SELECT id
FROM receipt_imports
WHERE user_id = ?
  AND fiscal_drive_number = ?
  AND fiscal_document_number = ?
  AND fiscal_sign = ?;

-- name: InsertReceiptImport :exec
INSERT INTO receipt_imports (
    id, user_id, fiscal_drive_number, fiscal_document_number, fiscal_sign,
    total_sum, store_id, transaction_date, payload_json, fns_check_url, created_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetTransactionReceiptItemIndex :one
SELECT receipt_item_index
FROM transactions
WHERE id = ? AND user_id = ? AND receipt_import_id = ?;
