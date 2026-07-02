-- name: GetAccountByID :one
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
    a.created_at,
    a.updated_at,
    b.name AS bank_name,
    b.icon_path AS bank_icon
FROM accounts a
LEFT JOIN banks b ON b.id = a.bank_id
WHERE a.id = ? AND a.user_id = ?;

-- name: ListAccountsByUserActive :many
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
    a.created_at,
    a.updated_at,
    b.name AS bank_name,
    b.icon_path AS bank_icon
FROM accounts a
LEFT JOIN banks b ON b.id = a.bank_id
WHERE a.user_id = ? AND a.status = 'active'
ORDER BY a.name;

-- name: ListAccountsByUserAndStatus :many
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
    a.created_at,
    a.updated_at,
    b.name AS bank_name,
    b.icon_path AS bank_icon
FROM accounts a
LEFT JOIN banks b ON b.id = a.bank_id
WHERE a.user_id = ? AND a.status = ?
ORDER BY a.name;

-- name: InsertAccount :exec
INSERT INTO accounts (
    id, user_id, name, type, bank_id, initial_balance, current_balance,
    credit_limit, payment_account_id, status, is_primary, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 'active', ?, ?, ?);

-- name: UpdateAccount :exec
UPDATE accounts
SET name = ?, bank_id = ?, initial_balance = ?, credit_limit = ?, payment_account_id = ?, updated_at = ?
WHERE id = ? AND user_id = ?;

-- name: UpdateAccountStatus :execrows
UPDATE accounts
SET status = ?, updated_at = ?
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
WHERE user_id = ? AND status = 'active'
ORDER BY created_at, name
LIMIT 1;

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
WHERE user_id = ?
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
