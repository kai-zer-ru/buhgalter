package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
)

var ErrNotFound = errors.New("store not found")

type Store struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at,omitempty"`
}

func queries(db sqlcdb.DBTX) *sqlcdb.Queries {
	return sqlcdb.New(db)
}

func normalizeName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func List(ctx context.Context, db *sql.DB, userID string) ([]Store, error) {
	if err := SyncFromMerchants(ctx, db, userID); err != nil {
		return nil, err
	}
	rows, err := queries(db).ListStoresByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]Store, 0, len(rows))
	for _, row := range rows {
		out = append(out, Store{ID: row.ID, Name: row.Name, CreatedAt: row.CreatedAt})
	}
	return out, nil
}

func GetOrCreate(ctx context.Context, db sqlcdb.DBTX, userID, name string) (*string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, nil
	}
	norm := normalizeName(name)
	existing, err := queries(db).GetStoreByNormalizedName(ctx, sqlcdb.GetStoreByNormalizedNameParams{
		UserID:         userID,
		NameNormalized: norm,
	})
	if err == nil {
		id := existing.ID
		return &id, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	id := uuid.NewString()
	now := time.Now().UTC().Format(time.RFC3339)
	if err := queries(db).InsertStore(ctx, sqlcdb.InsertStoreParams{
		ID:             id,
		UserID:         userID,
		Name:           name,
		NameNormalized: norm,
		CreatedAt:      now,
	}); err != nil {
		return nil, err
	}
	return &id, nil
}

func ValidateForUser(ctx context.Context, db *sql.DB, userID, storeID string) error {
	if storeID == "" {
		return nil
	}
	_, err := queries(db).GetStoreByID(ctx, sqlcdb.GetStoreByIDParams{ID: storeID, UserID: userID})
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	return err
}
