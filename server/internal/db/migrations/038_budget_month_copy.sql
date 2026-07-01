-- +goose Up
ALTER TABLE budgets ADD COLUMN month TEXT NOT NULL DEFAULT '';
ALTER TABLE budgets ADD COLUMN copy_forward INTEGER NOT NULL DEFAULT 0;

UPDATE budgets
SET month = COALESCE(
    (SELECT strftime('%Y-%m', MIN(bp.period_start)) FROM budget_periods bp WHERE bp.budget_id = budgets.id),
    strftime('%Y-%m', created_at)
)
WHERE month = '';

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
