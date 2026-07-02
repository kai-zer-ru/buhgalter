package account

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kai-zer-ru/buhgalter/internal/accountbalance"
	"github.com/kai-zer-ru/buhgalter/internal/bank"
	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
	"github.com/kai-zer-ru/buhgalter/internal/money"
)

type Account struct {
	ID                 string  `json:"id"`
	Name               string  `json:"name"`
	Type               string  `json:"type"`
	BankID             *string `json:"bank_id"`
	BankName           *string `json:"bank_name,omitempty"`
	BankIcon           *string `json:"bank_icon,omitempty"`
	InitialBalance     int64   `json:"initial_balance"`
	Balance            int64   `json:"balance"`
	BalanceDisplay     string  `json:"balance_display"`
	CreditLimit        *int64  `json:"credit_limit,omitempty"`
	CreditLimitDisplay *string `json:"credit_limit_display,omitempty"`
	PaymentAccountID   *string `json:"payment_account_id,omitempty"`
	Status             string  `json:"status"`
	IsPrimary          bool    `json:"is_primary"`
	CreatedAt          string  `json:"created_at"`
	UpdatedAt          string  `json:"updated_at"`
}

func queries(db *sql.DB) *sqlcdb.Queries {
	return sqlcdb.New(db)
}

func creditLimitDisplay(limit *int64) *string {
	if limit == nil {
		return nil
	}
	s := money.FormatRubles(*limit)
	return &s
}

func accountFromRow(
	id, name, accType, status, createdAt, updatedAt string,
	bankID *string,
	initialBalance, currentBalance int64,
	creditLimit *int64,
	paymentAccountID *string,
	isPrimary int64,
	bankName, bankIcon *string,
) Account {
	a := Account{
		ID:               id,
		Name:             name,
		Type:             accType,
		BankID:           bankID,
		BankName:         bankName,
		BankIcon:         bankIcon,
		InitialBalance:   initialBalance,
		Status:           status,
		IsPrimary:        isPrimary != 0,
		CreatedAt:        createdAt,
		UpdatedAt:        updatedAt,
		CreditLimit:      creditLimit,
		PaymentAccountID: paymentAccountID,
	}
	a.Balance = currentBalance
	a.BalanceDisplay = money.FormatRubles(currentBalance)
	a.CreditLimitDisplay = creditLimitDisplay(creditLimit)
	return a
}

func accountFromGetRow(row sqlcdb.GetAccountByIDRow) Account {
	return accountFromRow(
		row.ID, row.Name, row.Type, row.Status, row.CreatedAt, row.UpdatedAt,
		row.BankID, row.InitialBalance, row.CurrentBalance,
		row.CreditLimit, row.PaymentAccountID, row.IsPrimary, row.BankName, row.BankIcon,
	)
}

func accountFromListActive(row sqlcdb.ListAccountsByUserActiveRow) Account {
	return accountFromRow(
		row.ID, row.Name, row.Type, row.Status, row.CreatedAt, row.UpdatedAt,
		row.BankID, row.InitialBalance, row.CurrentBalance,
		row.CreditLimit, row.PaymentAccountID, row.IsPrimary, row.BankName, row.BankIcon,
	)
}

func accountFromListStatus(row sqlcdb.ListAccountsByUserAndStatusRow) Account {
	return accountFromRow(
		row.ID, row.Name, row.Type, row.Status, row.CreatedAt, row.UpdatedAt,
		row.BankID, row.InitialBalance, row.CurrentBalance,
		row.CreditLimit, row.PaymentAccountID, row.IsPrimary, row.BankName, row.BankIcon,
	)
}

func ListByUser(ctx context.Context, db *sql.DB, userID, status string) ([]Account, error) {
	q := queries(db)
	if status != "" {
		rows, err := q.ListAccountsByUserAndStatus(ctx, sqlcdb.ListAccountsByUserAndStatusParams{
			UserID: userID,
			Status: status,
		})
		if err != nil {
			return nil, err
		}
		out := make([]Account, 0, len(rows))
		for _, row := range rows {
			out = append(out, accountFromListStatus(row))
		}
		return out, nil
	}

	rows, err := q.ListAccountsByUserActive(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]Account, 0, len(rows))
	for _, row := range rows {
		out = append(out, accountFromListActive(row))
	}
	return out, nil
}

