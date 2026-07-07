-- name: InsertReceiptScanJob :exec
INSERT INTO receipt_scan_jobs (id, user_id, status, created_at, updated_at)
VALUES (?, ?, ?, ?, ?);

-- name: GetReceiptScanJob :one
SELECT id, status, error_message, result_json, created_at, started_at, finished_at
FROM receipt_scan_jobs
WHERE id = ? AND user_id = ?;

-- name: SetReceiptScanJobRunning :exec
UPDATE receipt_scan_jobs
SET status = ?, started_at = ?, updated_at = ?
WHERE id = ? AND user_id = ?;

-- name: SetReceiptScanJobDone :exec
UPDATE receipt_scan_jobs
SET status = ?, result_json = ?, error_message = NULL, finished_at = ?, updated_at = ?
WHERE id = ? AND user_id = ?;

-- name: SetReceiptScanJobFailed :exec
UPDATE receipt_scan_jobs
SET status = ?, error_message = ?, finished_at = ?, updated_at = ?
WHERE id = ? AND user_id = ?;

-- name: FailInterruptedReceiptScanJobs :execrows
UPDATE receipt_scan_jobs
SET status = ?, error_message = ?, finished_at = ?, updated_at = ?
WHERE status IN (?, ?) AND finished_at IS NULL;
