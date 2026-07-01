package bank

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"fmt"

	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
)

//go:embed data/banks_ru.json
var banksJSON []byte

type bankRecord struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	BIC       string `json:"bic"`
	IconPath  string `json:"icon_path"`
	SortOrder int    `json:"sort_order"`
}

// SeedIfEmpty loads banks from embedded JSON (upserts — adds new and updates existing).
func SeedIfEmpty(ctx context.Context, db *sql.DB) error {
	var banks []bankRecord
	if err := json.Unmarshal(banksJSON, &banks); err != nil {
		return fmt.Errorf("parse banks json: %w", err)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	q := sqlcdb.New(tx)
	for _, b := range banks {
		var bic *string
		if b.BIC != "" {
			bic = &b.BIC
		}
		if err := q.UpsertBank(ctx, sqlcdb.UpsertBankParams{
			ID:        b.ID,
			Name:      b.Name,
			Bic:       bic,
			IconPath:  b.IconPath,
			SortOrder: int64(b.SortOrder),
		}); err != nil {
			return fmt.Errorf("upsert bank %s: %w", b.ID, err)
		}
	}
	return tx.Commit()
}

type Bank struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	BIC       *string `json:"bic"`
	IconPath  string  `json:"icon_path"`
	SortOrder int     `json:"sort_order"`
}

func ListAll(ctx context.Context, db *sql.DB) ([]Bank, error) {
	rows, err := sqlcdb.New(db).ListBanks(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Bank, 0, len(rows))
	for _, b := range rows {
		out = append(out, Bank{
			ID:        b.ID,
			Name:      b.Name,
			BIC:       b.Bic,
			IconPath:  b.IconPath,
			SortOrder: int(b.SortOrder),
		})
	}
	return out, nil
}

func Exists(ctx context.Context, db *sql.DB, id string) (bool, error) {
	return sqlcdb.New(db).BankExists(ctx, id)
}
