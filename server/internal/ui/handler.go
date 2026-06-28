package ui

import (
	"encoding/json"
	"net/http"

	"github.com/kai-zer-ru/buhgalter/internal/account"
	"github.com/kai-zer-ru/buhgalter/internal/apperror"
	"github.com/kai-zer-ru/buhgalter/internal/auth"
	"github.com/kai-zer-ru/buhgalter/internal/bank"
	"github.com/kai-zer-ru/buhgalter/internal/category"
	"github.com/kai-zer-ru/buhgalter/internal/credit"
	"github.com/kai-zer-ru/buhgalter/internal/db"
	"github.com/kai-zer-ru/buhgalter/internal/debt"
)

type Handler struct {
	Store *db.Handle
}

type AccountRef struct {
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	Type   string  `json:"type"`
	Status string  `json:"status"`
	BankID *string `json:"bank_id,omitempty"`
}

type MetaResponse struct {
	Accounts          []AccountRef        `json:"accounts"`
	Banks             []bank.Bank         `json:"banks"`
	ExpenseCategories []category.Category `json:"expense_categories"`
	IncomeCategories  []category.Category `json:"income_categories"`
	Debtors           []debt.Debtor       `json:"debtors"`
	ActiveCredits     []credit.Credit     `json:"active_credits"`
	ClosedCredits     []credit.Credit     `json:"closed_credits"`
}

func (h *Handler) Meta(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	ctx := r.Context()
	sqlDB := h.Store.DB()
	userID := info.User.ID

	accountRows, err := account.ListByUser(ctx, sqlDB, userID, "")
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}
	accounts := make([]AccountRef, 0, len(accountRows))
	for _, row := range accountRows {
		accounts = append(accounts, AccountRef{
			ID: row.ID, Name: row.Name, Type: row.Type, Status: row.Status, BankID: row.BankID,
		})
	}

	banks, err := bank.ListAll(ctx, sqlDB)
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}

	expenseCategories, err := category.ListByUser(ctx, sqlDB, userID, "expense")
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}
	incomeCategories, err := category.ListByUser(ctx, sqlDB, userID, "income")
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}

	debtors, err := debt.ListDebtors(ctx, sqlDB, userID)
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}

	activeCredits, err := credit.List(ctx, sqlDB, userID, "active")
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}
	closedCredits, err := credit.List(ctx, sqlDB, userID, "closed")
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}

	writeJSON(w, http.StatusOK, MetaResponse{
		Accounts:          accounts,
		Banks:             banks,
		ExpenseCategories: expenseCategories,
		IncomeCategories:  incomeCategories,
		Debtors:           debtors,
		ActiveCredits:     activeCredits,
		ClosedCredits:     closedCredits,
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
