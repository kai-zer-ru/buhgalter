-- Wipe financial data for one user (keep profile, sessions, API tokens).
-- Household dissolve also deletes notification_settings / templates leftover on the principal.

-- name: DeleteUserCreditPayments :exec
DELETE FROM credit_payments
WHERE credit_id IN (SELECT id FROM credits WHERE user_id = ?);

-- name: DeleteUserDebtTransactions :exec
DELETE FROM debt_transactions
WHERE debt_id IN (SELECT id FROM debts WHERE user_id = ?);

-- name: DeleteUserCredits :exec
DELETE FROM credits WHERE user_id = ?;

-- name: DeleteUserDebts :exec
DELETE FROM debts WHERE user_id = ?;

-- name: DeleteUserDebtors :exec
DELETE FROM debtors WHERE user_id = ?;

-- name: DeleteUserSubscriptions :exec
DELETE FROM subscriptions WHERE user_id = ?;

-- name: DeleteUserRecurringOperations :exec
DELETE FROM recurring_operations WHERE user_id = ?;

-- name: DeleteUserBudgets :exec
DELETE FROM budgets WHERE user_id = ?;

-- name: DeleteUserTransactionTemplates :exec
DELETE FROM transaction_templates WHERE user_id = ?;

-- name: DeleteUserCategoryRules :exec
DELETE FROM category_rules WHERE user_id = ?;

-- name: DeleteUserTransactions :exec
DELETE FROM transactions WHERE user_id = ?;

-- name: DeleteUserMerchants :exec
DELETE FROM merchants WHERE user_id = ?;

-- name: DeleteUserTags :exec
DELETE FROM tags WHERE user_id = ?;

-- name: ClearUserAccountSelfRefs :exec
UPDATE accounts
SET payment_account_id = NULL,
    auto_topup_source_account_id = NULL
WHERE user_id = ?;

-- name: DeleteUserAccounts :exec
DELETE FROM accounts WHERE user_id = ?;

-- name: DeleteUserCategories :exec
DELETE FROM categories WHERE user_id = ?;

-- name: DeleteUserImportJobs :exec
DELETE FROM import_jobs WHERE user_id = ?;

-- name: DeleteUserImportIdempotency :exec
DELETE FROM import_idempotency WHERE user_id = ?;

-- name: DeleteUserChangeEvents :exec
DELETE FROM user_change_events WHERE user_id = ?;

-- name: DeleteUserNotificationLog :exec
DELETE FROM notification_log WHERE user_id = ?;

-- name: DeleteUserNotificationSettings :exec
DELETE FROM notification_settings WHERE user_id = ?;

-- name: DeleteUserNotificationTemplates :exec
DELETE FROM notification_templates WHERE user_id = ?;

-- name: DeleteUserReceiptImportItemCategories :exec
DELETE FROM receipt_import_item_categories WHERE user_id = ?;

-- name: DeleteUserStores :exec
DELETE FROM stores WHERE user_id = ?;

-- name: DeleteUserReceiptImports :exec
DELETE FROM receipt_imports WHERE user_id = ?;

-- name: DeleteUserReceiptScanJobs :exec
DELETE FROM receipt_scan_jobs WHERE user_id = ?;

-- name: DeleteUserMerchantAliases :exec
DELETE FROM merchant_aliases WHERE user_id = ?;

-- name: DeleteUserReceiptFiscalPreviews :exec
DELETE FROM receipt_fiscal_previews WHERE user_id = ?;
