-- name: GetAccountOwnerDisplay :one
SELECT a.user_id, COALESCE(u.display_name, '') AS display_name
FROM accounts a
JOIN users u ON u.id = a.user_id
WHERE a.id = ?;

-- name: GetAccountByID :one
SELECT
    a.id,
    a.user_id,
    a.original_owner_user_id,
    COALESCE(ou.display_name, '') AS original_owner_display_name,
    a.name,
    a.type,
    a.bank_id,
    a.initial_balance,
    a.current_balance,
    a.credit_limit,
    a.payment_account_id,
    a.auto_topup_enabled,
    a.auto_topup_threshold,
    a.auto_topup_target,
    a.auto_topup_source_account_id,
    a.status,
    a.is_primary,
    a.currency,
    a.created_at,
    a.updated_at,
    b.name AS bank_name,
    b.icon_path AS bank_icon
FROM accounts a
LEFT JOIN banks b ON b.id = a.bank_id
LEFT JOIN users ou ON ou.id = a.original_owner_user_id
WHERE a.id = sqlc.arg('id') AND a.user_id IN (sqlc.slice('owner_ids'));

-- name: ListAccountsByUserActive :many
SELECT
    a.id,
    a.user_id,
    a.original_owner_user_id,
    COALESCE(ou.display_name, '') AS original_owner_display_name,
    a.name,
    a.type,
    a.bank_id,
    a.initial_balance,
    a.current_balance,
    a.credit_limit,
    a.payment_account_id,
    a.auto_topup_enabled,
    a.auto_topup_threshold,
    a.auto_topup_target,
    a.auto_topup_source_account_id,
    a.status,
    a.is_primary,
    a.currency,
    a.created_at,
    a.updated_at,
    b.name AS bank_name,
    b.icon_path AS bank_icon
FROM accounts a
LEFT JOIN banks b ON b.id = a.bank_id
LEFT JOIN users ou ON ou.id = a.original_owner_user_id
WHERE a.user_id IN (sqlc.slice('owner_ids')) AND a.status = 'active'
ORDER BY a.name;

-- name: ListAccountsForExport :many
SELECT
    a.id,
    a.name,
    a.type,
    a.bank_id,
    a.initial_balance,
    a.credit_limit,
    a.payment_account_id,
    pa.name AS payment_account_name,
    a.auto_topup_enabled,
    a.auto_topup_threshold,
    a.auto_topup_target,
    a.auto_topup_source_account_id,
    src.name AS auto_topup_source_name,
    a.status,
    a.is_primary,
    a.currency,
    b.name AS bank_name
FROM accounts a
LEFT JOIN banks b ON b.id = a.bank_id
LEFT JOIN accounts pa ON pa.id = a.payment_account_id
LEFT JOIN accounts src ON src.id = a.auto_topup_source_account_id
WHERE a.user_id = ? AND a.status IN ('active', 'archived')
ORDER BY CASE a.type WHEN 'cash' THEN 0 WHEN 'bank' THEN 1 ELSE 2 END, a.name;

-- name: ListAccountsByUserAndStatus :many
SELECT
    a.id,
    a.user_id,
    a.original_owner_user_id,
    COALESCE(ou.display_name, '') AS original_owner_display_name,
    a.name,
    a.type,
    a.bank_id,
    a.initial_balance,
    a.current_balance,
    a.credit_limit,
    a.payment_account_id,
    a.auto_topup_enabled,
    a.auto_topup_threshold,
    a.auto_topup_target,
    a.auto_topup_source_account_id,
    a.status,
    a.is_primary,
    a.currency,
    a.created_at,
    a.updated_at,
    b.name AS bank_name,
    b.icon_path AS bank_icon
FROM accounts a
LEFT JOIN banks b ON b.id = a.bank_id
LEFT JOIN users ou ON ou.id = a.original_owner_user_id
WHERE a.status = sqlc.arg('status') AND a.user_id IN (sqlc.slice('owner_ids'))
ORDER BY a.name;

-- name: InsertAccount :exec
INSERT INTO accounts (
    id, user_id, name, type, bank_id, initial_balance, current_balance,
    credit_limit, payment_account_id, status, is_primary, currency, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 'active', ?, ?, ?, ?);

-- name: UpdateAccount :exec
UPDATE accounts
SET name = ?, bank_id = ?, initial_balance = ?, credit_limit = ?, payment_account_id = ?, updated_at = ?
WHERE id = ? AND user_id = ?;

-- name: UpdateAccountCurrency :exec
UPDATE accounts
SET currency = ?, updated_at = ?
WHERE id = ? AND user_id = ?;

