package importexport

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
)

type ImportJobStatus string

const (
	ImportJobQueued  ImportJobStatus = "queued"
	ImportJobRunning ImportJobStatus = "running"
	ImportJobDone    ImportJobStatus = "done"
	ImportJobFailed  ImportJobStatus = "failed"
)

type ImportJob struct {
	ID         string          `json:"id"`
	Filename   string          `json:"filename"`
	Status     ImportJobStatus `json:"status"`
	Error      *string         `json:"error_message,omitempty"`
	Report     *Report         `json:"report,omitempty"`
	CreatedAt  string          `json:"created_at"`
	StartedAt  *string         `json:"started_at,omitempty"`
	FinishedAt *string         `json:"finished_at,omitempty"`
}

func createImportJobRecord(ctx context.Context, db *sql.DB, userID, filename string) (ImportJob, error) {
	job := ImportJob{
		ID:       uuid.NewString(),
		Filename: filename,
		Status:   ImportJobQueued,
	}
	now := time.Now().UTC().Format(time.RFC3339)
	job.CreatedAt = now
	if err := sqlcdb.New(db).InsertImportJob(ctx, sqlcdb.InsertImportJobParams{
		ID:        job.ID,
		UserID:    userID,
		Filename:  filename,
		Status:    string(job.Status),
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		return ImportJob{}, err
	}
	return job, nil
}

func getImportJobRecord(ctx context.Context, db *sql.DB, userID, jobID string) (ImportJob, error) {
	row, err := sqlcdb.New(db).GetImportJob(ctx, sqlcdb.GetImportJobParams{ID: jobID, UserID: userID})
	if err != nil {
		return ImportJob{}, err
	}
	return importJobFromRow(row)
}

func setImportJobRunning(ctx context.Context, db *sql.DB, userID, jobID string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	return sqlcdb.New(db).SetImportJobRunning(ctx, sqlcdb.SetImportJobRunningParams{
		Status:    string(ImportJobRunning),
		StartedAt: &now,
		UpdatedAt: now,
		ID:        jobID,
		UserID:    userID,
	})
}

func setImportJobDone(ctx context.Context, db *sql.DB, userID, jobID string, report Report) error {
	now := time.Now().UTC().Format(time.RFC3339)
	data, err := json.Marshal(report)
	if err != nil {
		return err
	}
	reportJSON := string(data)
	return sqlcdb.New(db).SetImportJobDone(ctx, sqlcdb.SetImportJobDoneParams{
		Status:     string(ImportJobDone),
		ReportJson: &reportJSON,
		FinishedAt: &now,
		UpdatedAt:  now,
		ID:         jobID,
		UserID:     userID,
	})
}

func setImportJobProgress(ctx context.Context, db *sql.DB, userID, jobID string, report Report) error {
	now := time.Now().UTC().Format(time.RFC3339)
	data, err := json.Marshal(report)
	if err != nil {
		return err
	}
	reportJSON := string(data)
	return sqlcdb.New(db).SetImportJobProgress(ctx, sqlcdb.SetImportJobProgressParams{
		ReportJson: &reportJSON,
		UpdatedAt:  now,
		ID:         jobID,
		UserID:     userID,
	})
}

func setImportJobFailed(ctx context.Context, db *sql.DB, userID, jobID string, importErr error) error {
	now := time.Now().UTC().Format(time.RFC3339)
	msg := "import failed"
	if importErr != nil {
		msg = importErr.Error()
	}
	return sqlcdb.New(db).SetImportJobFailed(ctx, sqlcdb.SetImportJobFailedParams{
		Status:       string(ImportJobFailed),
		ErrorMessage: &msg,
		FinishedAt:   &now,
		UpdatedAt:    now,
		ID:           jobID,
		UserID:       userID,
	})
}

func failInterruptedJobs(ctx context.Context, db *sql.DB) (int64, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	msg := "import interrupted: server restarted"
	return sqlcdb.New(db).FailInterruptedImportJobs(ctx, sqlcdb.FailInterruptedImportJobsParams{
		Status:       string(ImportJobFailed),
		ErrorMessage: &msg,
		FinishedAt:   &now,
		UpdatedAt:    now,
		Status_2:     string(ImportJobQueued),
		Status_3:     string(ImportJobRunning),
	})
}

// RecoverInterruptedJobs marks stale queued/running jobs as failed after restart.
func RecoverInterruptedJobs(ctx context.Context, db *sql.DB) (int64, error) {
	return failInterruptedJobs(ctx, db)
}

func isNotFound(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}

func importJobFromRow(row sqlcdb.GetImportJobRow) (ImportJob, error) {
	job := ImportJob{
		ID:        row.ID,
		Filename:  row.Filename,
		Status:    ImportJobStatus(row.Status),
		CreatedAt: row.CreatedAt,
	}
	if row.ErrorMessage != nil && *row.ErrorMessage != "" {
		msg := *row.ErrorMessage
		job.Error = &msg
	}
	if row.ReportJson != nil && *row.ReportJson != "" {
		var rep Report
		if err := json.Unmarshal([]byte(*row.ReportJson), &rep); err != nil {
			return ImportJob{}, fmt.Errorf("decode report_json: %w", err)
		}
		job.Report = &rep
	}
	if row.StartedAt != nil && *row.StartedAt != "" {
		job.StartedAt = row.StartedAt
	}
	if row.FinishedAt != nil && *row.FinishedAt != "" {
		job.FinishedAt = row.FinishedAt
	}
	return job, nil
}
