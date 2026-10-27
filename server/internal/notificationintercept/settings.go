package notificationintercept

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"unicode"

	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
)

type BankBinding struct {
	PackageName string `json:"package_name"`
	BankID      string `json:"bank_id"`
	AccountID   string `json:"account_id"`
}

type CardBinding struct {
	BankID    string `json:"bank_id"`
	Last4     string `json:"last4"`
	AccountID string `json:"account_id"`
}

type SettingsView struct {
	BankBindings []BankBinding `json:"bank_bindings"`
	CardBindings []CardBinding `json:"card_bindings"`
}

type UpdateInput struct {
	BankBindings []BankBinding `json:"bank_bindings"`
	CardBindings []CardBinding `json:"card_bindings"`
}

func GetSettings(ctx context.Context, db *sql.DB, userID string) (SettingsView, error) {
	q := sqlcdb.New(db)
	banks, err := q.ListNotificationInterceptBankBindings(ctx, userID)
	if err != nil {
		return SettingsView{}, err
	}
	cards, err := q.ListNotificationInterceptCardBindings(ctx, userID)
	if err != nil {
		return SettingsView{}, err
	}
	view := SettingsView{
		BankBindings: make([]BankBinding, 0, len(banks)),
		CardBindings: make([]CardBinding, 0, len(cards)),
	}
	for _, b := range banks {
		view.BankBindings = append(view.BankBindings, BankBinding{
			PackageName: b.PackageName,
			BankID:      b.BankID,
			AccountID:   b.AccountID,
		})
	}
	for _, c := range cards {
		view.CardBindings = append(view.CardBindings, CardBinding{
			BankID:    c.BankID,
			Last4:     c.Last4,
			AccountID: c.AccountID,
		})
	}
	return view, nil
}

func ReplaceSettings(ctx context.Context, db *sql.DB, userID string, in UpdateInput) (SettingsView, error) {
	return ReplaceSettingsOwned(ctx, db, userID, []string{userID}, in)
}

func ReplaceSettingsOwned(ctx context.Context, db *sql.DB, userID string, ownerIDs []string, in UpdateInput) (SettingsView, error) {
	normalizeInput(&in)
	if len(ownerIDs) == 0 {
		ownerIDs = []string{userID}
	}
	if err := validateBindings(ctx, db, ownerIDs, in); err != nil {
		return SettingsView{}, err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return SettingsView{}, err
	}
	defer func() { _ = tx.Rollback() }()
	q := sqlcdb.New(tx)
	if err := q.DeleteNotificationInterceptBankBindings(ctx, userID); err != nil {
		return SettingsView{}, err
	}
	if err := q.DeleteNotificationInterceptCardBindings(ctx, userID); err != nil {
		return SettingsView{}, err
	}
	for _, b := range in.BankBindings {
		if err := q.InsertNotificationInterceptBankBinding(ctx, sqlcdb.InsertNotificationInterceptBankBindingParams{
			UserID:      userID,
			AccountID:   b.AccountID,
			BankID:      b.BankID,
			PackageName: strings.TrimSpace(b.PackageName),
		}); err != nil {
			return SettingsView{}, err
		}
	}
	for _, c := range in.CardBindings {
		if err := q.InsertNotificationInterceptCardBinding(ctx, sqlcdb.InsertNotificationInterceptCardBindingParams{
			UserID:    userID,
			BankID:    c.BankID,
			Last4:     c.Last4,
			AccountID: c.AccountID,
		}); err != nil {
			return SettingsView{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return SettingsView{}, err
	}
	return GetSettings(ctx, db, userID)
}

func normalizeInput(in *UpdateInput) {
	if in.BankBindings == nil {
		in.BankBindings = []BankBinding{}
	}
	if in.CardBindings == nil {
		in.CardBindings = []CardBinding{}
	}
	for i := range in.BankBindings {
		in.BankBindings[i].BankID = strings.TrimSpace(in.BankBindings[i].BankID)
		in.BankBindings[i].AccountID = strings.TrimSpace(in.BankBindings[i].AccountID)
		in.BankBindings[i].PackageName = strings.TrimSpace(in.BankBindings[i].PackageName)
	}
	for i := range in.CardBindings {
		in.CardBindings[i].BankID = strings.TrimSpace(in.CardBindings[i].BankID)
		in.CardBindings[i].AccountID = strings.TrimSpace(in.CardBindings[i].AccountID)
		in.CardBindings[i].Last4 = normalizeLast4(in.CardBindings[i].Last4)
	}
}

func validateBindings(ctx context.Context, db *sql.DB, ownerIDs []string, in UpdateInput) error {
	q := sqlcdb.New(db)
	seenAccounts := map[string]struct{}{}
	for _, b := range in.BankBindings {
		if b.AccountID == "" || b.BankID == "" || b.PackageName == "" {
			return fmt.Errorf("ERR_NOTIFICATION_INTERCEPT_BINDING_INVALID")
		}
		if _, dup := seenAccounts[b.AccountID]; dup {
			return fmt.Errorf("ERR_NOTIFICATION_INTERCEPT_BINDING_INVALID")
		}
		seenAccounts[b.AccountID] = struct{}{}
		if err := assertBindableAccount(ctx, q, ownerIDs, b.AccountID); err != nil {
			return err
		}
		ok, err := q.BankExists(ctx, b.BankID)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("ERR_NOTIFICATION_INTERCEPT_BANK_UNKNOWN")
		}
	}
	seenCards := map[string]struct{}{}
	for _, c := range in.CardBindings {
		if c.AccountID == "" || c.BankID == "" || len(c.Last4) != 4 {
			return fmt.Errorf("ERR_NOTIFICATION_INTERCEPT_BINDING_INVALID")
		}
		key := c.BankID + ":" + c.Last4
		if _, dup := seenCards[key]; dup {
			return fmt.Errorf("ERR_NOTIFICATION_INTERCEPT_BINDING_INVALID")
		}
		seenCards[key] = struct{}{}
		if err := assertBindableAccount(ctx, q, ownerIDs, c.AccountID); err != nil {
			return err
		}
		ok, err := q.BankExists(ctx, c.BankID)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("ERR_NOTIFICATION_INTERCEPT_BANK_UNKNOWN")
		}
	}
	return nil
}

func assertBindableAccount(ctx context.Context, q *sqlcdb.Queries, ownerIDs []string, accountID string) error {
	if len(ownerIDs) == 0 {
		return fmt.Errorf("ERR_ACCOUNT_NOT_FOUND")
	}
	acc, err := q.GetAccountByID(ctx, sqlcdb.GetAccountByIDParams{ID: accountID, OwnerIds: ownerIDs})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("ERR_ACCOUNT_NOT_FOUND")
		}
		return err
	}
	if acc.Type != "bank" && acc.Type != "credit_card" {
		return fmt.Errorf("ERR_NOTIFICATION_INTERCEPT_ACCOUNT_TYPE")
	}
	if acc.Status != "active" {
		return fmt.Errorf("ERR_ACCOUNT_ARCHIVED")
	}
	return nil
}

func normalizeLast4(value string) string {
	var digits strings.Builder
	for _, r := range value {
		if unicode.IsDigit(r) {
			digits.WriteRune(r)
		}
	}
	s := digits.String()
	if len(s) >= 4 {
		return s[len(s)-4:]
	}
	return s
}