-- name: AccountCurrencyInUse :one
SELECT CAST(
    EXISTS (
        SELECT 1 FROM transactions t
        WHERE t.user_id = sqlc.arg(user_id)
          AND (t.account_id = sqlc.arg(account_id) OR t.transfer_account_id = sqlc.arg(account_id))
    )
    OR EXISTS (SELECT 1 FROM credits c WHERE c.debit_account_id = sqlc.arg(account_id))
    OR EXISTS (SELECT 1 FROM subscriptions s WHERE s.account_id = sqlc.arg(account_id))
    OR EXISTS (SELECT 1 FROM recurring_operations r WHERE r.account_id = sqlc.arg(account_id))
    OR EXISTS (SELECT 1 FROM budgets b WHERE b.account_id = sqlc.arg(account_id))
    OR EXISTS (
        SELECT 1 FROM accounts x
        WHERE x.id != sqlc.arg(account_id)
          AND (x.payment_account_id = sqlc.arg(account_id) OR x.auto_topup_source_account_id = sqlc.arg(account_id))
    )
AS INTEGER) AS in_use;

-- name: UpdateAccountStatus :execrows
UPDATE accounts
SET status = ?, updated_at = ?
WHERE id = ? AND user_id = ?;

-- name: LockAccountRow :execrows
UPDATE accounts
SET updated_at = updated_at
WHERE id = ? AND user_id = ?;

-- name: CountActiveAccountsByName :one
SELECT COUNT(*) AS count
FROM accounts
WHERE user_id = ? AND name = ? AND status = 'active';

-- name: CountActiveAccountsByNameExcluding :one
SELECT COUNT(*) AS count
FROM accounts
WHERE user_id = ? AND name = ? AND status = 'active' AND id != ?;

-- name: CountActiveAccountsByUser :one
SELECT COUNT(*) AS count
FROM accounts
WHERE user_id = ? AND status = 'active';

-- name: ClearPrimaryAccounts :exec
UPDATE accounts
SET is_primary = 0
WHERE user_id = ? AND status = 'active';

-- name: SetAccountPrimary :exec
UPDATE accounts
SET is_primary = 1
WHERE id = ? AND user_id = ? AND status = 'active';

-- name: ClearAccountPrimaryFlag :exec
UPDATE accounts
SET is_primary = 0
WHERE id = ? AND user_id = ?;

-- name: FirstActiveAccountID :one
SELECT id
FROM accounts
WHERE user_id = ? AND status = 'active' AND type IN ('cash', 'bank')
ORDER BY created_at, name
LIMIT 1;

-- name: CountActivePrimaryEligibleAccountsByUser :one
SELECT COUNT(*) AS count
FROM accounts
WHERE user_id = ? AND status = 'active' AND type IN ('cash', 'bank');

-- name: GetActiveAccountByName :one
SELECT
    a.id,
    a.name,
    a.type,
    a.bank_id,
    a.initial_balance,
    a.current_balance,
    a.credit_limit,
    a.payment_account_id,
    a.auto_topup_enabled,
    a.auto_topup_threshold,
    a.auto_topup_target,
    a.auto_topup_source_account_id,
    a.status,
    a.is_primary,
    a.currency,
    a.created_at,
    a.updated_at,
    b.name AS bank_name,
    b.icon_path AS bank_icon
FROM accounts a
LEFT JOIN banks b ON b.id = a.bank_id
WHERE a.user_id = ? AND a.status = 'active' AND a.name = ?
LIMIT 1;

-- name: ListActiveAccountNames :many
SELECT name
FROM accounts
WHERE user_id = ? AND status = 'active';

-- name: ListAccountRefsByUser :many
SELECT id, name, type, status, bank_id
FROM accounts
WHERE user_id IN (sqlc.slice('owner_ids'))
ORDER BY name;

-- name: UpdateAccountCurrentBalance :exec
UPDATE accounts
SET current_balance = ?, updated_at = ?
WHERE id = ? AND user_id = ?;

-- name: ListAllAccountIDsByUser :many
SELECT id, initial_balance
FROM accounts
WHERE user_id = ?;

-- name: ListDistinctAccountUserIDs :many
SELECT DISTINCT user_id FROM accounts;

-- name: ListAutoTopupBeneficiaryAccountIDs :many
SELECT id
FROM accounts
WHERE user_id = ?
  AND status = 'active'
  AND type = 'bank'
  AND auto_topup_enabled = 1;

-- name: UpdateAccountAutoTopup :exec
UPDATE accounts
SET
    auto_topup_enabled = ?,
    auto_topup_threshold = ?,
    auto_topup_target = ?,
    auto_topup_source_account_id = ?,
    updated_at = ?
WHERE id = ? AND user_id = ?;

-- name: DisableAutoTopup :exec
UPDATE accounts
SET auto_topup_enabled = 0, updated_at = ?
WHERE id = ? AND user_id = ?;

-- name: DisableAutoTopupUsingSource :exec
UPDATE accounts
SET auto_topup_enabled = 0, updated_at = ?
WHERE user_id = ?
  AND auto_topup_enabled = 1
  AND auto_topup_source_account_id = ?;

-- name: DisableAutoTopupForBeneficiary :exec
UPDATE accounts
SET auto_topup_enabled = 0, updated_at = ?
WHERE id = ? AND user_id = ?;