func GetByID(ctx context.Context, db *sql.DB, userID, id string) (Account, error) {
	row, err := queries(db).GetAccountByID(ctx, sqlcdb.GetAccountByIDParams{ID: id, UserID: userID})
	if errors.Is(err, sql.ErrNoRows) {
		return Account{}, ErrNotFound
	}
	if err != nil {
		return Account{}, err
	}
	return accountFromGetRow(row), nil
}

var ErrNotFound = errors.New("account not found")

type CreateInput struct {
	Name             string
	Type             string
	BankID           *string
	InitialBalance   int64
	CreditLimit      *int64
	PaymentAccountID *string
}

func Create(ctx context.Context, db *sql.DB, userID string, in CreateInput) (Account, error) {
	if err := validateNameUnique(ctx, db, userID, in.Name, ""); err != nil {
		return Account{}, err
	}
	creditLimit, paymentAccountID, err := validateAccountFields(ctx, db, userID, "", in.Type, in.BankID, in.CreditLimit, in.PaymentAccountID)
	if err != nil {
		return Account{}, err
	}

	id := uuid.NewString()
	now := time.Now().UTC().Format(time.RFC3339)
	q := queries(db)
	count, err := q.CountActiveAccountsByUser(ctx, userID)
	if err != nil {
		return Account{}, err
	}
	isPrimary := int64(0)
	if count == 0 {
		isPrimary = 1
	}
	if err := q.InsertAccount(ctx, sqlcdb.InsertAccountParams{
		ID:               id,
		UserID:           userID,
		Name:             in.Name,
		Type:             in.Type,
		BankID:           in.BankID,
		InitialBalance:   in.InitialBalance,
		CurrentBalance:   in.InitialBalance,
		CreditLimit:      creditLimit,
		PaymentAccountID: paymentAccountID,
		IsPrimary:        isPrimary,
		CreatedAt:        now,
		UpdatedAt:        now,
	}); err != nil {
		return Account{}, fmt.Errorf("insert account: %w", err)
	}
	return GetByID(ctx, db, userID, id)
}

type UpdateInput struct {
	Name             string
	BankID           *string
	InitialBalance   *int64
	CreditLimit      *int64
	PaymentAccountID *string
}

func Update(ctx context.Context, db *sql.DB, userID, id string, in UpdateInput) (Account, error) {
	existing, err := GetByID(ctx, db, userID, id)
	if err != nil {
		return Account{}, err
	}
	if existing.Status != "active" {
		return Account{}, ErrArchived
	}
	if err := validateNameUnique(ctx, db, userID, in.Name, id); err != nil {
		return Account{}, err
	}

	bankID := existing.BankID
	creditLimit := existing.CreditLimit
	paymentAccountID := existing.PaymentAccountID

	switch existing.Type {
	case "bank", "credit_card":
		if err := validateBankID(ctx, db, in.BankID); err != nil {
			return Account{}, err
		}
		bankID = in.BankID
	}
	if existing.Type == "credit_card" {
		cl := creditLimit
		if in.CreditLimit != nil {
			cl = in.CreditLimit
		}
		pa := paymentAccountID
		if in.PaymentAccountID != nil {
			if *in.PaymentAccountID == "" {
				pa = nil
			} else {
				pa = in.PaymentAccountID
			}
		}
		var validatedCL *int64
		var validatedPA *string
		validatedCL, validatedPA, err = validateAccountFields(ctx, db, userID, id, existing.Type, bankID, cl, pa)
		if err != nil {
			return Account{}, err
		}
		creditLimit = validatedCL
		paymentAccountID = validatedPA
	}

	balance := existing.InitialBalance
	if in.InitialBalance != nil {
		balance = *in.InitialBalance
	}

	now := time.Now().UTC().Format(time.RFC3339)
	if err := queries(db).UpdateAccount(ctx, sqlcdb.UpdateAccountParams{
		Name:             in.Name,
		BankID:           bankID,
		InitialBalance:   balance,
		CreditLimit:      creditLimit,
		PaymentAccountID: paymentAccountID,
		UpdatedAt:        now,
		ID:               id,
		UserID:           userID,
	}); err != nil {
		return Account{}, err
	}
	if err := accountbalance.Refresh(ctx, db, userID, id); err != nil {
		return Account{}, err
	}
	return GetByID(ctx, db, userID, id)
}

