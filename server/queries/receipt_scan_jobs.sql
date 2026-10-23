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

-- name: GetDoneScanJobResultByFiscalKey :one
SELECT result_json
FROM receipt_scan_jobs
WHERE user_id = ?
  AND status = 'done'
  AND result_json IS NOT NULL
  AND TRIM(result_json) != ''
  AND json_extract(result_json, '$.fiscal_drive_number') = sqlc.arg(fiscal_drive_number)
  AND json_extract(result_json, '$.fiscal_document_number') = sqlc.arg(fiscal_document_number)
  AND json_extract(result_json, '$.fiscal_sign') = sqlc.arg(fiscal_sign)
ORDER BY finished_at DESC
LIMIT 1;
