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
		return acc.Balance, nil
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
	db *sql.DB,
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
	_, err := transaction.CreateTransferForAccountDelete(ctx, db, userID, transaction.TransferInput{
		FromAccountID:   fromID,
		ToAccountID:     toID,
		Amount:          amount,
		Description:     &desc,
		TransactionDate: time.Now().UTC(),
	})
	return err
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
