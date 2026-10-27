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
	"github.com/kai-zer-ru/buhgalter/internal/auth"
	"github.com/kai-zer-ru/buhgalter/internal/bank"
	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
	"github.com/kai-zer-ru/buhgalter/internal/features"
	"github.com/kai-zer-ru/buhgalter/internal/money"
	"github.com/kai-zer-ru/buhgalter/internal/scope"
)

type PersonRef struct {
	UserID      string `json:"user_id"`
	DisplayName string `json:"display_name"`
}

type Account struct {
	ID                        string     `json:"id"`
	OwnerUserID               string     `json:"-"`
	Scope                     string     `json:"scope"`
	OriginalOwner             *PersonRef `json:"original_owner"`
	Name                      string     `json:"name"`
	Type                      string     `json:"type"`
	BankID                    *string    `json:"bank_id"`
	BankName                  *string    `json:"bank_name,omitempty"`
	BankIcon                  *string    `json:"bank_icon,omitempty"`
	Currency                  string     `json:"currency"`
	InitialBalance            int64      `json:"initial_balance"`
	Balance                   int64      `json:"balance"`
	BalanceDisplay            string     `json:"balance_display"`
	CreditLimit               *int64     `json:"credit_limit,omitempty"`
	CreditLimitDisplay        *string    `json:"credit_limit_display,omitempty"`
	PaymentAccountID          *string    `json:"payment_account_id,omitempty"`
	AutoTopupEnabled          bool       `json:"auto_topup_enabled"`
	AutoTopupThreshold        *int64     `json:"auto_topup_threshold,omitempty"`
	AutoTopupThresholdDisplay *string    `json:"auto_topup_threshold_display,omitempty"`
	AutoTopupTarget           *int64     `json:"auto_topup_target,omitempty"`
	AutoTopupTargetDisplay    *string    `json:"auto_topup_target_display,omitempty"`
	AutoTopupSourceAccountID  *string    `json:"auto_topup_source_account_id,omitempty"`
	Status                    string     `json:"status"`
	IsPrimary                 bool       `json:"is_primary"`
	CreatedAt                 string     `json:"created_at"`
	UpdatedAt                 string     `json:"updated_at"`
}

func queries(db sqlcdb.DBTX) *sqlcdb.Queries {
	return sqlcdb.New(db)
}

func creditLimitDisplay(limit *int64) *string {
	if limit == nil {
		return nil
	}
	s := money.FormatRubles(*limit)
	return &s
}

func amountDisplayPtr(v *int64) *string {
	if v == nil {
		return nil
	}
	s := money.FormatRubles(*v)
	return &s
}

func originalOwnerRef(id *string, displayName string) *PersonRef {
	if id == nil || *id == "" {
		return nil
	}
	return &PersonRef{UserID: *id, DisplayName: displayName}
}

func decorateScope(a Account, viewer auth.Scope) Account {
	a.Scope = scope.AccountScope(viewer, a.OwnerUserID)
	return a
}

func accountFromRow(
	id, ownerUserID, name, accType, status, createdAt, updatedAt string,
	originalOwner *PersonRef,
	bankID *string,
	initialBalance, currentBalance int64,
	creditLimit *int64,
	paymentAccountID *string,
	autoTopupEnabled int64,
	autoTopupThreshold, autoTopupTarget *int64,
	autoTopupSourceAccountID *string,
	isPrimary int64,
	currency string,
	bankName, bankIcon *string,
) Account {
	a := Account{
		ID:                       id,
		OwnerUserID:              ownerUserID,
		OriginalOwner:            originalOwner,
		Name:                     name,
		Type:                     accType,
		BankID:                   bankID,
		BankName:                 bankName,
		BankIcon:                 bankIcon,
		Currency:                 currency,
		InitialBalance:           initialBalance,
		Status:                   status,
		IsPrimary:                isPrimary != 0,
		CreatedAt:                createdAt,
		UpdatedAt:                updatedAt,
		CreditLimit:              creditLimit,
		PaymentAccountID:         paymentAccountID,
		AutoTopupEnabled:         autoTopupEnabled != 0,
		AutoTopupThreshold:       autoTopupThreshold,
		AutoTopupTarget:          autoTopupTarget,
		AutoTopupSourceAccountID: autoTopupSourceAccountID,
	}
	a.Balance = currentBalance
	a.BalanceDisplay = money.FormatRubles(currentBalance)
	a.CreditLimitDisplay = creditLimitDisplay(creditLimit)
	a.AutoTopupThresholdDisplay = amountDisplayPtr(autoTopupThreshold)
	a.AutoTopupTargetDisplay = amountDisplayPtr(autoTopupTarget)
	return a
}

