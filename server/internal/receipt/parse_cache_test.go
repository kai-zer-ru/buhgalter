package receipt_test

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kai-zer-ru/buhgalter/internal/db"
	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
	"github.com/kai-zer-ru/buhgalter/internal/receipt"
	"github.com/kai-zer-ru/buhgalter/internal/receipt/fns"
)

type countingFetcher struct {
	calls int
}

func (c *countingFetcher) FetchReceipt(_ context.Context, _ string) (fns.ParsedReceipt, error) {
	c.calls++
	return fns.ParsedReceipt{
		FiscalDriveNumber:    "7288940700123456",
		FiscalDocumentNumber: "12345",
		FiscalSign:           "987654321",
		StoreName:            "Cached Store",
		TotalSum:             10000,
		Items:                []fns.Item{{Name: "x", Quantity: 1, Price: 10000, Sum: 10000}},
	}, nil
}

func TestParseForUserUsesFiscalPreviewCache(t *testing.T) {
	dir := t.TempDir()
	mgr, err := db.NewManager(filepath.Join(dir, "cache.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer mgr.Close()
	ctx := context.Background()
	userID := uuid.NewString()
	if _, err := mgr.DB().ExecContext(ctx,
		`INSERT INTO users (id, login, password_hash, timezone) VALUES (?, 'cacheuser', 'hash', 'UTC')`,
		userID,
	); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Format(time.RFC3339)

	cached := fns.ParsedReceipt{
		FiscalDriveNumber:    "7288940700123456",
		FiscalDocumentNumber: "12345",
		FiscalSign:           "987654321",
		StoreName:            "Cached Store",
		TotalSum:             10000,
		Items:                []fns.Item{{Name: "x", Quantity: 1, Price: 10000, Sum: 10000}},
	}
	raw, err := json.Marshal(cached)
	if err != nil {
		t.Fatal(err)
	}
	q := sqlcdb.New(mgr.DB())
	if err := q.UpsertReceiptFiscalPreview(ctx, sqlcdb.UpsertReceiptFiscalPreviewParams{
		UserID:               userID,
		FiscalDriveNumber:    cached.FiscalDriveNumber,
		FiscalDocumentNumber: cached.FiscalDocumentNumber,
		FiscalSign:           cached.FiscalSign,
		ResultJson:           string(raw),
		FetchedAt:            now,
	}); err != nil {
		t.Fatal(err)
	}

	qr := "t=20260706T1753&s=4047.50&fn=" + cached.FiscalDriveNumber +
		"&i=" + cached.FiscalDocumentNumber + "&fp=" + cached.FiscalSign + "&n=1"
	fetcher := &countingFetcher{}
	out, err := receipt.ParseForUser(ctx, mgr.DB(), userID, qr, fetcher)
	if err != nil {
		t.Fatal(err)
	}
	if out.StoreName != cached.StoreName {
		t.Fatalf("store: got %q want %q", out.StoreName, cached.StoreName)
	}
	if fetcher.calls != 0 {
		t.Fatalf("expected no external fetch, got %d calls", fetcher.calls)
	}
}

func TestParseForUserPersistsProviderResponse(t *testing.T) {
	dir := t.TempDir()
	mgr, err := db.NewManager(filepath.Join(dir, "persist.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer mgr.Close()
	ctx := context.Background()
	userID := uuid.NewString()
	if _, err := mgr.DB().ExecContext(ctx,
		`INSERT INTO users (id, login, password_hash, timezone) VALUES (?, 'persistuser', 'hash', 'UTC')`,
		userID,
	); err != nil {
		t.Fatal(err)
	}

	qr := "t=20260706T1753&s=4047.50&fn=7288940700123456&i=12345&fp=987654321&n=1"
	fetcher := &countingFetcher{}
	if _, err := receipt.ParseForUser(ctx, mgr.DB(), userID, qr, fetcher); err != nil {
		t.Fatal(err)
	}
	if fetcher.calls != 1 {
		t.Fatalf("expected 1 provider call, got %d", fetcher.calls)
	}
	if _, err := receipt.ParseForUser(ctx, mgr.DB(), userID, qr, fetcher); err != nil {
		t.Fatal(err)
	}
	if fetcher.calls != 1 {
		t.Fatalf("expected cached second parse, provider calls %d", fetcher.calls)
	}
}