var ErrArchived = errors.New("account is archived")
var ErrCreditCardArchiveNotFullyPaid = errors.New("credit card must be fully paid before archive")

// RequiresBalanceTransfer reports whether deleting the account must move remaining funds first.
func RequiresBalanceTransfer(acc Account) bool {
	return !IsCreditCard(acc.Type) && acc.Balance > 0
}

// DeleteTransferDescription is the transfer comment when deleting an account with balance.
func DeleteTransferDescription(accountName string) string {
	return fmt.Sprintf("Удаление счёта \"%s\"", accountName)
}

// ArchiveTransferDescription is the transfer comment when archiving an account with balance.
func ArchiveTransferDescription(accountName string) string {
	return fmt.Sprintf("Архивация счёта \"%s\"", accountName)
}

func validateCreditCardFullyPaid(acc Account) error {
	if !IsCreditCard(acc.Type) {
		return nil
	}
	if acc.CreditLimit == nil || acc.Balance < *acc.CreditLimit {
		return ErrCreditCardArchiveNotFullyPaid
	}
	return nil
}

func SetStatus(ctx context.Context, db *sql.DB, userID, id, status string) (Account, error) {
	existing, err := GetByID(ctx, db, userID, id)
	if err != nil {
		return Account{}, err
	}
	if existing.Status == "deleted" {
		return Account{}, ErrArchived
	}
	if existing.Status == status {
		return existing, nil
	}
	if status == "archived" || status == "deleted" {
		if err := validateCreditCardFullyPaid(existing); err != nil {
			return Account{}, err
		}
	}
	inactiveTarget := status == "archived" || status == "deleted"
	wasPrimary := existing.IsPrimary && existing.Status == "active" && inactiveTarget
	now := time.Now().UTC().Format(time.RFC3339)
	q := queries(db)
	if inactiveTarget && existing.IsPrimary {
		if err := q.ClearAccountPrimaryFlag(ctx, sqlcdb.ClearAccountPrimaryFlagParams{ID: id, UserID: userID}); err != nil {
			return Account{}, err
		}
	}
	n, err := q.UpdateAccountStatus(ctx, sqlcdb.UpdateAccountStatusParams{
		Status:    status,
		UpdatedAt: now,
		ID:        id,
		UserID:    userID,
	})
	if err != nil {
		return Account{}, err
	}
	if n == 0 {
		return Account{}, ErrNotFound
	}
	if wasPrimary {
		if err := promoteNextPrimary(ctx, db, userID); err != nil {
			return Account{}, err
		}
	}
	return GetByID(ctx, db, userID, id)
}

func SetPrimary(ctx context.Context, db *sql.DB, userID, id string) (Account, error) {
	acc, err := GetByID(ctx, db, userID, id)
	if err != nil {
		return Account{}, err
	}
	if acc.Status != "active" {
		return Account{}, ErrArchived
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return Account{}, err
	}
	defer func() { _ = tx.Rollback() }()

	q := sqlcdb.New(tx)
	if err := q.ClearPrimaryAccounts(ctx, userID); err != nil {
		return Account{}, err
	}
	if err := q.SetAccountPrimary(ctx, sqlcdb.SetAccountPrimaryParams{ID: id, UserID: userID}); err != nil {
		return Account{}, err
	}
	if err := tx.Commit(); err != nil {
		return Account{}, err
	}
	return GetByID(ctx, db, userID, id)
}

