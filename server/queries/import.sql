-- name: GetImportIdempotency :one
SELECT response_json
FROM import_idempotency
WHERE user_id = ? AND idempotency_key = ?;

-- name: InsertImportIdempotency :exec
INSERT INTO import_idempotency (id, user_id, idempotency_key, response_json, created_at)
VALUES (?, ?, ?, ?, ?);

-- name: ListTransactionDedupRows :many
SELECT
    t.type,
    t.amount,
    substr(t.transaction_date, 1, 10) AS tx_date,
    a.name AS account_name,
    COALESCE(c.name, '') AS category_name
FROM transactions t
JOIN accounts a ON a.id = t.account_id
LEFT JOIN categories c ON c.id = t.category_id
WHERE t.user_id = ?
  AND t.type IN ('income', 'expense', 'transfer')
  AND (t.transfer_group_id IS NULL OR t.id = (
      SELECT x.id FROM transactions x
      WHERE x.transfer_group_id = t.transfer_group_id
      ORDER BY x.created_at ASC, x.id ASC
      LIMIT 1
  ));

-- name: InsertImportJob :exec
INSERT INTO import_jobs (id, user_id, filename, status, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?);

-- name: GetImportJob :one
SELECT id, filename, status, error_message, report_json, created_at, started_at, finished_at
FROM import_jobs WHERE id = ? AND user_id = ?;

-- name: SetImportJobRunning :exec
UPDATE import_jobs
SET status = ?, started_at = ?, updated_at = ?
WHERE id = ? AND user_id = ?;

-- name: SetImportJobDone :exec
UPDATE import_jobs
SET status = ?, report_json = ?, error_message = NULL, finished_at = ?, updated_at = ?
WHERE id = ? AND user_id = ?;

-- name: SetImportJobProgress :exec
UPDATE import_jobs
SET report_json = ?, updated_at = ?
WHERE id = ? AND user_id = ?;

-- name: SetImportJobFailed :exec
UPDATE import_jobs
SET status = ?, error_message = ?, finished_at = ?, updated_at = ?
WHERE id = ? AND user_id = ?;

-- name: FailInterruptedImportJobs :execrows
UPDATE import_jobs
SET status = ?, error_message = ?, finished_at = ?, updated_at = ?
WHERE status IN (?, ?) AND finished_at IS NULL;

