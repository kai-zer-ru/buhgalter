-- name: ReassignAccountOwner :execrows
UPDATE accounts
SET user_id = ?,
    original_owner_user_id = ?,
    is_primary = 0,
    updated_at = ?
WHERE id = ? AND user_id = ?;

-- name: ListAccountsForMove :many
SELECT id, original_owner_user_id, status FROM accounts WHERE user_id = ?;

-- name: DeleteUserChangeEventsForPrincipalTransactions :exec
DELETE FROM user_change_events
WHERE user_change_events.user_id = ?
  AND entity_type = 'transaction'
  AND entity_id IN (SELECT id FROM transactions WHERE transactions.user_id = ?);

-- name: ReassignTransactionsByAccount :execrows
UPDATE transactions
SET user_id = ?, updated_at = ?
WHERE account_id = ? AND user_id = ?;

-- name: ReassignCreditsByDebitAccount :execrows
UPDATE credits
SET user_id = ?, updated_at = ?
WHERE debit_account_id = ? AND user_id = ?;

-- name: ReassignSubscriptionsByAccount :execrows
UPDATE subscriptions
SET user_id = ?, updated_at = ?
WHERE account_id = ? AND user_id = ?;

-- name: ReassignRecurringByAccount :execrows
UPDATE recurring_operations
SET user_id = ?, updated_at = ?
WHERE account_id = ? AND user_id = ?;

-- name: ReassignTemplatesByAccount :execrows
UPDATE transaction_templates
SET user_id = ?, updated_at = ?
WHERE account_id = ? AND user_id = ?;

-- name: ReassignBudgetsByAccount :execrows
UPDATE budgets
SET user_id = ?, updated_at = ?
WHERE account_id = ? AND user_id = ?;

-- name: ReassignDebtsByOpenAccount :execrows
UPDATE debts
SET user_id = ?
WHERE id IN (
    SELECT d.id FROM debts d
    JOIN debt_transactions dtx ON dtx.debt_id = d.id AND dtx.role = 'open'
    JOIN transactions t ON t.id = dtx.transaction_id
    WHERE t.account_id = ? AND d.user_id = ?
);

-- name: ReassignReceiptImportsByAccount :execrows
UPDATE receipt_imports
SET user_id = ?
WHERE receipt_imports.user_id = ?
  AND receipt_imports.id IN (
    SELECT DISTINCT t.receipt_import_id
    FROM transactions t
    WHERE t.account_id = ?
      AND t.receipt_import_id IS NOT NULL
  );

-- name: ClearCategoryRuleDefaultAccount :exec
UPDATE category_rules
SET default_account_id = NULL
WHERE default_account_id = ? AND user_id = ?;

-- name: ListTransactionCatalogRefsByAccount :many
SELECT id, category_id, subcategory_id, merchant_id, store_id
FROM transactions
WHERE account_id = ? AND user_id = ?;

-- name: UpdateTransactionCatalogRefs :exec
UPDATE transactions
SET category_id = ?, subcategory_id = ?, merchant_id = ?, store_id = ?, updated_at = ?
WHERE id = ? AND user_id = ?;

-- name: ListTransactionTagIDs :many
SELECT tag_id FROM transaction_tags WHERE transaction_id = ?;

-- name: ReplaceTransactionTag :exec
UPDATE transaction_tags SET tag_id = ?
WHERE transaction_id = ? AND tag_id = ?;

-- name: GetAccountPaymentLink :one
SELECT id, type, payment_account_id, auto_topup_enabled, auto_topup_source_account_id, is_primary, status, user_id
FROM accounts
WHERE id = ? AND user_id = ?;

-- name: GetAccountOwnerID :one
SELECT user_id FROM accounts WHERE id = ?;

-- name: ListCreditPaymentAccountIDs :many
SELECT DISTINCT t.account_id
FROM credit_payments cp
JOIN credits c ON c.id = cp.credit_id
JOIN transactions t ON t.id = cp.transaction_id
WHERE c.debit_account_id = ? AND c.user_id = ? AND t.account_id != ?;

-- name: ListDebtSettleAccountIDs :many
SELECT DISTINCT t.account_id
FROM debts d
JOIN debt_transactions dtx ON dtx.debt_id = d.id AND dtx.role = 'settle'
JOIN transactions t ON t.id = dtx.transaction_id
JOIN debt_transactions open_dtx ON open_dtx.debt_id = d.id AND open_dtx.role = 'open'
JOIN transactions open_t ON open_t.id = open_dtx.transaction_id
WHERE open_t.account_id = ? AND d.user_id = ? AND t.account_id != ?;

