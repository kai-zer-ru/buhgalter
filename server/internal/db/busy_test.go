package db_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/kai-zer-ru/buhgalter/internal/db"
)

func TestIsBusy(t *testing.T) {
	if !db.IsBusy(fmt.Errorf("database is locked")) {
		t.Fatal("expected busy")
	}
	if !db.IsBusy(fmt.Errorf("SQLITE_BUSY: snapshot in use")) {
		t.Fatal("expected busy")
	}
	if db.IsBusy(errors.New("other")) {
		t.Fatal("expected not busy")
	}
}

func TestWithBusyRetryEventuallySucceeds(t *testing.T) {
	var calls int
	err := db.WithBusyRetry(context.Background(), 5, func() error {
		calls++
		if calls < 3 {
			return fmt.Errorf("database is locked")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if calls != 3 {
		t.Fatalf("expected 3 calls, got %d", calls)
	}
}

func TestOpenSetsBusyTimeout(t *testing.T) {
	dir := t.TempDir()
	sqlDB, err := db.Open(dir + "/busy.db")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer sqlDB.Close()

	var timeout int
	if err := sqlDB.QueryRow(`PRAGMA busy_timeout`).Scan(&timeout); err != nil {
		t.Fatalf("pragma: %v", err)
	}
	if timeout != 10000 {
		t.Fatalf("expected busy_timeout=10000, got %d", timeout)
	}
}

func TestConcurrentReadsWithBusyRetry(t *testing.T) {
	dir := t.TempDir()
	sqlDB, err := db.Open(dir + "/concurrent.db")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer sqlDB.Close()

	_, err = sqlDB.Exec(`UPDATE system_settings SET is_configured = 1 WHERE id = 1`)
	if err != nil {
		t.Fatalf("configure: %v", err)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		tx, err := sqlDB.Begin()
		if err != nil {
			return
		}
		_, _ = tx.Exec(`UPDATE system_settings SET updated_at = datetime('now') WHERE id = 1`)
		time.Sleep(200 * time.Millisecond)
		_ = tx.Commit()
	}()

	var wg sync.WaitGroup
	errCh := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := db.WithBusyRetry(context.Background(), 8, func() error {
				var configured int
				return sqlDB.QueryRow(`SELECT is_configured FROM system_settings WHERE id = 1`).Scan(&configured)
			})
			if err != nil {
				errCh <- err
			}
		}()
	}
	wg.Wait()
	<-done
	close(errCh)
	for err := range errCh {
		t.Fatalf("concurrent read failed: %v", err)
	}
}
