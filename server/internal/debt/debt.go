package debt

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kai-zer-ru/buhgalter/internal/accountbalance"
	"github.com/kai-zer-ru/buhgalter/internal/categoryseed"
	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
	"github.com/kai-zer-ru/buhgalter/internal/money"
	"github.com/kai-zer-ru/buhgalter/internal/timeutil"
)

type Debt struct {
	ID              string  `json:"id"`
	DebtorID        string  `json:"debtor_id"`
	DebtorName      string  `json:"debtor_name"`
	Direction       string  `json:"direction"`
	Amount          int64   `json:"amount"`
	AmountDisplay   string  `json:"amount_display"`
	AffectsBalance  bool    `json:"affects_balance"`
	DebtDate        string  `json:"debt_date"`
	DueDate         string  `json:"due_date"`
	Description     *string `json:"description"`
	TransactionID   *string `json:"transaction_id"`
	IsSettled       bool    `json:"is_settled"`
	SettledAt       *string `json:"settled_at"`
	IsOverdue       bool    `json:"is_overdue"`
	CreatedAt       string  `json:"created_at"`
	AccountID       *string `json:"account_id,omitempty"`
	AccountName     *string `json:"account_name,omitempty"`
}

type CreateInput struct {
	DebtorID        *string
	DebtorName      *string
	Direction       string
	Amount          int64
	DebtDate        time.Time
	DueDate         time.Time
	AffectsBalance  bool
	Description     *string
	AccountID       string
}

type SettleInput struct {
	Amount         int64 // 0 = погасить весь остаток
	SettledAt      time.Time
	AffectsBalance bool
	AccountID      string
}

var (
	ErrNotFound          = errors.New("debt not found")
	ErrInvalidDirection  = errors.New("invalid direction")
	ErrInvalidAmount     = errors.New("invalid amount")
	ErrInvalidDueDate    = errors.New("invalid due date")
	ErrInvalidDebtDate   = errors.New("invalid debt date")
	ErrAccountRequired   = errors.New("account required when affects balance")
	ErrInvalidAccount    = errors.New("invalid account")
	ErrAccountArchived   = errors.New("account is archived")
	ErrAlreadySettled    = errors.New("debt already settled")
	ErrInvalidSettleAmount   = errors.New("invalid settle amount")
	ErrCannotBorrowFromDebtor = errors.New("cannot borrow from debtor who owes you")
	ErrCannotLendToCreditor   = errors.New("cannot lend to creditor you owe")
	ErrPlannedNotAllowed      = errors.New("planned operation is not allowed for debt transactions")
)

func queries(db sqlcdb.DBTX) *sqlcdb.Queries {
	return sqlcdb.New(db)
}

func List(ctx context.Context, db *sql.DB, userID string, settledFilter string) ([]Debt, error) {
	tz, err := userTimezone(ctx, db, userID)
	if err != nil {
		return nil, err
	}
	now := timeutil.NowUTC()

	var rows interface{}
	switch settledFilter {
	case "true", "1":
		rows, err = queries(db).ListSettledDebtsByUser(ctx, userID)
	case "false", "0":
		rows, err = queries(db).ListActiveDebtsByUser(ctx, userID)
	default:
		rows, err = queries(db).ListAllDebtsByUser(ctx, userID)
	}
	if err != nil {
		return nil, err
	}

	return debtsFromRows(rows, tz, now)
}

func debtsFromRows(rows interface{}, tz string, now time.Time) ([]Debt, error) {
	switch list := rows.(type) {
	case []sqlcdb.ListActiveDebtsByUserRow:
		return mapDebtRows(list, tz, now, func(i int) debtRowFields { return debtRowFromActive(list[i]) })
	case []sqlcdb.ListSettledDebtsByUserRow:
		return mapDebtRows(list, tz, now, func(i int) debtRowFields { return debtRowFromSettled(list[i]) })
	case []sqlcdb.ListAllDebtsByUserRow:
		return mapDebtRows(list, tz, now, func(i int) debtRowFields { return debtRowFromAll(list[i]) })
	default:
		return nil, fmt.Errorf("unexpected debt row type %T", rows)
	}
}

