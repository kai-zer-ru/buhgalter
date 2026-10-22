-- name: ListTransactionTemplatesByUser :many
SELECT
    id, user_id, name, type,
    account_id, to_account_id, category_id, subcategory_id,
    amount, description, merchant_id, icon, sort_order,
    created_at, updated_at
FROM transaction_templates
WHERE user_id = ?
ORDER BY sort_order ASC, name COLLATE NOCASE ASC;

-- name: GetTransactionTemplateByID :one
SELECT
    id, user_id, name, type,
    account_id, to_account_id, category_id, subcategory_id,
    amount, description, merchant_id, icon, sort_order,
    created_at, updated_at
FROM transaction_templates
WHERE id = ? AND user_id = ?;

-- name: CountTransactionTemplatesByUser :one
SELECT COUNT(*) FROM transaction_templates WHERE user_id = ?;

-- name: MaxTransactionTemplateSortOrder :one
SELECT COALESCE(MAX(sort_order), 0)
FROM transaction_templates
WHERE user_id = ?;

-- name: InsertTransactionTemplate :exec
INSERT INTO transaction_templates (
    id, user_id, name, type,
    account_id, to_account_id, category_id, subcategory_id,
    amount, description, merchant_id, icon, sort_order,
    created_at, updated_at
) VALUES (
    ?, ?, ?, ?,
    ?, ?, ?, ?,
    ?, ?, ?, ?, ?,
    ?, ?
);

-- name: UpdateTransactionTemplate :execrows
UPDATE transaction_templates
SET name = ?,
    type = ?,
    account_id = ?,
    to_account_id = ?,
    category_id = ?,
    subcategory_id = ?,
    amount = ?,
    description = ?,
    merchant_id = ?,
    icon = ?,
    updated_at = ?
WHERE id = ? AND user_id = ?;

-- name: UpdateTransactionTemplateSortOrder :exec
UPDATE transaction_templates
SET sort_order = ?, updated_at = ?
WHERE id = ? AND user_id = ?;

-- name: DeleteTransactionTemplate :execrows
DELETE FROM transaction_templates
WHERE id = ? AND user_id = ?;

-- name: ListTagsByTemplateID :many
SELECT tg.id, tg.user_id, tg.name, tg.created_at
FROM transaction_template_tags tt
JOIN tags tg ON tg.id = tt.tag_id
WHERE tt.template_id = ?
ORDER BY tg.name COLLATE NOCASE ASC;

-- name: ListTagsForTemplates :many
SELECT tt.template_id, tg.id AS tag_id, tg.name AS tag_name
FROM transaction_template_tags tt
JOIN tags tg ON tg.id = tt.tag_id
WHERE tt.template_id IN (sqlc.slice('template_ids'))
ORDER BY tg.name COLLATE NOCASE ASC;

-- name: DeleteTransactionTemplateTags :exec
DELETE FROM transaction_template_tags WHERE template_id = ?;

-- name: InsertTransactionTemplateTag :exec
INSERT INTO transaction_template_tags (template_id, tag_id)
VALUES (?, ?);
