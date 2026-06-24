package importexport

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
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
	_, err := db.ExecContext(ctx, `
		INSERT INTO import_jobs (
			id, user_id, filename, status, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?)
	`, job.ID, userID, filename, string(job.Status), now, now)
	if err != nil {
		return ImportJob{}, err
	}
	return job, nil
}

func getImportJobRecord(ctx context.Context, db *sql.DB, userID, jobID string) (ImportJob, error) {
	row := db.QueryRowContext(ctx, `
		SELECT id, filename, status, error_message, report_json, created_at, started_at, finished_at
		FROM import_jobs
		WHERE id = ? AND user_id = ?
	`, jobID, userID)

	var (
		job        ImportJob
		statusRaw  string
		errMsg     sql.NullString
		reportRaw  sql.NullString
		startedAt  sql.NullString
		finishedAt sql.NullString
	)
	if err := row.Scan(
		&job.ID,
		&job.Filename,
		&statusRaw,
		&errMsg,
		&reportRaw,
		&job.CreatedAt,
		&startedAt,
		&finishedAt,
	); err != nil {
		return ImportJob{}, err
	}
	job.Status = ImportJobStatus(statusRaw)
	if errMsg.Valid && errMsg.String != "" {
		msg := errMsg.String
		job.Error = &msg
	}
	if reportRaw.Valid && reportRaw.String != "" {
		var rep Report
		if err := json.Unmarshal([]byte(reportRaw.String), &rep); err != nil {
			return ImportJob{}, fmt.Errorf("decode report_json: %w", err)
		}
		job.Report = &rep
	}
	if startedAt.Valid && startedAt.String != "" {
		v := startedAt.String
		job.StartedAt = &v
	}
	if finishedAt.Valid && finishedAt.String != "" {
		v := finishedAt.String
		job.FinishedAt = &v
	}
	return job, nil
}

func setImportJobRunning(ctx context.Context, db *sql.DB, userID, jobID string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := db.ExecContext(ctx, `
		UPDATE import_jobs
		SET status = ?, started_at = ?, updated_at = ?
		WHERE id = ? AND user_id = ?
	`, string(ImportJobRunning), now, now, jobID, userID)
	return err
}

func setImportJobDone(ctx context.Context, db *sql.DB, userID, jobID string, report Report) error {
	now := time.Now().UTC().Format(time.RFC3339)
	data, err := json.Marshal(report)
	if err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, `
		UPDATE import_jobs
		SET status = ?, report_json = ?, error_message = NULL, finished_at = ?, updated_at = ?
		WHERE id = ? AND user_id = ?
	`, string(ImportJobDone), string(data), now, now, jobID, userID)
	return err
}

func setImportJobProgress(ctx context.Context, db *sql.DB, userID, jobID string, report Report) error {
	now := time.Now().UTC().Format(time.RFC3339)
	data, err := json.Marshal(report)
	if err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, `
		UPDATE import_jobs
		SET report_json = ?, updated_at = ?
		WHERE id = ? AND user_id = ?
	`, string(data), now, jobID, userID)
	return err
}

func setImportJobFailed(ctx context.Context, db *sql.DB, userID, jobID string, importErr error) error {
	now := time.Now().UTC().Format(time.RFC3339)
	msg := "import failed"
	if importErr != nil {
		msg = importErr.Error()
	}
	_, err := db.ExecContext(ctx, `
		UPDATE import_jobs
		SET status = ?, error_message = ?, finished_at = ?, updated_at = ?
		WHERE id = ? AND user_id = ?
	`, string(ImportJobFailed), msg, now, now, jobID, userID)
	return err
}

func failInterruptedJobs(ctx context.Context, db *sql.DB) (int64, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := db.ExecContext(ctx, `
		UPDATE import_jobs
		SET status = ?, error_message = ?, finished_at = ?, updated_at = ?
		WHERE status IN (?, ?) AND finished_at IS NULL
	`, string(ImportJobFailed), "import interrupted: server restarted", now, now, string(ImportJobQueued), string(ImportJobRunning))
	if err != nil {
		return 0, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}
	return affected, nil
}

// RecoverInterruptedJobs marks stale queued/running jobs as failed after restart.
func RecoverInterruptedJobs(ctx context.Context, db *sql.DB) (int64, error) {
	return failInterruptedJobs(ctx, db)
}

func isNotFound(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}
