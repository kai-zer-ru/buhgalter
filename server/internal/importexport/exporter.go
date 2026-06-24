package importexport

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/csv"
	"fmt"
	"strings"
	"time"

	"github.com/kai-zer-ru/buhgalter/internal/transaction"
)

// ExportCSV returns Cubux-compatible CSV bytes for the given filters.
func ExportCSV(ctx context.Context, db *sql.DB, userID, displayName string, f ExportFilters) ([]byte, string, error) {
	from := normalizeExportDate(f.From, false)
	to := normalizeExportDate(f.To, true)

	var all []transaction.Transaction
	page := 1
	for {
		res, err := transaction.List(ctx, db, userID, transaction.ListFilters{
			AccountID: f.AccountID,
			CategoryID: f.CategoryID,
			From:       from,
			To:         to,
			Sort:       "date_asc",
			Page:       page,
			Limit:      200,
		})
		if err != nil {
			return nil, "", err
		}
		all = append(all, res.Data...)
		if int64(page*200) >= res.Meta.Total {
			break
		}
		page++
	}

	var buf bytes.Buffer
	buf.Write([]byte{0xEF, 0xBB, 0xBF}) // UTF-8 BOM
	w := csv.NewWriter(&buf)
	_ = w.Write(CubuxHeaders)

	seenTransfers := make(map[string]bool)
	for _, tx := range all {
		if tx.Type == "transfer" {
			if tx.TransferGroupID == nil || !tx.TransferIsOut {
				continue
			}
			if seenTransfers[*tx.TransferGroupID] {
				continue
			}
			seenTransfers[*tx.TransferGroupID] = true
		}
		line := txToCubuxLine(tx, displayName)
		if line == nil {
			continue
		}
		if err := w.Write(line); err != nil {
			return nil, "", err
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, "", err
	}

	filename := fmt.Sprintf("buhgalter_export_%s.csv", time.Now().Format("2006"))
	return buf.Bytes(), filename, nil
}

func txToCubuxLine(tx transaction.Transaction, displayName string) []string {
	date := formatExportDate(tx.TransactionDate)
	desc := ""
	if tx.Description != nil {
		desc = *tx.Description
	}
	cat := ""
	if tx.CategoryName != nil {
		cat = *tx.CategoryName
	}
	sub := ""
	if tx.SubcategoryName != nil {
		sub = *tx.SubcategoryName
	}

	switch tx.Type {
	case "expense":
		return []string{
			"Расходы", date, FormatCubuxAmount(tx.Amount), "RUB", tx.AccountName,
			"", "", "", cat, sub, desc, "", displayName,
		}
	case "income":
		return []string{
			"Доходы", date, "", "", "",
			FormatCubuxAmount(tx.Amount), "RUB", tx.AccountName, cat, sub, desc, "", displayName,
		}
	case "transfer":
		toAcct := ""
		if tx.TransferAccountName != "" {
			toAcct = tx.TransferAccountName
		}
		return []string{
			"Перевод", date, FormatCubuxAmount(tx.Amount), "RUB", tx.AccountName,
			FormatCubuxAmount(tx.Amount), "RUB", toAcct, "Перевод", "", desc, "", displayName,
		}
	default:
		return nil
	}
}

func formatExportDate(txDate string) string {
	txDate = strings.TrimSpace(txDate)
	if len(txDate) >= 10 {
		parts := strings.Split(txDate[:10], "-")
		if len(parts) == 3 {
			return parts[2] + "." + parts[1] + "." + parts[0]
		}
	}
	return txDate
}

func normalizeExportDate(s string, endOfDay bool) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if len(s) == 10 && strings.Count(s, "-") == 2 {
		if endOfDay {
			return s + " 23:59:59"
		}
		return s + " 00:00:00"
	}
	return s
}