func accountFromGetRow(row sqlcdb.GetAccountByIDRow) Account {
	return accountFromRow(
		row.ID, row.UserID, row.Name, row.Type, row.Status, row.CreatedAt, row.UpdatedAt,
		originalOwnerRef(row.OriginalOwnerUserID, row.OriginalOwnerDisplayName),
		row.BankID, row.InitialBalance, row.CurrentBalance,
		row.CreditLimit, row.PaymentAccountID,
		row.AutoTopupEnabled, row.AutoTopupThreshold, row.AutoTopupTarget, row.AutoTopupSourceAccountID,
		row.IsPrimary, row.Currency, row.BankName, row.BankIcon,
	)
}

func accountFromListActive(row sqlcdb.ListAccountsByUserActiveRow) Account {
	return accountFromRow(
		row.ID, row.UserID, row.Name, row.Type, row.Status, row.CreatedAt, row.UpdatedAt,
		originalOwnerRef(row.OriginalOwnerUserID, row.OriginalOwnerDisplayName),
		row.BankID, row.InitialBalance, row.CurrentBalance,
		row.CreditLimit, row.PaymentAccountID,
		row.AutoTopupEnabled, row.AutoTopupThreshold, row.AutoTopupTarget, row.AutoTopupSourceAccountID,
		row.IsPrimary, row.Currency, row.BankName, row.BankIcon,
	)
}

func accountFromListStatus(row sqlcdb.ListAccountsByUserAndStatusRow) Account {
	return accountFromRow(
		row.ID, row.UserID, row.Name, row.Type, row.Status, row.CreatedAt, row.UpdatedAt,
		originalOwnerRef(row.OriginalOwnerUserID, row.OriginalOwnerDisplayName),
		row.BankID, row.InitialBalance, row.CurrentBalance,
		row.CreditLimit, row.PaymentAccountID,
		row.AutoTopupEnabled, row.AutoTopupThreshold, row.AutoTopupTarget, row.AutoTopupSourceAccountID,
		row.IsPrimary, row.Currency, row.BankName, row.BankIcon,
	)
}

func ListByUser(ctx context.Context, db *sql.DB, userID, status string) ([]Account, error) {
	return ListByScope(ctx, db, auth.Scope{UserID: userID}, status)
}