type debtRowFields struct {
	id, debtorID, debtorName, direction, debtDate, dueDate, createdAt string
	amount                                                  int64
	affectsBalance, isSettled                               int64
	description, transactionID, settledAt                     *string
	accountID, accountName                                    *string
}

func mapDebtRows[T any](list []T, tz string, now time.Time, at func(int) debtRowFields) ([]Debt, error) {
	out := make([]Debt, 0, len(list))
	for i := range list {
		d, err := debtFromFields(at(i), tz, now)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, nil
}

func debtRowFromActive(r sqlcdb.ListActiveDebtsByUserRow) debtRowFields {
	return debtRowFields{
		id: r.ID, debtorID: r.DebtorID, debtorName: r.DebtorName, direction: r.Direction,
		amount: r.Amount, affectsBalance: r.AffectsBalance, debtDate: r.DebtDate, dueDate: r.DueDate,
		description: r.Description, transactionID: r.TransactionID, isSettled: r.IsSettled,
		settledAt: r.SettledAt, createdAt: r.CreatedAt,
		accountID: r.OpenAccountID, accountName: r.OpenAccountName,
	}
}

func debtRowFromSettled(r sqlcdb.ListSettledDebtsByUserRow) debtRowFields {
	return debtRowFields{
		id: r.ID, debtorID: r.DebtorID, debtorName: r.DebtorName, direction: r.Direction,
		amount: r.Amount, affectsBalance: r.AffectsBalance, debtDate: r.DebtDate, dueDate: r.DueDate,
		description: r.Description, transactionID: r.TransactionID, isSettled: r.IsSettled,
		settledAt: r.SettledAt, createdAt: r.CreatedAt,
		accountID: r.OpenAccountID, accountName: r.OpenAccountName,
	}
}

func debtRowFromAll(r sqlcdb.ListAllDebtsByUserRow) debtRowFields {
	return debtRowFields{
		id: r.ID, debtorID: r.DebtorID, debtorName: r.DebtorName, direction: r.Direction,
		amount: r.Amount, affectsBalance: r.AffectsBalance, debtDate: r.DebtDate, dueDate: r.DueDate,
		description: r.Description, transactionID: r.TransactionID, isSettled: r.IsSettled,
		settledAt: r.SettledAt, createdAt: r.CreatedAt,
		accountID: r.OpenAccountID, accountName: r.OpenAccountName,
	}
}

func debtFromGetRow(row sqlcdb.GetDebtByIDRow, tz string, now time.Time) (Debt, error) {
	return debtFromFields(debtRowFields{
		id: row.ID, debtorID: row.DebtorID, debtorName: row.DebtorName, direction: row.Direction,
		amount: row.Amount, affectsBalance: row.AffectsBalance, debtDate: row.DebtDate, dueDate: row.DueDate,
		description: row.Description, transactionID: row.TransactionID, isSettled: row.IsSettled,
		settledAt: row.SettledAt, createdAt: row.CreatedAt,
		accountID: row.OpenAccountID, accountName: row.OpenAccountName,
	}, tz, now)
}

func debtFromFields(f debtRowFields, tz string, now time.Time) (Debt, error) {
	overdue := false
	if f.isSettled == 0 {
		due, err := timeutil.ParseUTC(f.dueDate)
		if err != nil {
			return Debt{}, err
		}
		overdue, err = timeutil.IsOverdueInTZ(due, now, tz)
		if err != nil {
			return Debt{}, err
		}
	}
	return Debt{
		ID:             f.id,
		DebtorID:       f.debtorID,
		DebtorName:     f.debtorName,
		Direction:      f.direction,
		Amount:         f.amount,
		AmountDisplay:  money.FormatRubles(f.amount),
		AffectsBalance: f.affectsBalance == 1,
		DebtDate:       f.debtDate,
		DueDate:        f.dueDate,
		Description:    f.description,
		TransactionID:  f.transactionID,
		IsSettled:      f.isSettled == 1,
		SettledAt:      f.settledAt,
		IsOverdue:      overdue,
		CreatedAt:      f.createdAt,
		AccountID:      f.accountID,
		AccountName:    f.accountName,
	}, nil
}

func GetByID(ctx context.Context, db *sql.DB, userID, id string) (Debt, error) {
	row, err := queries(db).GetDebtByID(ctx, sqlcdb.GetDebtByIDParams{ID: id, UserID: userID})
	if errors.Is(err, sql.ErrNoRows) {
		return Debt{}, ErrNotFound
	}
	if err != nil {
		return Debt{}, err
	}
	tz, err := userTimezone(ctx, db, userID)
	if err != nil {
		return Debt{}, err
	}
	return debtFromGetRow(row, tz, timeutil.NowUTC())
}

func Create(ctx context.Context, db *sql.DB, userID string, in CreateInput) (Debt, error) {
	if in.Direction != "lent" && in.Direction != "borrowed" {
		return Debt{}, ErrInvalidDirection
	}
	if in.Amount <= 0 {
		return Debt{}, ErrInvalidAmount
	}
	debtorID, err := resolveDebtor(ctx, db, userID, in.DebtorID, in.DebtorName)
	if err != nil {
		return Debt{}, err
	}
	if err := validateDebtDirection(ctx, db, userID, debtorID, in.Direction); err != nil {
		return Debt{}, err
	}
	if in.AffectsBalance {
		if in.AccountID == "" {
			return Debt{}, ErrAccountRequired
		}
		if err := validateActiveAccount(ctx, db, userID, in.AccountID); err != nil {
			return Debt{}, err
		}
	}

	id := uuid.NewString()
	now := time.Now().UTC().Format(time.RFC3339)
	debtDate := timeutil.FormatUTC(in.DebtDate)
	dueDate := timeutil.FormatUTC(in.DueDate)
	affects := int64(0)
	if in.AffectsBalance {
		affects = 1
	}

	dbTx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return Debt{}, err
	}
	defer func() { _ = dbTx.Rollback() }()

	q := queries(dbTx)
	var txID *string
	if in.AffectsBalance {
		created, err := insertDebtTransaction(ctx, dbTx, userID, in, debtorID)
		if err != nil {
			return Debt{}, err
		}
		txID = &created
	}

	if err := q.InsertDebt(ctx, sqlcdb.InsertDebtParams{
		ID: id, UserID: userID, DebtorID: debtorID, Direction: in.Direction,
		Amount: in.Amount, AffectsBalance: affects, DebtDate: debtDate, DueDate: dueDate,
		Description: in.Description, TransactionID: txID,
		IsSettled: 0, SettledAt: nil, CreatedAt: now,
	}); err != nil {
		return Debt{}, err
	}
	if txID != nil {
		if err := q.InsertDebtTransactionLink(ctx, sqlcdb.InsertDebtTransactionLinkParams{
			DebtID: id, TransactionID: *txID, Role: "open",
		}); err != nil {
			return Debt{}, err
		}
	}
	if err := dbTx.Commit(); err != nil {
		return Debt{}, err
	}
	if in.AffectsBalance {
		if err := accountbalance.Refresh(ctx, db, userID, in.AccountID); err != nil {
			return Debt{}, err
		}
	}
	return GetByID(ctx, db, userID, id)
}