func promoteNextPrimary(ctx context.Context, db *sql.DB, userID string) error {
	q := queries(db)
	nextID, err := q.FirstActiveAccountID(ctx, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if err := q.ClearPrimaryAccounts(ctx, userID); err != nil {
		return err
	}
	return q.SetAccountPrimary(ctx, sqlcdb.SetAccountPrimaryParams{ID: nextID, UserID: userID})
}

func Delete(ctx context.Context, db *sql.DB, userID, id string) error {
	_, err := SetStatus(ctx, db, userID, id, "deleted")
	return err
}

// IsCreditCard reports whether the account type is credit_card.
func IsCreditCard(accType string) bool {
	return accType == "credit_card"
}

func validateNameUnique(ctx context.Context, db *sql.DB, userID, name, excludeID string) error {
	name = strings.TrimSpace(name)
	if len(name) < 1 || len(name) > 64 {
		return ErrInvalidName
	}
	q := queries(db)
	var n int64
	var err error
	if excludeID != "" {
		n, err = q.CountActiveAccountsByNameExcluding(ctx, sqlcdb.CountActiveAccountsByNameExcludingParams{
			UserID: userID,
			Name:   name,
			ID:     excludeID,
		})
	} else {
		n, err = q.CountActiveAccountsByName(ctx, sqlcdb.CountActiveAccountsByNameParams{
			UserID: userID,
			Name:   name,
		})
	}
	if err != nil {
		return err
	}
	if n > 0 {
		return ErrNameTaken
	}
	return nil
}

var (
	ErrInvalidName           = errors.New("invalid account name")
	ErrNameTaken             = errors.New("account name already exists")
	ErrInvalidType           = errors.New("invalid account type")
	ErrBankRequired          = errors.New("bank_id required for bank account")
	ErrBankForbidden         = errors.New("bank_id not allowed for cash account")
	ErrBankNotFound          = errors.New("bank not found")
	ErrCreditLimitRequired   = errors.New("credit_limit required for credit card")
	ErrCreditLimitForbidden  = errors.New("credit_limit not allowed for this account type")
	ErrInvalidCreditLimit    = errors.New("credit_limit must be positive")
	ErrInvalidPaymentAccount = errors.New("invalid payment account")
)

func validateBankID(ctx context.Context, db *sql.DB, bankID *string) error {
	if bankID == nil || *bankID == "" {
		return ErrBankRequired
	}
	ok, err := bank.Exists(ctx, db, *bankID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrBankNotFound
	}
	return nil
}

func validatePaymentAccount(ctx context.Context, db *sql.DB, userID, selfID string, paymentAccountID *string) (*string, error) {
	if paymentAccountID == nil || *paymentAccountID == "" {
		return nil, nil
	}
	if selfID != "" && *paymentAccountID == selfID {
		return nil, ErrInvalidPaymentAccount
	}
	acc, err := GetByID(ctx, db, userID, *paymentAccountID)
	if errors.Is(err, ErrNotFound) {
		return nil, ErrInvalidPaymentAccount
	}
	if err != nil {
		return nil, err
	}
	if acc.Status != "active" || acc.Type == "credit_card" {
		return nil, ErrInvalidPaymentAccount
	}
	id := *paymentAccountID
	return &id, nil
}

func validateAccountFields(
	ctx context.Context,
	db *sql.DB,
	userID, selfID, accType string,
	bankID *string,
	creditLimit *int64,
	paymentAccountID *string,
) (*int64, *string, error) {
	switch accType {
	case "cash":
		if bankID != nil && *bankID != "" {
			return nil, nil, ErrBankForbidden
		}
		if creditLimit != nil {
			return nil, nil, ErrCreditLimitForbidden
		}
		if paymentAccountID != nil && *paymentAccountID != "" {
			return nil, nil, ErrInvalidPaymentAccount
		}
		return nil, nil, nil
	case "bank":
		if err := validateBankID(ctx, db, bankID); err != nil {
			return nil, nil, err
		}
		if creditLimit != nil {
			return nil, nil, ErrCreditLimitForbidden
		}
		if paymentAccountID != nil && *paymentAccountID != "" {
			return nil, nil, ErrInvalidPaymentAccount
		}
		return nil, nil, nil
	case "credit_card":
		if err := validateBankID(ctx, db, bankID); err != nil {
			return nil, nil, err
		}
		if creditLimit == nil || *creditLimit <= 0 {
			return nil, nil, ErrCreditLimitRequired
		}
		pa, err := validatePaymentAccount(ctx, db, userID, selfID, paymentAccountID)
		if err != nil {
			return nil, nil, err
		}
		cl := *creditLimit
		return &cl, pa, nil
	default:
		return nil, nil, ErrInvalidType
	}
}
