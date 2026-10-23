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

-- name: AttachReceiptToTransaction :execrows
UPDATE transactions
SET receipt_import_id = ?,
    receipt_item_index = NULL,
    store_id = COALESCE(store_id, ?),
    updated_at = ?
WHERE id = ? AND user_id = ? AND type = 'expense' AND receipt_import_id IS NULL;

-- name: DetachReceiptFromTransaction :execrows
UPDATE transactions
SET receipt_import_id = NULL,
    receipt_item_index = NULL,
    updated_at = ?
WHERE id = ? AND user_id = ? AND receipt_import_id IS NOT NULL;

-- name: ListReceiptLinkedTransactions :many
SELECT t.id, t.receipt_item_index
FROM transactions t
WHERE t.user_id = ? AND t.receipt_import_id = ?
ORDER BY t.receipt_item_index ASC, t.transaction_date ASC, t.id ASC;

-- name: ListReceiptLinkedTransactionCategories :many
SELECT t.receipt_item_index, t.category_id, t.subcategory_id
FROM transactions t
WHERE t.user_id = ? AND t.receipt_import_id = ?
  AND t.category_id IS NOT NULL AND t.category_id != ''
  AND t.receipt_item_index IS NOT NULL;

-- name: ListReceiptImportItemCategories :many
SELECT item_index, category_id, subcategory_id
FROM receipt_import_item_categories
WHERE user_id = ? AND receipt_import_id = ?
ORDER BY item_index ASC;

-- name: UpsertReceiptImportItemCategory :exec
INSERT INTO receipt_import_item_categories (
    receipt_import_id, item_index, user_id, category_id, subcategory_id
) VALUES (?, ?, ?, ?, ?)
ON CONFLICT(receipt_import_id, item_index) DO UPDATE SET
    category_id = excluded.category_id,
    subcategory_id = excluded.subcategory_id;

-- name: DeleteReceiptImportItemCategories :exec
DELETE FROM receipt_import_item_categories
WHERE user_id = ? AND receipt_import_id = ?;

-- name: CountReceiptLinkedTransactions :one
SELECT COUNT(*) AS n
FROM transactions t
WHERE t.user_id = ? AND t.receipt_import_id = ?;
