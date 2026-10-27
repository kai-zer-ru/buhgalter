package debt

import (
	"context"
	"database/sql"
	"errors"

	"github.com/kai-zer-ru/buhgalter/internal/auth"
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
	Deletable       bool    `json:"deletable"`
}

type DebtorDetail struct {
	Debtor
	IOwe         int64             `json:"i_owe"`
	OwedToMe     int64             `json:"owed_to_me"`
	ByCurrency   []DebtorCurrency  `json:"by_currency"`
	Debts        []Debt            `json:"debts"`
	Transactions []DebtTransaction `json:"transactions"`
}

type DebtorCurrency struct {
	IOwe     int64  `json:"i_owe"`
	OwedToMe int64  `json:"owed_to_me"`
	Currency string `json:"currency"`
}

func GetDebtorDetail(ctx context.Context, db *sql.DB, userID, debtorID string) (DebtorDetail, error) {
	return GetDebtorDetailByScope(ctx, db, auth.Scope{UserID: userID}, debtorID)
}

func GetDebtorDetailByScope(ctx context.Context, db *sql.DB, sc auth.Scope, debtorID string) (DebtorDetail, error) {
	debtor, err := GetDebtorByScope(ctx, db, sc, debtorID)
	if err != nil {
		return DebtorDetail{}, err
	}
	tz, err := userTimezone(ctx, db, sc.UserID)
	if err != nil {
		return DebtorDetail{}, err
	}
	now := timeutil.NowUTC()

	rows, err := queries(db).ListDebtsByDebtor(ctx, sqlcdb.ListDebtsByDebtorParams{
		UserID: debtor.OwnerUserID, DebtorID: debtorID,
	})
	if err != nil {
		return DebtorDetail{}, err
	}

	debts := make([]Debt, 0, len(rows))
	type pair struct{ iOwe, owed int64 }
	by := map[string]pair{}
	for _, row := range rows {
		d, err := debtFromFields(debtRowFromDebtorList(row), tz, now)
		if err != nil {
			return DebtorDetail{}, err
		}
		debts = append(debts, d)
		if !d.IsSettled {
			cur := money.ProfileOrDefault(d.Currency)
			p := by[cur]
			switch d.Direction {
			case "borrowed":
				p.iOwe += d.Amount
			case "lent":
				p.owed += d.Amount
			}
			by[cur] = p
		}
	}
	profile, err := queries(db).GetUserFormatting(ctx, sc.UserID)
	if err != nil {
		return DebtorDetail{}, err
	}
	profileCur := money.ProfileOrDefault(profile.Currency)
	codes := make([]string, 0, len(by))
	for c := range by {
		codes = append(codes, c)
	}
	order := money.OrderPresentWithProfileFirst(profileCur, codes)
	byCurrency := make([]DebtorCurrency, 0, len(order))
	var iOwe, owedToMe int64
	for _, c := range order {
		p := by[c]
		byCurrency = append(byCurrency, DebtorCurrency{IOwe: p.iOwe, OwedToMe: p.owed, Currency: c})
		if c == profileCur {
			iOwe, owedToMe = p.iOwe, p.owed
		}
	}

	q := queries(db)
	txIDRows, err := q.ListTransactionIDsByDebtor(ctx, sqlcdb.ListTransactionIDsByDebtorParams{
		UserID: debtor.OwnerUserID, DebtorID: debtorID,
	})
	if err != nil {
		return DebtorDetail{}, err
	}

	txs := make([]DebtTransaction, 0, len(txIDRows))
	for _, txID := range txIDRows {
		tx, err := loadDebtTransaction(ctx, q, debtor.OwnerUserID, txID)
		if err != nil {
			continue
		}
		txs = append(txs, tx)
	}

	return DebtorDetail{
		Debtor:       debtor,
		IOwe:         iOwe,
		OwedToMe:     owedToMe,
		ByCurrency:   byCurrency,
		Debts:        debts,
		Transactions: txs,
	}, nil
}

func loadDebtTransaction(ctx context.Context, q *sqlcdb.Queries, userID, id string) (DebtTransaction, error) {
	row, err := q.GetTransactionByID(ctx, sqlcdb.GetTransactionByIDParams{ID: id, OwnerIds: []string{userID}})
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
	link, err := q.GetDebtLinkByTransactionID(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		t.Deletable = true
		return t, nil
	}
	if err != nil {
		return DebtTransaction{}, err
	}
	if link.Role != "open" {
		t.Deletable = true
		return t, nil
	}
	n, err := q.CountDebtLinksByDebt(ctx, link.DebtID)
	if err != nil {
		return DebtTransaction{}, err
	}
	t.Deletable = n <= 1
	return t, nil
}

func debtRowFromDebtorList(r sqlcdb.ListDebtsByDebtorRow) debtRowFields {
	return debtRowFields{
		id: r.ID, debtorID: r.DebtorID, debtorName: r.DebtorName, direction: r.Direction,
		amount: r.Amount, affectsBalance: r.AffectsBalance, debtDate: r.DebtDate, dueDate: r.DueDate,
		description: r.Description, transactionID: r.TransactionID, isSettled: r.IsSettled,
		settledAt: r.SettledAt, createdAt: r.CreatedAt,
		accountID: r.OpenAccountID, accountName: r.OpenAccountName,
		currency: r.Currency,
	}
}
