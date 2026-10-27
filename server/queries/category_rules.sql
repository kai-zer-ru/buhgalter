-- name: ListCategoryRulesByUser :many
SELECT
    id, user_id, kind, pattern,
    category_id, subcategory_id, default_account_id,
    priority, created_at, updated_at
FROM category_rules
WHERE user_id IN (sqlc.slice('owner_ids'))
ORDER BY priority ASC, id ASC;

-- name: ListCategoryRulesByUserAndKind :many
SELECT
    id, user_id, kind, pattern,
    category_id, subcategory_id, default_account_id,
    priority, created_at, updated_at
FROM category_rules
WHERE kind = sqlc.arg('kind') AND user_id IN (sqlc.slice('owner_ids'))
ORDER BY priority ASC, id ASC;

-- name: GetCategoryRuleByID :one
SELECT
    id, user_id, kind, pattern,
    category_id, subcategory_id, default_account_id,
    priority, created_at, updated_at
FROM category_rules
WHERE id = sqlc.arg('id') AND user_id IN (sqlc.slice('owner_ids'));

-- name: CountCategoryRulesByUser :one
SELECT COUNT(*) FROM category_rules WHERE user_id = ?;

-- name: MaxCategoryRulePriority :one
SELECT COALESCE(MAX(priority), 0)
FROM category_rules
WHERE user_id = ?;

-- name: InsertCategoryRule :exec
INSERT INTO category_rules (
    id, user_id, kind, pattern,
    category_id, subcategory_id, default_account_id,
    priority, created_at, updated_at
) VALUES (
    ?, ?, ?, ?,
    ?, ?, ?,
    ?, ?, ?
);

-- name: UpdateCategoryRule :execrows
UPDATE category_rules
SET kind = ?,
    pattern = ?,
    category_id = ?,
    subcategory_id = ?,
    default_account_id = ?,
    updated_at = ?
WHERE id = ? AND user_id = ?;

-- name: UpdateCategoryRulePriority :exec
UPDATE category_rules
SET priority = ?, updated_at = ?
WHERE id = ? AND user_id = ?;

-- name: DeleteCategoryRule :execrows
DELETE FROM category_rules
WHERE id = ? AND user_id = ?;