func ListByScope(ctx context.Context, db *sql.DB, sc auth.Scope, status string) ([]Account, error) {
	q := queries(db)
	ids := sc.OwnerIDs()
	if status != "" {
		rows, err := q.ListAccountsByUserAndStatus(ctx, sqlcdb.ListAccountsByUserAndStatusParams{
			OwnerIds: ids,
			Status:   status,
		})
		if err != nil {
			return nil, err
		}
		out := make([]Account, 0, len(rows))
		for _, row := range rows {
			out = append(out, decorateScope(accountFromListStatus(row), sc))
		}
		return out, nil
	}

	rows, err := q.ListAccountsByUserActive(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make([]Account, 0, len(rows))
	for _, row := range rows {
		out = append(out, decorateScope(accountFromListActive(row), sc))
	}
	return out, nil
}

func GetByID(ctx context.Context, db sqlcdb.DBTX, userID, id string) (Account, error) {
	return GetByScope(ctx, db, auth.Scope{UserID: userID}, id)
}

func GetByScope(ctx context.Context, db sqlcdb.DBTX, sc auth.Scope, id string) (Account, error) {
	row, err := queries(db).GetAccountByID(ctx, sqlcdb.GetAccountByIDParams{ID: id, OwnerIds: sc.OwnerIDs()})
	if errors.Is(err, sql.ErrNoRows) {
		return Account{}, ErrNotFound
	}
	if err != nil {
		return Account{}, err
	}
	return decorateScope(accountFromGetRow(row), sc), nil
}

// LockRow takes a write lock on the account row (SQLite deferred tx → reserved).
func LockRow(ctx context.Context, db sqlcdb.DBTX, userID, id string) error {
	n, err := queries(db).LockAccountRow(ctx, sqlcdb.LockAccountRowParams{ID: id, UserID: userID})
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func LockRowByScope(ctx context.Context, db sqlcdb.DBTX, sc auth.Scope, id string) error {
	acc, err := GetByScope(ctx, db, sc, id)
	if err != nil {
		return err
	}
	return LockRow(ctx, db, acc.OwnerUserID, id)
}

var ErrNotFound = errors.New("account not found")
var ErrArchived = errors.New("account is archived")
var ErrCreditCardPrimary = errors.New("credit card cannot be primary")
var ErrCreditCardArchiveNotFullyPaid = errors.New("credit card must be fully paid before archive")
var ErrCreditLimitImmutable = errors.New("credit limit cannot be changed via account update")
var ErrCreditCardLimitDecreaseNotFullyPaid = errors.New("credit card must be fully paid before decreasing limit")
var ErrInvalidCurrency = errors.New("invalid currency")
var ErrCurrencyLocked = errors.New("account currency cannot be changed: account is in use")
var ErrCurrencyMismatch = money.ErrCurrencyMismatch
var ErrNotHouseholdMember = errors.New("not household member")
var ErrSharedPrimary = errors.New("shared account cannot be primary")

type CreateInput struct {
	Name             string
	Type             string
	BankID           *string
	InitialBalance   int64
	CreditLimit      *int64
	PaymentAccountID *string
	// Currency is optional; empty means the owner's profile currency.
	Currency string
	Shared   bool
}

func Create(ctx context.Context, db *sql.DB, userID string, in CreateInput) (Account, error) {
	if err := validateNameUnique(ctx, db, userID, in.Name, ""); err != nil {
		return Account{}, err
	}
	currency, err := resolveCreateCurrency(ctx, db, userID, in.Currency)
	if err != nil {
		return Account{}, err
	}
	creditLimit, paymentAccountID, err := validateAccountFields(ctx, db, userID, "", in.Type, currency, in.BankID, in.CreditLimit, in.PaymentAccountID)
	if err != nil {
		return Account{}, err
	}

	id := uuid.NewString()
	now := time.Now().UTC().Format(time.RFC3339)
	q := queries(db)
	isPrimary := int64(0)
	if !in.Shared && !IsCreditCard(in.Type) {
		count, err := q.CountActivePrimaryEligibleAccountsByUser(ctx, userID)
		if err != nil {
			return Account{}, err
		}
		if count == 0 {
			isPrimary = 1
		}
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
		Currency:         currency,
		CreatedAt:        now,
		UpdatedAt:        now,
	}); err != nil {
		return Account{}, fmt.Errorf("insert account: %w", err)
	}
	acc, err := GetByID(ctx, db, userID, id)
	if err != nil {
		return Account{}, err
	}
	if in.Shared {
		acc.Scope = scope.Household
	}
	return acc, nil
}

type UpdateInput struct {
	Name             string
	BankID           *string
	InitialBalance   *int64
	CreditLimit      *int64
	PaymentAccountID *string
	AutoTopup        *AutoTopupInput
	// Currency nil (old clients) keeps the current currency.
	Currency *string
}

type AutoTopupInput struct {
	Enabled         bool
	Threshold       int64
	Target          int64
	SourceAccountID string
}

func Update(ctx context.Context, db *sql.DB, userID, id string, in UpdateInput) (Account, error) {
	return UpdateByScope(ctx, db, auth.Scope{UserID: userID}, id, in)
}

func UpdateByScope(ctx context.Context, db *sql.DB, sc auth.Scope, id string, in UpdateInput) (Account, error) {
	existing, err := GetByScope(ctx, db, sc, id)
	if err != nil {
		return Account{}, err
	}
	userID := existing.OwnerUserID
	if existing.Status != "active" {
		return Account{}, ErrArchived
	}
	if err := validateNameUnique(ctx, db, userID, in.Name, id); err != nil {
		return Account{}, err
	}
	newCurrency, err := resolveUpdateCurrency(ctx, db, userID, existing, in.Currency)
	if err != nil {
		return Account{}, err
	}
	currency := existing.Currency
	if newCurrency != "" {
		currency = newCurrency
	}
	if err := validateAutoTopupCurrency(ctx, db, userID, existing, currency, in.AutoTopup); err != nil {
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
		if in.CreditLimit != nil {
			if existing.CreditLimit == nil || *in.CreditLimit != *existing.CreditLimit {
				return Account{}, ErrCreditLimitImmutable
			}
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
		validatedCL, validatedPA, err = validateAccountFields(ctx, db, userID, id, existing.Type, currency, bankID, creditLimit, pa)
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
	if newCurrency != "" {
		if err := queries(db).UpdateAccountCurrency(ctx, sqlcdb.UpdateAccountCurrencyParams{
			Currency:  newCurrency,
			UpdatedAt: now,
			ID:        id,
			UserID:    userID,
		}); err != nil {
			return Account{}, err
		}
	}
	if err := accountbalance.Refresh(ctx, db, userID, id); err != nil {
		return Account{}, err
	}
	if in.AutoTopup != nil {
		if err := applyAutoTopupSettings(ctx, db, userID, id, existing.Type, *in.AutoTopup); err != nil {
			return Account{}, err
		}
	}
	acc, err := GetByID(ctx, db, userID, id)
	if err != nil {
		return Account{}, err
	}
	return decorateScope(acc, sc), nil
}

// resolveCreateCurrency validates an explicit code or falls back to the profile currency.
func resolveCreateCurrency(ctx context.Context, db sqlcdb.DBTX, userID, code string) (string, error) {
	code = money.Normalize(code)
	if code != "" {
		if !money.Valid(code) {
			return "", ErrInvalidCurrency
		}
		return code, nil
	}
	row, err := queries(db).GetUserFormatting(ctx, userID)
	if err != nil {
		return "", err
	}
	if c := money.Normalize(row.Currency); money.Valid(c) {
		return c, nil
	}
	return money.DefaultCurrency, nil
}

// ProfileCurrency returns the user's profile currency, or DefaultCurrency.
func ProfileCurrency(ctx context.Context, db sqlcdb.DBTX, userID string) (string, error) {
	row, err := queries(db).GetUserFormatting(ctx, userID)
	if err != nil {
		return "", err
	}
	return money.ProfileOrDefault(row.Currency), nil
}

// ActiveCurrencies is DISTINCT currencies of active accounts, profile first even if none.
func ActiveCurrencies(ctx context.Context, db sqlcdb.DBTX, userID string) ([]string, error) {
	return ActiveCurrenciesOwned(ctx, db, userID, []string{userID})
}

func ActiveCurrenciesOwned(ctx context.Context, db sqlcdb.DBTX, tzUserID string, ownerIDs []string) ([]string, error) {
	profile, err := ProfileCurrency(ctx, db, tzUserID)
	if err != nil {
		return nil, err
	}
	if len(ownerIDs) == 0 {
		return money.OrderWithProfileFirst(profile, nil), nil
	}
	rows, err := queries(db).ListAccountsByUserActive(ctx, ownerIDs)
	if err != nil {
		return nil, err
	}
	codes := make([]string, 0, len(rows))
	for _, row := range rows {
		codes = append(codes, row.Currency)
	}
	return money.OrderWithProfileFirst(profile, codes), nil
}

// resolveUpdateCurrency returns the new currency to store, or "" when it does not change.
// A change is allowed only while the account has no transactions and no references.
func resolveUpdateCurrency(ctx context.Context, db sqlcdb.DBTX, userID string, existing Account, code *string) (string, error) {
	if code == nil {
		return "", nil
	}
	c := money.Normalize(*code)
	if c == "" || c == existing.Currency {
		return "", nil
	}
	if !money.Valid(c) {
		return "", ErrInvalidCurrency
	}
	inUse, err := queries(db).AccountCurrencyInUse(ctx, sqlcdb.AccountCurrencyInUseParams{
		UserID:    userID,
		AccountID: existing.ID,
	})
	if err != nil {
		return "", err
	}
	if inUse != 0 {
		return "", ErrCurrencyLocked
	}
	return c, nil
}

// ChangeCreditLimit sets a new credit_limit and adjusts initial_balance by the same delta
// so debt (limit − balance) stays unchanged on increase. Decrease is allowed only when
// the card is fully paid (balance >= current credit_limit).
func ChangeCreditLimit(ctx context.Context, db *sql.DB, userID, id string, newLimit int64) (Account, error) {
	return ChangeCreditLimitByScope(ctx, db, auth.Scope{UserID: userID}, id, newLimit)
}

func ChangeCreditLimitByScope(ctx context.Context, db *sql.DB, sc auth.Scope, id string, newLimit int64) (Account, error) {
	existing, err := GetByScope(ctx, db, sc, id)
	if err != nil {
		return Account{}, err
	}
	if existing.Status != "active" {
		return Account{}, ErrArchived
	}
	if !IsCreditCard(existing.Type) {
		return Account{}, ErrInvalidType
	}
	if newLimit <= 0 {
		return Account{}, ErrInvalidCreditLimit
	}
	if existing.CreditLimit == nil {
		return Account{}, ErrCreditLimitRequired
	}
	oldLimit := *existing.CreditLimit
	if newLimit == oldLimit {
		return existing, nil
	}
	delta := newLimit - oldLimit
	if delta < 0 && existing.Balance < oldLimit {
		return Account{}, ErrCreditCardLimitDecreaseNotFullyPaid
	}
	newCL := newLimit
	now := time.Now().UTC().Format(time.RFC3339)
	owner := existing.OwnerUserID
	if err := queries(db).UpdateAccount(ctx, sqlcdb.UpdateAccountParams{
		Name:             existing.Name,
		BankID:           existing.BankID,
		InitialBalance:   existing.InitialBalance + delta,
		CreditLimit:      &newCL,
		PaymentAccountID: existing.PaymentAccountID,
		UpdatedAt:        now,
		ID:               id,
		UserID:           owner,
	}); err != nil {
		return Account{}, err
	}
	if err := accountbalance.Refresh(ctx, db, owner, id); err != nil {
		return Account{}, err
	}
	return GetByScope(ctx, db, sc, id)
}

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

func SetStatus(ctx context.Context, db sqlcdb.DBTX, userID, id, status string) (Account, error) {
	return SetStatusByScope(ctx, db, auth.Scope{UserID: userID}, id, status)
}

func SetStatusByScope(ctx context.Context, db sqlcdb.DBTX, sc auth.Scope, id, status string) (Account, error) {
	existing, err := GetByScope(ctx, db, sc, id)
	if err != nil {
		return Account{}, err
	}
	userID := existing.OwnerUserID
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
	if inactiveTarget {
		if err := disableAutoTopupForInactive(ctx, db, userID, id); err != nil {
			return Account{}, err
		}
	}
	return GetByScope(ctx, db, sc, id)
}

func SetPrimary(ctx context.Context, db *sql.DB, userID, id string) (Account, error) {
	return SetPrimaryByScope(ctx, db, auth.Scope{UserID: userID}, id)
}

func SetPrimaryByScope(ctx context.Context, db *sql.DB, sc auth.Scope, id string) (Account, error) {
	acc, err := GetByScope(ctx, db, sc, id)
	if err != nil {
		return Account{}, err
	}
	if acc.Scope == scope.Household || (sc.PrincipalID != "" && acc.OwnerUserID == sc.PrincipalID) {
		return Account{}, ErrSharedPrimary
	}
	if acc.Status != "active" {
		return Account{}, ErrArchived
	}
	if IsCreditCard(acc.Type) {
		return Account{}, ErrCreditCardPrimary
	}

	userID := sc.UserID
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
	return GetByScope(ctx, db, sc, id)
}

func promoteNextPrimary(ctx context.Context, db sqlcdb.DBTX, userID string) error {
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

func Delete(ctx context.Context, db sqlcdb.DBTX, userID, id string) error {
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
	ErrInvalidName               = errors.New("invalid account name")
	ErrNameTaken                 = errors.New("account name already exists")
	ErrInvalidType               = errors.New("invalid account type")
	ErrBankRequired              = errors.New("bank_id required for bank account")
	ErrBankForbidden             = errors.New("bank_id not allowed for cash account")
	ErrBankNotFound              = errors.New("bank not found")
	ErrCreditLimitRequired       = errors.New("credit_limit required for credit card")
	ErrCreditLimitForbidden      = errors.New("credit_limit not allowed for this account type")
	ErrInvalidCreditLimit        = errors.New("credit_limit must be positive")
	ErrInvalidPaymentAccount     = errors.New("invalid payment account")
	ErrAutoTopupNotAllowed       = errors.New("auto topup not allowed for this account type")
	ErrInvalidAutoTopupThreshold = errors.New("invalid auto topup threshold")
	ErrInvalidAutoTopupTarget    = errors.New("invalid auto topup target")
	ErrInvalidAutoTopupSource    = errors.New("invalid auto topup source account")
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

func validatePaymentAccount(ctx context.Context, db *sql.DB, userID, selfID, cardCurrency string, paymentAccountID *string) (*string, error) {
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
	if acc.Currency != cardCurrency {
		return nil, ErrCurrencyMismatch
	}
	id := *paymentAccountID
	return &id, nil
}

func validateAccountFields(
	ctx context.Context,
	db *sql.DB,
	userID, selfID, accType, currency string,
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
		pa, err := validatePaymentAccount(ctx, db, userID, selfID, currency, paymentAccountID)
		if err != nil {
			return nil, nil, err
		}
		cl := *creditLimit
		return &cl, pa, nil
	default:
		return nil, nil, ErrInvalidType
	}
}

func applyAutoTopupSettings(ctx context.Context, db *sql.DB, userID, accountID, accType string, in AutoTopupInput) error {
	if accType != "bank" {
		return ErrAutoTopupNotAllowed
	}
	enabled := int64(0)
	var threshold, target *int64
	var sourceID *string
	if in.Enabled {
		if ok, err := features.IsEnabled(ctx, db, features.BalanceMaintenance); err != nil {
			return err
		} else if !ok {
			return ErrAutoTopupNotAllowed
		}
		if err := validateAutoTopupAmounts(in.Threshold, in.Target); err != nil {
			return err
		}
		src, err := validateAutoTopupSource(ctx, db, userID, accountID, in.SourceAccountID)
		if err != nil {
			return err
		}
		enabled = 1
		threshold = &in.Threshold
		target = &in.Target
		sourceID = src
	}
	now := time.Now().UTC().Format(time.RFC3339)
	return queries(db).UpdateAccountAutoTopup(ctx, sqlcdb.UpdateAccountAutoTopupParams{
		AutoTopupEnabled:         enabled,
		AutoTopupThreshold:       threshold,
		AutoTopupTarget:          target,
		AutoTopupSourceAccountID: sourceID,
		UpdatedAt:                now,
		ID:                       accountID,
		UserID:                   userID,
	})
}

func validateAutoTopupAmounts(threshold, target int64) error {
	if threshold < 0 {
		return ErrInvalidAutoTopupThreshold
	}
	if target <= 0 {
		return ErrInvalidAutoTopupTarget
	}
	if threshold >= target {
		return ErrInvalidAutoTopupTarget
	}
	return nil
}

func validateAutoTopupSource(ctx context.Context, db *sql.DB, userID, beneficiaryID, sourceID string) (*string, error) {
	sourceID = strings.TrimSpace(sourceID)
	if sourceID == "" || sourceID == beneficiaryID {
		return nil, ErrInvalidAutoTopupSource
	}
	acc, err := GetByID(ctx, db, userID, sourceID)
	if errors.Is(err, ErrNotFound) {
		return nil, ErrInvalidAutoTopupSource
	}
	if err != nil {
		return nil, err
	}
	if acc.Status != "active" || acc.Type != "bank" {
		return nil, ErrInvalidAutoTopupSource
	}
	beneficiary, err := GetByID(ctx, db, userID, beneficiaryID)
	if err != nil {
		return nil, err
	}
	if acc.Currency != beneficiary.Currency {
		return nil, ErrCurrencyMismatch
	}
	return &sourceID, nil
}

// validateAutoTopupCurrency rejects, before any write, an auto top-up source whose currency
// differs from the beneficiary's (new or kept) currency.
func validateAutoTopupCurrency(ctx context.Context, db *sql.DB, userID string, existing Account, currency string, in *AutoTopupInput) error {
	var sourceID string
	switch {
	case in != nil && in.Enabled:
		sourceID = strings.TrimSpace(in.SourceAccountID)
	case in == nil && existing.AutoTopupEnabled && existing.AutoTopupSourceAccountID != nil:
		sourceID = *existing.AutoTopupSourceAccountID
	}
	if sourceID == "" || sourceID == existing.ID {
		return nil
	}
	src, err := GetByID(ctx, db, userID, sourceID)
	if errors.Is(err, ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if src.Currency != currency {
		return ErrCurrencyMismatch
	}
	return nil
}

func disableAutoTopupForInactive(ctx context.Context, db sqlcdb.DBTX, userID, accountID string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	q := queries(db)
	if err := q.DisableAutoTopupForBeneficiary(ctx, sqlcdb.DisableAutoTopupForBeneficiaryParams{
		UpdatedAt: now,
		ID:        accountID,
		UserID:    userID,
	}); err != nil {
		return err
	}
	src := accountID
	return q.DisableAutoTopupUsingSource(ctx, sqlcdb.DisableAutoTopupUsingSourceParams{
		UpdatedAt:                now,
		UserID:                   userID,
		AutoTopupSourceAccountID: &src,
	})
}
