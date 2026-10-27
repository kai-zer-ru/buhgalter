-- name: ListMerchantsByUser :many
SELECT id, user_id, name, icon, created_at
FROM merchants
WHERE user_id IN (sqlc.slice('owner_ids'))
ORDER BY name COLLATE NOCASE ASC;

-- name: GetMerchantUserID :one
SELECT user_id FROM merchants WHERE id = ?;

-- name: GetMerchantByID :one
SELECT id, user_id, name, icon, created_at
FROM merchants
WHERE id = sqlc.arg('id') AND user_id IN (sqlc.slice('owner_ids'));

-- name: GetMerchantByName :one
SELECT id, user_id, name, icon, created_at
FROM merchants
WHERE user_id = sqlc.arg(user_id) AND lower(name) = lower(sqlc.arg(name));

-- name: InsertMerchant :exec
INSERT INTO merchants (id, user_id, name, icon, created_at)
VALUES (?, ?, ?, ?, ?);

-- name: UpdateMerchant :execrows
UPDATE merchants
SET name = ?, icon = ?
WHERE id = ? AND user_id = ?;

-- name: DeleteMerchant :execrows
DELETE FROM merchants
WHERE id = ? AND user_id = ?;

-- name: ListMerchantAliasesByUser :many
SELECT id, user_id, merchant_id, name, created_at
FROM merchant_aliases
WHERE user_id IN (sqlc.slice('owner_ids'))
ORDER BY name COLLATE NOCASE ASC, id ASC;

-- name: InsertMerchantAlias :exec
INSERT INTO merchant_aliases (id, user_id, merchant_id, name, created_at)
VALUES (?, ?, ?, ?, ?);

-- name: DeleteMerchantAliasesByMerchant :exec
DELETE FROM merchant_aliases
WHERE merchant_id = ? AND user_id = ?;

-- name: ReassignTransactionsMerchant :exec
UPDATE transactions
SET merchant_id = sqlc.arg(target_id)
WHERE user_id = sqlc.arg(user_id) AND merchant_id = sqlc.arg(source_id);

-- name: ReassignTemplateMerchant :exec
UPDATE transaction_templates
SET merchant_id = sqlc.arg(target_id)
WHERE user_id = sqlc.arg(user_id) AND merchant_id = sqlc.arg(source_id);
