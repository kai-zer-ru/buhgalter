-- name: ListCategoriesByUser :many
SELECT
    c.id,
    c.name,
    c.type,
    c.icon,
    c.sort_order,
    c.is_primary,
    c.is_system,
    c.created_at,
    (SELECT COUNT(*) FROM subcategories s WHERE s.category_id = c.id) AS sub_count
FROM categories c
WHERE c.user_id = ?
ORDER BY c.is_system ASC, c.sort_order ASC, c.name ASC;

-- name: ListCategoriesByUserAndType :many
SELECT
    c.id,
    c.name,
    c.type,
    c.icon,
    c.sort_order,
    c.is_primary,
    c.is_system,
    c.created_at,
    (SELECT COUNT(*) FROM subcategories s WHERE s.category_id = c.id) AS sub_count
FROM categories c
WHERE c.user_id = ? AND c.type = ?
ORDER BY c.is_system ASC, c.sort_order ASC, c.name ASC;

-- name: GetCategoryByID :one
SELECT
    c.id,
    c.name,
    c.type,
    c.icon,
    c.sort_order,
    c.is_primary,
    c.is_system,
    c.created_at,
    (SELECT COUNT(*) FROM subcategories s WHERE s.category_id = c.id) AS sub_count
FROM categories c
WHERE c.id = ? AND c.user_id = ?;

-- name: InsertCategory :exec
INSERT INTO categories (id, user_id, name, type, icon, sort_order, is_primary, is_system, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: SetCategorySystem :exec
UPDATE categories SET is_system = ? WHERE id = ? AND user_id = ?;

-- name: UpdateSystemCategoryIcon :exec
UPDATE categories SET icon = ? WHERE id = ? AND user_id = ? AND is_system = 1;

-- name: UpdateCategory :exec
UPDATE categories
SET name = ?, icon = ?, sort_order = ?
WHERE id = ? AND user_id = ?;

-- name: UpdateCategorySortOrder :exec
UPDATE categories
SET sort_order = ?
WHERE id = ? AND user_id = ?;

-- name: ClearPrimaryCategories :exec
UPDATE categories
SET is_primary = 0
WHERE user_id = ? AND type = ?;

-- name: SetCategoryPrimary :exec
UPDATE categories
SET is_primary = 1
WHERE id = ? AND user_id = ?;

-- name: MaxCategorySortOrder :one
SELECT COALESCE(MAX(sort_order), 0)
FROM categories
WHERE user_id = ? AND type = ? AND is_system = 0;

-- name: CountCategoriesByType :one
SELECT COUNT(*) AS count
FROM categories
WHERE user_id = ? AND type = ?;

-- name: DeleteCategory :execrows
DELETE FROM categories
WHERE id = ? AND user_id = ?;

-- name: CountCategoriesByName :one
SELECT COUNT(*) AS count
FROM categories
WHERE user_id = ? AND name = ? AND type = ?;

-- name: CountCategoriesByNameExcluding :one
SELECT COUNT(*) AS count
FROM categories
WHERE user_id = ? AND name = ? AND type = ? AND id != ?;

-- name: ListSubcategoriesByCategory :many
SELECT id, category_id, name, icon, sort_order, created_at
FROM subcategories
WHERE category_id = ?
ORDER BY sort_order, name;

-- name: InsertSubcategory :exec
INSERT INTO subcategories (id, category_id, name, icon, sort_order, created_at)
VALUES (?, ?, ?, ?, ?, ?);

-- name: GetSubcategoryByID :one
SELECT s.id, s.category_id, s.name, s.icon, s.sort_order, s.created_at
FROM subcategories s
JOIN categories c ON c.id = s.category_id
WHERE s.id = ? AND c.user_id = ?;

-- name: UpdateSubcategory :exec
UPDATE subcategories
SET name = ?, icon = ?
WHERE id = ?;

-- name: UpdateSubcategorySortOrder :exec
UPDATE subcategories
SET sort_order = ?
WHERE id = ?;

-- name: MaxSubcategorySortOrder :one
SELECT COALESCE(MAX(sort_order), 0)
FROM subcategories
WHERE category_id = ?;

-- name: DeleteSubcategory :execrows
DELETE FROM subcategories
WHERE id = ?;
