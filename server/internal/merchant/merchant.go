package merchant

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
)

type Merchant struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Icon      string `json:"icon"`
	CreatedAt string `json:"created_at"`
}

var (
	ErrNotFound    = errors.New("merchant not found")
	ErrNameTaken   = errors.New("merchant name already exists")
	ErrInvalidName = errors.New("invalid merchant name")
)

func queries(db sqlcdb.DBTX) *sqlcdb.Queries {
	return sqlcdb.New(db)
}

func normalizeIcon(icon string) string {
	icon = strings.TrimSpace(icon)
	if icon == "" {
		return "default"
	}
	return icon
}

func List(ctx context.Context, db *sql.DB, userID string) ([]Merchant, error) {
	rows, err := queries(db).ListMerchantsByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]Merchant, 0, len(rows))
	for _, row := range rows {
		out = append(out, fromRow(row))
	}
	return out, nil
}

func Create(ctx context.Context, db *sql.DB, userID, name, icon string) (Merchant, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Merchant{}, ErrInvalidName
	}
	icon = normalizeIcon(icon)
	existing, err := queries(db).GetMerchantByName(ctx, sqlcdb.GetMerchantByNameParams{
		UserID: userID, Name: name,
	})
	if err == nil {
		return fromRow(existing), nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return Merchant{}, err
	}
	// SQLite lower()/NOCASE — ASCII; для кириллицы ищем через EqualFold.
	if m, ok := findByEqualFold(ctx, db, userID, name); ok {
		return m, nil
	}

	id := uuid.NewString()
	now := time.Now().UTC().Format(time.RFC3339)
	if err := queries(db).InsertMerchant(ctx, sqlcdb.InsertMerchantParams{
		ID: id, UserID: userID, Name: name, Icon: icon, CreatedAt: now,
	}); err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return Merchant{}, ErrNameTaken
		}
		return Merchant{}, err
	}
	return Get(ctx, db, userID, id)
}

func findByEqualFold(ctx context.Context, db *sql.DB, userID, name string) (Merchant, bool) {
	rows, err := queries(db).ListMerchantsByUser(ctx, userID)
	if err != nil {
		return Merchant{}, false
	}
	for _, row := range rows {
		if strings.EqualFold(row.Name, name) {
			return fromRow(row), true
		}
	}
	return Merchant{}, false
}

func Get(ctx context.Context, db *sql.DB, userID, id string) (Merchant, error) {
	row, err := queries(db).GetMerchantByID(ctx, sqlcdb.GetMerchantByIDParams{ID: id, UserID: userID})
	if errors.Is(err, sql.ErrNoRows) {
		return Merchant{}, ErrNotFound
	}
	if err != nil {
		return Merchant{}, err
	}
	return fromRow(row), nil
}

func Update(ctx context.Context, db *sql.DB, userID, id, name, icon string) (Merchant, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Merchant{}, ErrInvalidName
	}
	existing, err := Get(ctx, db, userID, id)
	if err != nil {
		return Merchant{}, err
	}
	icon = strings.TrimSpace(icon)
	if icon == "" {
		icon = existing.Icon
	}
	n, err := queries(db).UpdateMerchant(ctx, sqlcdb.UpdateMerchantParams{
		Name: name, Icon: icon, ID: id, UserID: userID,
	})
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return Merchant{}, ErrNameTaken
		}
		return Merchant{}, err
	}
	if n == 0 {
		return Merchant{}, ErrNotFound
	}
	return Get(ctx, db, userID, id)
}

func Delete(ctx context.Context, db *sql.DB, userID, id string) error {
	if _, err := Get(ctx, db, userID, id); err != nil {
		return err
	}
	n, err := queries(db).DeleteMerchant(ctx, sqlcdb.DeleteMerchantParams{ID: id, UserID: userID})
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// Resolve returns merchant id from id or name (get-or-create). Empty both → nil.
func Resolve(ctx context.Context, db *sql.DB, userID string, merchantID, merchantName *string) (*string, error) {
	if merchantID != nil && *merchantID != "" {
		if _, err := Get(ctx, db, userID, *merchantID); err != nil {
			return nil, err
		}
		return merchantID, nil
	}
	if merchantName != nil && strings.TrimSpace(*merchantName) != "" {
		m, err := Create(ctx, db, userID, *merchantName, "default")
		if err != nil && !errors.Is(err, ErrNameTaken) {
			return nil, err
		}
		if errors.Is(err, ErrNameTaken) {
			row, err := queries(db).GetMerchantByName(ctx, sqlcdb.GetMerchantByNameParams{
				UserID: userID, Name: strings.TrimSpace(*merchantName),
			})
			if err != nil {
				return nil, err
			}
			return &row.ID, nil
		}
		return &m.ID, nil
	}
	return nil, nil
}

func fromRow(row sqlcdb.Merchant) Merchant {
	return Merchant{ID: row.ID, Name: row.Name, Icon: row.Icon, CreatedAt: row.CreatedAt}
}
