package receipt

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
	"github.com/kai-zer-ru/buhgalter/internal/receipt/fns"
)

// ParseForUser resolves receipt preview: DB cache first, then external API.
func ParseForUser(ctx context.Context, db *sql.DB, userID, qr string, fetcher Fetcher) (fns.ParsedReceipt, error) {
	params, err := fns.ParseQR(qr)
	if err != nil {
		return fns.ParsedReceipt{}, err
	}
	if out, ok, err := previewFromFiscalCache(ctx, db, userID, params); err != nil {
		return fns.ParsedReceipt{}, err
	} else if ok {
		return finishParsedReceipt(ctx, db, userID, out), nil
	}
	if out, ok, err := previewFromReceiptImport(ctx, db, userID, params); err != nil {
		return fns.ParsedReceipt{}, err
	} else if ok {
		_ = saveFiscalPreview(ctx, db, userID, params, out)
		return finishParsedReceipt(ctx, db, userID, out), nil
	}
	if out, ok, err := previewFromScanJob(ctx, db, userID, params); err != nil {
		return fns.ParsedReceipt{}, err
	} else if ok {
		_ = saveFiscalPreview(ctx, db, userID, params, out)
		return finishParsedReceipt(ctx, db, userID, out), nil
	}
	out, err := fetcher.FetchReceipt(ctx, qr)
	if err != nil {
		return fns.ParsedReceipt{}, err
	}
	if err := saveFiscalPreview(ctx, db, userID, params, out); err != nil {
		return fns.ParsedReceipt{}, err
	}
	return finishParsedReceipt(ctx, db, userID, out), nil
}

func finishParsedReceipt(ctx context.Context, db *sql.DB, userID string, out fns.ParsedReceipt) fns.ParsedReceipt {
	applyLedgerFlags(ctx, db, userID, &out)
	return out
}

func previewFromFiscalCache(ctx context.Context, db *sql.DB, userID string, params fns.QRParams) (fns.ParsedReceipt, bool, error) {
	raw, err := queries(db).GetReceiptFiscalPreview(ctx, sqlcdb.GetReceiptFiscalPreviewParams{
		UserID:               userID,
		FiscalDriveNumber:    params.FiscalDriveNumber,
		FiscalDocumentNumber: params.FiscalDocumentNumber,
		FiscalSign:           params.FiscalSign,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return fns.ParsedReceipt{}, false, nil
	}
	if err != nil {
		return fns.ParsedReceipt{}, false, err
	}
	out, err := parsedReceiptFromJSON(raw)
	if err != nil {
		return fns.ParsedReceipt{}, false, err
	}
	return out, true, nil
}

func saveFiscalPreview(ctx context.Context, db *sql.DB, userID string, params fns.QRParams, receipt fns.ParsedReceipt) error {
	if !isValidPreviewReceipt(receipt, params) {
		return nil
	}
	raw, err := json.Marshal(receipt)
	if err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	return queries(db).UpsertReceiptFiscalPreview(ctx, sqlcdb.UpsertReceiptFiscalPreviewParams{
		UserID:               userID,
		FiscalDriveNumber:    params.FiscalDriveNumber,
		FiscalDocumentNumber: params.FiscalDocumentNumber,
		FiscalSign:           params.FiscalSign,
		ResultJson:           string(raw),
		FetchedAt:            now,
	})
}

func isValidPreviewReceipt(receipt fns.ParsedReceipt, params fns.QRParams) bool {
	if receipt.TotalSum <= 0 && len(receipt.Items) == 0 {
		return false
	}
	if strings.TrimSpace(receipt.FiscalDriveNumber) == "" {
		return false
	}
	if receipt.FiscalDriveNumber != params.FiscalDriveNumber ||
		receipt.FiscalDocumentNumber != params.FiscalDocumentNumber ||
		receipt.FiscalSign != params.FiscalSign {
		return false
	}
	return true
}

func parsedReceiptFromJSON(raw string) (fns.ParsedReceipt, error) {
	var out fns.ParsedReceipt
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return fns.ParsedReceipt{}, err
	}
	return out, nil
}

func previewFromReceiptImport(ctx context.Context, db *sql.DB, userID string, params fns.QRParams) (fns.ParsedReceipt, bool, error) {
	q := queries(db)
	importID, err := q.GetReceiptImportByFiscalKey(ctx, sqlcdb.GetReceiptImportByFiscalKeyParams{
		UserID:               userID,
		FiscalDriveNumber:    params.FiscalDriveNumber,
		FiscalDocumentNumber: params.FiscalDocumentNumber,
		FiscalSign:           params.FiscalSign,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return fns.ParsedReceipt{}, false, nil
	}
	if err != nil {
		return fns.ParsedReceipt{}, false, err
	}
	row, err := q.GetReceiptImportByID(ctx, sqlcdb.GetReceiptImportByIDParams{ID: importID, UserID: userID})
	if err != nil {
		return fns.ParsedReceipt{}, false, err
	}
	payload, err := fns.PayloadFromJSON(row.PayloadJson)
	if err != nil {
		return fns.ParsedReceipt{}, false, err
	}
	out, err := fns.ParsedFromPayload(payload, row.FnsCheckUrl)
	if err != nil {
		return fns.ParsedReceipt{}, false, err
	}
	return out, true, nil
}

func previewFromScanJob(ctx context.Context, db *sql.DB, userID string, params fns.QRParams) (fns.ParsedReceipt, bool, error) {
	fn := params.FiscalDriveNumber
	doc := params.FiscalDocumentNumber
	fp := params.FiscalSign
	raw, err := queries(db).GetDoneScanJobResultByFiscalKey(ctx, sqlcdb.GetDoneScanJobResultByFiscalKeyParams{
		UserID:               userID,
		FiscalDriveNumber:    &fn,
		FiscalDocumentNumber: &doc,
		FiscalSign:           &fp,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return fns.ParsedReceipt{}, false, nil
	}
	if err != nil {
		return fns.ParsedReceipt{}, false, err
	}
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return fns.ParsedReceipt{}, false, nil
	}
	out, err := parsedReceiptFromJSON(*raw)
	if err != nil {
		return fns.ParsedReceipt{}, false, err
	}
	return out, true, nil
}
