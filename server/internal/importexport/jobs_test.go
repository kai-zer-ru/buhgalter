package importexport

import (
	"database/sql"
	"fmt"
	"testing"
)

func TestRecoverInterruptedJobs(t *testing.T) {
	ctx, sqlDB, userID := seedImportUser(t)
	now := "2026-01-01 00:00:00"
	_, err := sqlDB.ExecContext(ctx, `
		INSERT INTO import_jobs (id, user_id, filename, status, created_at, updated_at, started_at)
		VALUES ('job-1', ?, 'stale.csv', 'running', ?, ?, ?)`,
		userID, now, now, now)
	if err != nil {
		t.Fatal(err)
	}
	n, err := RecoverInterruptedJobs(ctx, sqlDB)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("recovered %d", n)
	}
}

func TestImportJobLifecycle(t *testing.T) {
	ctx, sqlDB, userID := seedImportUser(t)
	job, err := createImportJobRecord(ctx, sqlDB, userID, "data.csv")
	if err != nil {
		t.Fatal(err)
	}
	if job.Status != ImportJobQueued {
		t.Fatalf("status %s", job.Status)
	}
	if err := setImportJobRunning(ctx, sqlDB, userID, job.ID); err != nil {
		t.Fatal(err)
	}
	progress := Report{TotalRows: 4, ValidRows: 4, ProcessedRows: 2}
	if err := setImportJobProgress(ctx, sqlDB, userID, job.ID, progress); err != nil {
		t.Fatal(err)
	}
	done := Report{TotalRows: 4, ValidRows: 4, CreatedTransactions: 4}
	if err := setImportJobDone(ctx, sqlDB, userID, job.ID, done); err != nil {
		t.Fatal(err)
	}
	got, err := getImportJobRecord(ctx, sqlDB, userID, job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != ImportJobDone || got.Report == nil || got.Report.CreatedTransactions != 4 {
		t.Fatalf("job %+v", got)
	}

	failJob, err := createImportJobRecord(ctx, sqlDB, userID, "bad.csv")
	if err != nil {
		t.Fatal(err)
	}
	if err := setImportJobFailed(ctx, sqlDB, userID, failJob.ID, fmt.Errorf("parse error")); err != nil {
		t.Fatal(err)
	}
	failed, err := getImportJobRecord(ctx, sqlDB, userID, failJob.ID)
	if err != nil {
		t.Fatal(err)
	}
	if failed.Status != ImportJobFailed || failed.Error == nil {
		t.Fatalf("failed job %+v", failed)
	}
	if !isNotFound(fmt.Errorf("wrap: %w", sql.ErrNoRows)) {
		t.Fatal("isNotFound should detect sql.ErrNoRows")
	}
	_, err = getImportJobRecord(ctx, sqlDB, userID, "missing")
	if !isNotFound(err) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestNormalizeHeaderAndFormatCubuxAmount(t *testing.T) {
	if got := NormalizeHeader("  Сумма "); got != "Сумма" {
		t.Fatalf("header %q", got)
	}
	if got := FormatCubuxAmount(5050); got == "" {
		t.Fatal("expected amount string")
	}
}
