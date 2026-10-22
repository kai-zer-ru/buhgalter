package httpserver

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/kai-zer-ru/buhgalter/internal/account"
	"github.com/kai-zer-ru/buhgalter/internal/accountbalance"
	"github.com/kai-zer-ru/buhgalter/internal/apperror"
	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
	"github.com/kai-zer-ru/buhgalter/internal/transaction"
)

type transferToRequest struct {
	TransferToAccountID *string `json:"transfer_to_account_id"`
}

func parseTransferToAccountID(r *http.Request, req transferToRequest) string {
	toID := strings.TrimSpace(r.URL.Query().Get("transfer_to_account_id"))
	if toID == "" && req.TransferToAccountID != nil {
		toID = strings.TrimSpace(*req.TransferToAccountID)
	}
	return toID
}

func accountTransferAmount(ctx context.Context, db *sql.DB, userID string, acc account.Account) (int64, error) {
	if account.IsCreditCard(acc.Type) {
		return 0, nil
	}
	computed, err := accountbalance.ComputeAll(ctx, db, userID)
	if err != nil {
		if acc.Status != "active" {
			return 0, nil
		}
		return acc.Balance, nil
	}
	if acc.Status != "active" {
		return computed[acc.ID], nil
	}
	return inactiveAccountTransferAmount(acc, computed), nil
}

func inactiveAccountTransferAmount(acc account.Account, computed map[string]int64) int64 {
	if account.IsCreditCard(acc.Type) {
		return 0
	}
	amount := computed[acc.ID]
	if acc.Balance > amount {
		return acc.Balance
	}
	return amount
}

func cashBankBalanceNeedsTransfer(acc account.Account, amount int64) bool {
	if account.IsCreditCard(acc.Type) {
		return false
	}
	return account.RequiresBalanceTransfer(acc) || amount > 0
}

var errTransferTargetRequired = errors.New("transfer target required")

func transferBalanceBeforeInactive(
	ctx context.Context,
	db sqlcdb.DBTX,
	userID, fromID, toID string,
	amount int64,
	description string,
) error {
	if amount <= 0 {
		return nil
	}
	if toID == "" {
		return errTransferTargetRequired
	}
	desc := description
	_, err := transaction.CreateTransferForAccountDeleteTx(ctx, db, userID, transaction.TransferInput{
		FromAccountID:   fromID,
		ToAccountID:     toID,
		Amount:          amount,
		Description:     &desc,
		TransactionDate: time.Now().UTC(),
	})
	return err
}

func inactivateAccount(
	ctx context.Context,
	db *sql.DB,
	userID, id, status, toID string,
) (account.Account, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return account.Account{}, err
	}
	defer func() { _ = tx.Rollback() }()

	if err := account.LockRow(ctx, tx, userID, id); err != nil {
		return account.Account{}, err
	}
	acc, err := account.GetByID(ctx, tx, userID, id)
	if err != nil {
		return account.Account{}, err
	}
	if acc.Status == "deleted" {
		return account.Account{}, account.ErrArchived
	}
	if acc.Status == status {
		return acc, nil
	}

	transferAmount, err := accountTransferAmount(ctx, db, userID, acc)
	if err != nil {
		return account.Account{}, err
	}

	transferDesc := account.ArchiveTransferDescription(acc.Name)
	if status == "deleted" {
		transferDesc = account.DeleteTransferDescription(acc.Name)
	}
	now := time.Now().UTC()
	transferredTo := ""
	if cashBankBalanceNeedsTransfer(acc, transferAmount) {
		if err := transferBalanceBeforeInactive(ctx, tx, userID, id, toID, transferAmount, transferDesc); err != nil {
			return account.Account{}, err
		}
		if transferAmount > 0 && toID != "" {
			transferredTo = toID
		}
	}

	updated, err := account.SetStatus(ctx, tx, userID, id, status)
	if err != nil {
		return account.Account{}, err
	}
	if err := tx.Commit(); err != nil {
		return account.Account{}, err
	}
	if transferredTo != "" {
		if err := transaction.RefreshBalances(ctx, db, userID, now, id, transferredTo); err == nil {
			if fresh, err := account.GetByID(ctx, db, userID, id); err == nil {
				updated = fresh
			}
		}
	}
	return updated, nil
}

func writeInactivateAccountError(w http.ResponseWriter, r *http.Request, err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, account.ErrNotFound) {
		apperror.WriteR(w, r, http.StatusNotFound, apperror.NotFound)
		return true
	}
	if errors.Is(err, account.ErrCreditCardArchiveNotFullyPaid) {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_CREDIT_CARD_ARCHIVE_NOT_FULLY_PAID")
		return true
	}
	return writeAccountTransferError(w, r, err)
}

func writeAccountTransferError(w http.ResponseWriter, r *http.Request, err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, errTransferTargetRequired) {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_ACCOUNT_TRANSFER_REQUIRED")
		return true
	}
	switch {
	case errors.Is(err, transaction.ErrInvalidAccount):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_ACCOUNT_NOT_FOUND")
	case errors.Is(err, transaction.ErrAccountArchived):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_ACCOUNT_ARCHIVED")
	case errors.Is(err, transaction.ErrSameAccount):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_TRANSFER_SAME_ACCOUNT")
	case errors.Is(err, transaction.ErrInvalidAmount):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_TX_AMOUNT_POSITIVE")
	case errors.Is(err, transaction.ErrCreditCardPaymentExceedsLimit):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_CREDIT_CARD_PAYMENT_EXCEEDS_LIMIT")
	case errors.Is(err, transaction.ErrTransferNotFound):
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
	default:
		if strings.Contains(err.Error(), "invalid timezone") {
			apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_INVALID_TIMEZONE")
		} else {
			apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		}
	}
	return true
}
