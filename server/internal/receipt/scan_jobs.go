package receipt

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
	"github.com/kai-zer-ru/buhgalter/internal/receipt/fns"
)

type ScanJobStatus string

const (
	ScanJobQueued  ScanJobStatus = "queued"
	ScanJobRunning ScanJobStatus = "running"
	ScanJobDone    ScanJobStatus = "done"
	ScanJobFailed  ScanJobStatus = "failed"
)

type ScanJob struct {
	ID           string             `json:"id"`
	Status       ScanJobStatus      `json:"status"`
	ErrorMessage *string            `json:"error_message,omitempty"`
	Result       *fns.ParsedReceipt `json:"result,omitempty"`
	CreatedAt    string             `json:"created_at"`
	StartedAt    *string            `json:"started_at,omitempty"`
	FinishedAt   *string            `json:"finished_at,omitempty"`
}

func createScanJobRecord(ctx context.Context, db *sql.DB, userID string) (ScanJob, error) {
	job := ScanJob{
		ID:     uuid.NewString(),
		Status: ScanJobQueued,
	}
	now := time.Now().UTC().Format(time.RFC3339)
	job.CreatedAt = now
	if err := sqlcdb.New(db).InsertReceiptScanJob(ctx, sqlcdb.InsertReceiptScanJobParams{
		ID:        job.ID,
		UserID:    userID,
		Status:    string(job.Status),
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		return ScanJob{}, err
	}
	return job, nil
}

func getScanJobRecord(ctx context.Context, db *sql.DB, userID, jobID string) (ScanJob, error) {
	row, err := sqlcdb.New(db).GetReceiptScanJob(ctx, sqlcdb.GetReceiptScanJobParams{ID: jobID, UserID: userID})
	if err != nil {
		return ScanJob{}, err
	}
	job, err := scanJobFromRow(row)
	if err != nil {
		return ScanJob{}, err
	}
	if job.Result != nil {
		applyLedgerFlags(ctx, db, userID, job.Result)
	}
	return job, nil
}

func setScanJobRunning(ctx context.Context, db *sql.DB, userID, jobID string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	return sqlcdb.New(db).SetReceiptScanJobRunning(ctx, sqlcdb.SetReceiptScanJobRunningParams{
		Status:    string(ScanJobRunning),
		StartedAt: &now,
		UpdatedAt: now,
		ID:        jobID,
		UserID:    userID,
	})
}

func setScanJobDone(ctx context.Context, db *sql.DB, userID, jobID string, result fns.ParsedReceipt) error {
	now := time.Now().UTC().Format(time.RFC3339)
	data, err := json.Marshal(result)
	if err != nil {
		return err
	}
	raw := string(data)
	return sqlcdb.New(db).SetReceiptScanJobDone(ctx, sqlcdb.SetReceiptScanJobDoneParams{
		Status:     string(ScanJobDone),
		ResultJson: &raw,
		FinishedAt: &now,
		UpdatedAt:  now,
		ID:         jobID,
		UserID:     userID,
	})
}

func setScanJobFailed(ctx context.Context, db *sql.DB, userID, jobID string, scanErr error) error {
	now := time.Now().UTC().Format(time.RFC3339)
	msg := "scan failed"
	if scanErr != nil {
		msg = scanErr.Error()
	}
	return sqlcdb.New(db).SetReceiptScanJobFailed(ctx, sqlcdb.SetReceiptScanJobFailedParams{
		Status:       string(ScanJobFailed),
		ErrorMessage: &msg,
		FinishedAt:   &now,
		UpdatedAt:    now,
		ID:           jobID,
		UserID:       userID,
	})
}

func scanJobFromRow(row sqlcdb.GetReceiptScanJobRow) (ScanJob, error) {
	job := ScanJob{
		ID:        row.ID,
		Status:    ScanJobStatus(row.Status),
		CreatedAt: row.CreatedAt,
	}
	if row.ErrorMessage != nil && *row.ErrorMessage != "" {
		job.ErrorMessage = row.ErrorMessage
	}
	if row.ResultJson != nil && *row.ResultJson != "" {
		var result fns.ParsedReceipt
		if err := json.Unmarshal([]byte(*row.ResultJson), &result); err != nil {
			return ScanJob{}, fmt.Errorf("decode result_json: %w", err)
		}
		job.Result = &result
	}
	if row.StartedAt != nil && *row.StartedAt != "" {
		job.StartedAt = row.StartedAt
	}
	if row.FinishedAt != nil && *row.FinishedAt != "" {
		job.FinishedAt = row.FinishedAt
	}
	return job, nil
}

func isScanJobNotFound(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}

// RecoverInterruptedScanJobs marks stale queued/running jobs as failed after restart.
func RecoverInterruptedScanJobs(ctx context.Context, db *sql.DB) (int64, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	msg := "scan interrupted: server restarted"
	return sqlcdb.New(db).FailInterruptedReceiptScanJobs(ctx, sqlcdb.FailInterruptedReceiptScanJobsParams{
		Status:       string(ScanJobFailed),
		ErrorMessage: &msg,
		FinishedAt:   &now,
		UpdatedAt:    now,
		Status_2:     string(ScanJobQueued),
		Status_3:     string(ScanJobRunning),
	})
}
