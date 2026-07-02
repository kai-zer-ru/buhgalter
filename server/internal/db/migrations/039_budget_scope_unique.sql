-- +goose Up
-- Uniqueness by scope + category/subcategory + month (account_id does not allow duplicates).
DROP INDEX IF EXISTS idx_budgets_active_unique;
CREATE UNIQUE INDEX idx_budgets_active_unique ON budgets(
    user_id,
    scope,
    IFNULL(category_id, ''),
    IFNULL(subcategory_id, ''),
    month
) WHERE is_active = 1;

-- +goose Down
-- Forward-only migration.