func Settle(ctx context.Context, db *sql.DB, userID, id string, in SettleInput) (Debt, error) {
	existing, err := GetByID(ctx, db, userID, id)
	if err != nil {
		return Debt{}, err
	}
	if existing.IsSettled {
		return Debt{}, ErrAlreadySettled
	}

	settleAmount := in.Amount
	if settleAmount == 0 {
		settleAmount = existing.Amount
	}
	if settleAmount <= 0 || settleAmount > existing.Amount {
		return Debt{}, ErrInvalidSettleAmount
	}

	if in.AffectsBalance {
		if in.AccountID == "" {
			return Debt{}, ErrAccountRequired
		}
		if err := validateActiveAccount(ctx, db, userID, in.AccountID); err != nil {
			return Debt{}, err
		}
	}

	dbTx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return Debt{}, err
	}
	defer func() { _ = dbTx.Rollback() }()

	q := queries(dbTx)
	if in.AffectsBalance {
		settleTxID, err := insertSettleTransaction(ctx, dbTx, userID, existing, in, settleAmount)
		if err != nil {
			return Debt{}, err
		}
		if err := q.InsertDebtTransactionLink(ctx, sqlcdb.InsertDebtTransactionLinkParams{
			DebtID: id, TransactionID: settleTxID, Role: "settle",
		}); err != nil {
			return Debt{}, err
		}
	}

	remaining := existing.Amount - settleAmount
	if remaining == 0 {
		settledAt := timeutil.FormatUTC(in.SettledAt)
		n, err := q.SettleDebt(ctx, sqlcdb.SettleDebtParams{
			SettledAt: &settledAt, ID: id, UserID: userID,
		})
		if err != nil {
			return Debt{}, err
		}
		if n == 0 {
			return Debt{}, ErrNotFound
		}
	} else {
		n, err := q.ReduceDebtAmount(ctx, sqlcdb.ReduceDebtAmountParams{
			Amount: remaining, ID: id, UserID: userID,
		})
		if err != nil {
			return Debt{}, err
		}
		if n == 0 {
			return Debt{}, ErrNotFound
		}
	}

	if err := dbTx.Commit(); err != nil {
		return Debt{}, err
	}
	if in.AffectsBalance {
		if err := accountbalance.Refresh(ctx, db, userID, in.AccountID); err != nil {
			return Debt{}, err
		}
	}
	return GetByID(ctx, db, userID, id)
}

