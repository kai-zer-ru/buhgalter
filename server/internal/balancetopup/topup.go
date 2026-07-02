package balancetopup

import (
	"context"
	"database/sql"
	"errors"
	"time"

	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
	"github.com/kai-zer-ru/buhgalter/internal/notify"
	"github.com/kai-zer-ru/buhgalter/internal/timeutil"
	"github.com/kai-zer-ru/buhgalter/internal/transaction"
)

const Description = "Автопополнение"

type accountSnap struct {
	id      string
	name    string
	accType string
	status  string
	balance int64
	enabled bool
	threshold,
	target *int64
	sourceID *string
}

// CheckAfterRefresh evaluates auto-topup for the given beneficiary account IDs.
func CheckAfterRefresh(ctx context.Context, db *sql.DB, userID string, accountIDs ...string) {
	for _, id := range accountIDs {
		if id == "" {
			continue
		}
		_, _ = ApplyIfNeeded(ctx, db, userID, id)
	}
}

// CheckAllForUser evaluates auto-topup for every enabled bank beneficiary of the user.
func CheckAllForUser(ctx context.Context, db *sql.DB, userID string) {
	q := sqlcdb.New(db)
	ids, err := q.ListAutoTopupBeneficiaryAccountIDs(ctx, userID)
	if err != nil {
		return
	}
	CheckAfterRefresh(ctx, db, userID, ids...)
}

// ApplyIfNeeded creates a transfer when the beneficiary balance is below the configured threshold.
func ApplyIfNeeded(ctx context.Context, db *sql.DB, userID, beneficiaryID string) (bool, error) {
	beneficiary, err := loadAccount(ctx, db, userID, beneficiaryID)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if beneficiary.accType != "bank" || beneficiary.status != "active" || !beneficiary.enabled {
		return false, nil
	}
	if beneficiary.threshold == nil || beneficiary.target == nil || beneficiary.sourceID == nil {
		return false, nil
	}
	if beneficiary.balance >= *beneficiary.threshold {
		return false, nil
	}
	transferAmount := *beneficiary.target - beneficiary.balance
	if transferAmount <= 0 {
		return false, nil
	}

	source, err := loadAccount(ctx, db, userID, *beneficiary.sourceID)
	if errors.Is(err, sql.ErrNoRows) {
		return disableAndNotify(ctx, db, userID, beneficiary, accountSnap{name: *beneficiary.sourceID}, transferAmount)
	}
	if err != nil {
		return false, err
	}
	if source.accType != "bank" || source.status != "active" || source.balance < transferAmount {
		return disableAndNotify(ctx, db, userID, beneficiary, source, transferAmount)
	}

	desc := Description
	now := timeutil.NowUTC()
	_, err = transaction.CreateTransfer(ctx, db, userID, transaction.TransferInput{
		FromAccountID:   source.id,
		ToAccountID:     beneficiary.id,
		Amount:          transferAmount,
		Commission:      0,
		Description:     &desc,
		TransactionDate: now,
	})
	if err != nil {
		return false, err
	}
	CheckAfterRefresh(ctx, db, userID, source.id)
	return true, nil
}

func loadAccount(ctx context.Context, db *sql.DB, userID, accountID string) (accountSnap, error) {
	row, err := sqlcdb.New(db).GetAccountByID(ctx, sqlcdb.GetAccountByIDParams{ID: accountID, UserID: userID})
	if err != nil {
		return accountSnap{}, err
	}
	return accountSnap{
		id:        row.ID,
		name:      row.Name,
		accType:   row.Type,
		status:    row.Status,
		balance:   row.CurrentBalance,
		enabled:   row.AutoTopupEnabled != 0,
		threshold: row.AutoTopupThreshold,
		target:    row.AutoTopupTarget,
		sourceID:  row.AutoTopupSourceAccountID,
	}, nil
}

func disableAndNotify(ctx context.Context, db *sql.DB, userID string, beneficiary, source accountSnap, transferAmount int64) (bool, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	q := sqlcdb.New(db)
	if err := q.DisableAutoTopup(ctx, sqlcdb.DisableAutoTopupParams{
		UpdatedAt: now,
		ID:        beneficiary.id,
		UserID:    userID,
	}); err != nil {
		return false, err
	}
	notifyAutoTopupDisabled(ctx, db, userID, beneficiary, source, transferAmount)
	return false, nil
}

func notifyAutoTopupDisabled(ctx context.Context, db *sql.DB, userID string, beneficiary, source accountSnap, transferAmount int64) {
	q := sqlcdb.New(db)
	if err := q.EnsureNotificationSettings(ctx, userID); err != nil {
		return
	}
	settings, err := q.GetNotificationSettings(ctx, userID)
	if err != nil || settings.TriggerAutoTopupDisabled != 1 {
		return
	}
	localeCode, _, currencyCode, err := notify.UserFormatting(ctx, db, userID)
	if err != nil {
		return
	}
	externalURL := notify.ResolveExternalURL(ctx, db)
	customTemplates, err := q.ListNotificationTemplates(ctx, userID)
	if err != nil {
		return
	}
	customMap := notify.ToTemplateMap(customTemplates)
	text, err := notify.Format(notify.TriggerAutoTopupDisabled, localeCode, customMap[notify.TriggerAutoTopupDisabled], notify.FormatData{
		"account":        beneficiary.name,
		"source_account": source.name,
		"amount":         notify.FormatAmountDisplay(transferAmount, currencyCode),
		"source_balance": notify.FormatAmountDisplay(source.balance, currencyCode),
		"account_url":    notify.AccountURLPlaceholderValue(externalURL, localeCode, beneficiary.id),
	})
	if err != nil {
		return
	}
	dedupDate := time.Now().UTC().Format("2006-01-02")
	notify.Deliver(ctx, db, settings, userID, notify.TriggerAutoTopupDisabled, beneficiary.id, dedupDate, text)
}
