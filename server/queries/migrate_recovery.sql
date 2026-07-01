-- Migration recovery (partial 033 rebuild). See internal/db/db.go recoverInterruptedAccountRebuild.

-- name: CountSqliteTable :one
SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?;

-- name: RenameAccountsNewToAccounts :exec
ALTER TABLE accounts_new RENAME TO accounts;

-- name: DropAccountsNewTable :exec
DROP TABLE accounts_new;