func Delete(ctx context.Context, db *sql.DB, userID, id string) error {
	if _, err := GetByID(ctx, db, userID, id); err != nil {
		return err
	}

	dbTx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = dbTx.Rollback() }()

	q := queries(dbTx)

	txIDs, err := q.ListTransactionIDsByDebt(ctx, id)
	if err != nil {
		return err
	}
	if err := q.ClearDebtTransactionLinkByDebtID(ctx, sqlcdb.ClearDebtTransactionLinkByDebtIDParams{
		ID: id, UserID: userID,
	}); err != nil {
		return err
	}
	n, err := q.DeleteDebt(ctx, sqlcdb.DeleteDebtParams{ID: id, UserID: userID})
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	for _, txID := range txIDs {
		if _, err := q.DeleteTransaction(ctx, sqlcdb.DeleteTransactionParams{
			ID: txID, UserID: userID,
		}); err != nil {
			return err
		}
	}
	if err := dbTx.Commit(); err != nil {
		return err
	}
	_ = accountbalance.Refresh(ctx, db, userID)
	return nil
}

func insertDebtTransaction(ctx context.Context, db sqlcdb.DBTX, userID string, in CreateInput, debtorID string) (string, error) {
	txType := "expense"
	if in.Direction == "borrowed" {
		txType = "income"
	}
	catID, err := categoryseed.DebtCategoryID(ctx, db, userID, txType)
	if err != nil {
		return "", err
	}
	q := queries(db)
	kind, err := resolveTransactionKind(ctx, db, userID, in.DebtDate)
	if err != nil {
		return "", err
	}
	debtor, err := q.GetDebtorByID(ctx, sqlcdb.GetDebtorByIDParams{ID: debtorID, UserID: userID})
	if err != nil {
		return "", err
	}
	desc := debtTxDescription(in.Description, debtor.Name, in.Direction, false, false)

	id := uuid.NewString()
	now := time.Now().UTC().Format(time.RFC3339)
	txDate := timeutil.FormatUTC(in.DebtDate)
	if err := q.InsertTransaction(ctx, sqlcdb.InsertTransactionParams{
		ID: id, UserID: userID, AccountID: in.AccountID,
		Type: txType, Kind: kind, Amount: in.Amount, Description: &desc,
		CategoryID: &catID, SubcategoryID: nil,
		TransferGroupID: nil, TransferAccountID: nil,
		TransactionDate: txDate, AffectsBalance: 1, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		return "", err
	}
	return id, nil
}

func insertSettleTransaction(ctx context.Context, db sqlcdb.DBTX, userID string, d Debt, in SettleInput, amount int64) (string, error) {
	txType := "income"
	if d.Direction == "borrowed" {
		txType = "expense"
	}
	catID, err := categoryseed.DebtCategoryID(ctx, db, userID, txType)
	if err != nil {
		return "", err
	}
	q := queries(db)
	kind, err := resolveTransactionKind(ctx, db, userID, in.SettledAt)
	if err != nil {
		return "", err
	}
	partial := amount < d.Amount
	desc := debtTxDescription(d.Description, d.DebtorName, d.Direction, true, partial)

	id := uuid.NewString()
	now := time.Now().UTC().Format(time.RFC3339)
	txDate := timeutil.FormatUTC(in.SettledAt)
	if err := q.InsertTransaction(ctx, sqlcdb.InsertTransactionParams{
		ID: id, UserID: userID, AccountID: in.AccountID,
		Type: txType, Kind: kind, Amount: amount, Description: &desc,
		CategoryID: &catID, SubcategoryID: nil,
		TransferGroupID: nil, TransferAccountID: nil,
		TransactionDate: txDate, AffectsBalance: 1, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		return "", err
	}
	return id, nil
}

func debtTxDescription(userDesc *string, debtorName, direction string, settle, partial bool) string {
	var action string
	switch {
	case direction == "lent" && !settle:
		action = "Дал в долг"
	case direction == "borrowed" && !settle:
		action = "Взял в долг"
	case direction == "lent" && settle && partial:
		action = "Частичный возврат долга"
	case direction == "borrowed" && settle && partial:
		action = "Частичное погашение долга"
	case direction == "lent" && settle:
		action = "Возврат долга"
	case direction == "borrowed" && settle:
		action = "Погашение долга"
	}
	base := fmt.Sprintf("%s: %s", action, debtorName)
	if userDesc != nil && strings.TrimSpace(*userDesc) != "" {
		return base + " — " + strings.TrimSpace(*userDesc)
	}
	return base
}

func validateDebtDirection(ctx context.Context, db sqlcdb.DBTX, userID, debtorID, direction string) error {
	var opposite string
	switch direction {
	case "lent":
		opposite = "borrowed"
	case "borrowed":
		opposite = "lent"
	default:
		return ErrInvalidDirection
	}
	count, err := queries(db).CountActiveDebtsByDebtorAndDirection(ctx, sqlcdb.CountActiveDebtsByDebtorAndDirectionParams{
		DebtorID:  debtorID,
		UserID:    userID,
		Direction: opposite,
	})
	if err != nil {
		return err
	}
	if count == 0 {
		return nil
	}
	if direction == "borrowed" {
		return ErrCannotBorrowFromDebtor
	}
	return ErrCannotLendToCreditor
}

func validateActiveAccount(ctx context.Context, db sqlcdb.DBTX, userID, accountID string) error {
	row, err := queries(db).GetAccountByID(ctx, sqlcdb.GetAccountByIDParams{ID: accountID, UserID: userID})
	if errors.Is(err, sql.ErrNoRows) {
		return ErrInvalidAccount
	}
	if err != nil {
		return err
	}
	if row.Status != "active" {
		return ErrAccountArchived
	}
	return nil
}

func userTimezone(ctx context.Context, db *sql.DB, userID string) (string, error) {
	tz, err := queries(db).GetUserTimezone(ctx, userID)
	if err != nil {
		return "", err
	}
	if tz == "" {
		return "Europe/Moscow", nil
	}
	return tz, nil
}

func userTimezoneDBTX(ctx context.Context, db sqlcdb.DBTX, userID string) (string, error) {
	tz, err := sqlcdb.New(db).GetUserTimezone(ctx, userID)
	if err != nil {
		return "", err
	}
	if tz == "" {
		return "Europe/Moscow", nil
	}
	return tz, nil
}

func resolveTransactionKind(ctx context.Context, db sqlcdb.DBTX, userID string, txDate time.Time) (string, error) {
	tz, err := userTimezoneDBTX(ctx, db, userID)
	if err != nil {
		return "", err
	}
	future, err := timeutil.IsFutureInTZ(txDate, timeutil.NowUTC(), tz)
	if err != nil {
		return "", err
	}
	if future {
		return "", ErrPlannedNotAllowed
	}
	return "manual", nil
}
