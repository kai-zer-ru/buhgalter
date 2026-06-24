package debt

import (
	"context"
	"database/sql"
	"errors"

	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
	"github.com/kai-zer-ru/buhgalter/internal/money"
	"github.com/kai-zer-ru/buhgalter/internal/timeutil"
)

type DebtTransaction struct {
	ID              string  `json:"id"`
	AccountID       string  `json:"account_id"`
	AccountName     string  `json:"account_name,omitempty"`
	Type            string  `json:"type"`
	Kind            string  `json:"kind"`
	Amount          int64   `json:"amount"`
	AmountDisplay   string  `json:"amount_display"`
	Description     *string `json:"description"`
	CategoryName    *string `json:"category_name,omitempty"`
	TransactionDate string  `json:"transaction_date"`
}

type DebtorDetail struct {
	Debtor
	IOwe         int64             `json:"i_owe"`
	OwedToMe     int64             `json:"owed_to_me"`
	Debts        []Debt            `json:"debts"`
	Transactions []DebtTransaction `json:"transactions"`
}

func GetDebtorDetail(ctx context.Context, db *sql.DB, userID, debtorID string) (DebtorDetail, error) {
	debtor, err := GetDebtor(ctx, db, userID, debtorID)
	if err != nil {
		return DebtorDetail{}, err
	}
	tz, err := userTimezone(ctx, db, userID)
	if err != nil {
		return DebtorDetail{}, err
	}
	now := timeutil.NowUTC()

	rows, err := queries(db).ListDebtsByDebtor(ctx, sqlcdb.ListDebtsByDebtorParams{
		UserID: userID, DebtorID: debtorID,
	})
	if err != nil {
		return DebtorDetail{}, err
	}

	debts := make([]Debt, 0, len(rows))
	var iOwe, owedToMe int64
	for _, row := range rows {
		d, err := debtFromFields(debtRowFromDebtorList(row), tz, now)
		if err != nil {
			return DebtorDetail{}, err
		}
		debts = append(debts, d)
		if !d.IsSettled {
			switch d.Direction {
			case "borrowed":
				iOwe += d.Amount
			case "lent":
				owedToMe += d.Amount
			}
		}
	}

	q := queries(db)
	txIDRows, err := q.ListTransactionIDsByDebtor(ctx, sqlcdb.ListTransactionIDsByDebtorParams{
		UserID: userID, DebtorID: debtorID,
	})
	if err != nil {
		return DebtorDetail{}, err
	}

	txs := make([]DebtTransaction, 0, len(txIDRows))
	for _, txID := range txIDRows {
		tx, err := loadDebtTransaction(ctx, q, userID, txID)
		if err != nil {
			continue
		}
		txs = append(txs, tx)
	}

	return DebtorDetail{
		Debtor:       debtor,
		IOwe:         iOwe,
		OwedToMe:     owedToMe,
		Debts:        debts,
		Transactions: txs,
	}, nil
}

func loadDebtTransaction(ctx context.Context, q *sqlcdb.Queries, userID, id string) (DebtTransaction, error) {
	row, err := q.GetTransactionByID(ctx, sqlcdb.GetTransactionByIDParams{ID: id, UserID: userID})
	if errors.Is(err, sql.ErrNoRows) {
		return DebtTransaction{}, err
	}
	if err != nil {
		return DebtTransaction{}, err
	}
	t := DebtTransaction{
		ID:              row.ID,
		AccountID:       row.AccountID,
		Type:            row.Type,
		Kind:            row.Kind,
		Amount:          row.Amount,
		AmountDisplay:   money.FormatRubles(row.Amount),
		Description:     row.Description,
		TransactionDate: row.TransactionDate,
	}
	if row.AccountName != nil {
		t.AccountName = *row.AccountName
	}
	if row.CategoryName != nil {
		t.CategoryName = row.CategoryName
	}
	return t, nil
}

func debtRowFromDebtorList(r sqlcdb.ListDebtsByDebtorRow) debtRowFields {
	return debtRowFields{
		id: r.ID, debtorID: r.DebtorID, debtorName: r.DebtorName, direction: r.Direction,
		amount: r.Amount, affectsBalance: r.AffectsBalance, debtDate: r.DebtDate, dueDate: r.DueDate,
		description: r.Description, transactionID: r.TransactionID, isSettled: r.IsSettled,
		settledAt: r.SettledAt, createdAt: r.CreatedAt,
		accountID: r.OpenAccountID, accountName: r.OpenAccountName,
	}
}
