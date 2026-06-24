package debt

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
)

type Debtor struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
}

var (
	ErrDebtorNotFound      = errors.New("debtor not found")
	ErrDebtorNameTaken     = errors.New("debtor name already exists")
	ErrDebtorHasActiveDebt = errors.New("debtor has active debts")
	ErrInvalidDebtorName   = errors.New("invalid debtor name")
)

func ListDebtors(ctx context.Context, db *sql.DB, userID string) ([]Debtor, error) {
	rows, err := queries(db).ListDebtorsByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]Debtor, 0, len(rows))
	for _, row := range rows {
		out = append(out, debtorFromRow(row))
	}
	return out, nil
}

func CreateDebtor(ctx context.Context, db *sql.DB, userID, name string) (Debtor, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Debtor{}, ErrInvalidDebtorName
	}
	existing, err := queries(db).GetDebtorByName(ctx, sqlcdb.GetDebtorByNameParams{
		UserID: userID, Name: name,
	})
	if err == nil {
		return debtorFromRow(existing), nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return Debtor{}, err
	}

	id := uuid.NewString()
	now := time.Now().UTC().Format(time.RFC3339)
	if err := queries(db).InsertDebtor(ctx, sqlcdb.InsertDebtorParams{
		ID: id, UserID: userID, Name: name, CreatedAt: now,
	}); err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return Debtor{}, ErrDebtorNameTaken
		}
		return Debtor{}, err
	}
	return GetDebtor(ctx, db, userID, id)
}

func GetDebtor(ctx context.Context, db *sql.DB, userID, id string) (Debtor, error) {
	row, err := queries(db).GetDebtorByID(ctx, sqlcdb.GetDebtorByIDParams{ID: id, UserID: userID})
	if errors.Is(err, sql.ErrNoRows) {
		return Debtor{}, ErrDebtorNotFound
	}
	if err != nil {
		return Debtor{}, err
	}
	return debtorFromRow(row), nil
}

func UpdateDebtor(ctx context.Context, db *sql.DB, userID, id, name string) (Debtor, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Debtor{}, ErrInvalidDebtorName
	}
	if _, err := GetDebtor(ctx, db, userID, id); err != nil {
		return Debtor{}, err
	}
	n, err := queries(db).UpdateDebtorName(ctx, sqlcdb.UpdateDebtorNameParams{
		Name: name, ID: id, UserID: userID,
	})
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return Debtor{}, ErrDebtorNameTaken
		}
		return Debtor{}, err
	}
	if n == 0 {
		return Debtor{}, ErrDebtorNotFound
	}
	return GetDebtor(ctx, db, userID, id)
}

func DeleteDebtor(ctx context.Context, db *sql.DB, userID, id string) error {
	if _, err := GetDebtor(ctx, db, userID, id); err != nil {
		return err
	}
	count, err := queries(db).CountActiveDebtsByDebtor(ctx, sqlcdb.CountActiveDebtsByDebtorParams{
		DebtorID: id, UserID: userID,
	})
	if err != nil {
		return err
	}
	if count > 0 {
		return ErrDebtorHasActiveDebt
	}
	n, err := queries(db).DeleteDebtor(ctx, sqlcdb.DeleteDebtorParams{ID: id, UserID: userID})
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrDebtorNotFound
	}
	return nil
}

func resolveDebtor(ctx context.Context, db *sql.DB, userID string, debtorID, debtorName *string) (string, error) {
	if debtorID != nil && *debtorID != "" {
		if _, err := GetDebtor(ctx, db, userID, *debtorID); err != nil {
			return "", err
		}
		return *debtorID, nil
	}
	if debtorName != nil && strings.TrimSpace(*debtorName) != "" {
		d, err := CreateDebtor(ctx, db, userID, *debtorName)
		if err != nil && !errors.Is(err, ErrDebtorNameTaken) {
			return "", err
		}
		if errors.Is(err, ErrDebtorNameTaken) {
			row, err := queries(db).GetDebtorByName(ctx, sqlcdb.GetDebtorByNameParams{
				UserID: userID, Name: strings.TrimSpace(*debtorName),
			})
			if err != nil {
				return "", err
			}
			return row.ID, nil
		}
		return d.ID, nil
	}
	return "", ErrInvalidDebtorName
}

func debtorFromRow(row sqlcdb.Debtor) Debtor {
	return Debtor{ID: row.ID, Name: row.Name, CreatedAt: row.CreatedAt}
}
