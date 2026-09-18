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
    CAST(COALESCE(substr(t.transaction_date, 12, 8), '') AS TEXT) AS tx_time,
    a.name AS account_name,
    COALESCE(ta.name, '') AS transfer_account_name,
    COALESCE(c.name, '') AS category_name,
    COALESCE(s.name, '') AS subcategory_name,
    COALESCE(t.description, '') AS description,
    COALESCE(m.name, '') AS merchant_name,
    CAST(COALESCE((
        SELECT group_concat(tg.name, ',')
        FROM transaction_tags tt
        JOIN tags tg ON tg.id = tt.tag_id
        WHERE tt.transaction_id = t.id
    ), '') AS TEXT) AS tag_names,
    CASE
        WHEN t.type = 'transfer' AND t.transfer_group_id IS NOT NULL THEN (
            SELECT COALESCE(SUM(x.amount), 0)
            FROM transactions x
            WHERE x.transfer_group_id = t.transfer_group_id
              AND x.type = 'expense'
        )
        ELSE 0
    END AS commission
FROM transactions t
JOIN accounts a ON a.id = t.account_id
LEFT JOIN accounts ta ON ta.id = t.transfer_account_id
LEFT JOIN categories c ON c.id = t.category_id
LEFT JOIN subcategories s ON s.id = t.subcategory_id
LEFT JOIN merchants m ON m.id = t.merchant_id
WHERE t.user_id = ?
  AND t.type IN ('income', 'expense', 'transfer')
  AND NOT (t.type = 'expense' AND t.transfer_group_id IS NOT NULL)
  AND (t.transfer_group_id IS NULL OR t.id = (
      SELECT x.id FROM transactions x
      WHERE x.transfer_group_id = t.transfer_group_id
        AND x.type = 'transfer'
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