-- name: ListDebtsOpenedOnAccount :many
SELECT d.id, d.debtor_id
FROM debts d
JOIN debt_transactions dtx ON dtx.debt_id = d.id AND dtx.role = 'open'
JOIN transactions t ON t.id = dtx.transaction_id
WHERE t.account_id = ? AND d.user_id = ?;

-- name: UpdateDebtDebtor :exec
UPDATE debts SET debtor_id = ? WHERE id = ?;

-- name: ListBudgetsCatalogByAccount :many
SELECT id, category_id, subcategory_id FROM budgets WHERE account_id = ? AND user_id = ?;

-- name: UpdateBudgetCatalog :exec
UPDATE budgets SET category_id = ?, subcategory_id = ? WHERE id = ?;

-- name: ListSubscriptionsCatalogByAccount :many
SELECT id, subcategory_id FROM subscriptions WHERE account_id = ? AND user_id = ?;

-- name: UpdateSubscriptionCatalog :exec
UPDATE subscriptions SET subcategory_id = ? WHERE id = ?;

-- name: ListRecurringCatalogByAccount :many
SELECT id, category_id, subcategory_id FROM recurring_operations WHERE account_id = ? AND user_id = ?;

-- name: UpdateRecurringCatalog :exec
UPDATE recurring_operations SET category_id = ?, subcategory_id = ? WHERE id = ?;

-- name: ListTemplatesCatalogByAccount :many
SELECT id, category_id, subcategory_id, merchant_id FROM transaction_templates WHERE account_id = ? AND user_id = ?;

-- name: UpdateTemplateCatalog :exec
UPDATE transaction_templates SET category_id = ?, subcategory_id = ?, merchant_id = ? WHERE id = ?;

-- name: ListBudgetsWithoutAccount :many
SELECT id, category_id, subcategory_id FROM budgets WHERE user_id = ? AND account_id IS NULL;

-- name: ReassignBudgetsWithoutAccount :exec
UPDATE budgets SET user_id = ?, updated_at = ? WHERE user_id = ? AND account_id IS NULL;

-- name: ListCategoryRulesByOwner :many
SELECT id, category_id, subcategory_id FROM category_rules WHERE user_id = ?;

-- name: UpdateCategoryRuleCatalog :exec
UPDATE category_rules SET category_id = ?, subcategory_id = ? WHERE id = ?;

-- name: ReassignCategoryRulesOwner :exec
UPDATE category_rules SET user_id = ?, updated_at = ? WHERE user_id = ?;

-- name: ListTemplatesWithoutAccount :many
SELECT id, category_id, subcategory_id, merchant_id FROM transaction_templates WHERE user_id = ? AND account_id IS NULL;

-- name: ReassignTemplatesWithoutAccount :exec
UPDATE transaction_templates SET user_id = ?, updated_at = ? WHERE user_id = ? AND account_id IS NULL;

-- name: ListDebtsWithoutTransactions :many
SELECT d.id, d.debtor_id
FROM debts d
WHERE d.user_id = ? AND NOT EXISTS (SELECT 1 FROM debt_transactions dt WHERE dt.debt_id = d.id);

-- name: ReassignDebtOwner :exec
UPDATE debts SET user_id = ? WHERE id = ? AND user_id = ?;

-- name: ClearPersonalRuleDefaultsOnPrincipalAccounts :exec
UPDATE category_rules
SET default_account_id = NULL, updated_at = ?
WHERE category_rules.user_id = ? AND default_account_id IN (SELECT id FROM accounts WHERE accounts.user_id = ?);

-- name: ClearPersonalTemplateAccountsOnPrincipal :exec
UPDATE transaction_templates
SET account_id = NULL, updated_at = ?
WHERE transaction_templates.user_id = ? AND account_id IN (SELECT id FROM accounts WHERE accounts.user_id = ?);

-- name: ClearPersonalTemplateToAccountsOnPrincipal :exec
UPDATE transaction_templates
SET to_account_id = NULL, updated_at = ?
WHERE transaction_templates.user_id = ? AND to_account_id IN (SELECT id FROM accounts WHERE accounts.user_id = ?);
