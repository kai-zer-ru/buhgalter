package bank

import (
	"encoding/json"
	"net/http"

	"github.com/kai-zer-ru/buhgalter/internal/apperror"
	"github.com/kai-zer-ru/buhgalter/internal/db"
)

type Handler struct {
	Store *db.Handle
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	banks, err := ListAll(r.Context(), h.Store.DB())
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}
	if banks == nil {
		banks = []Bank{}
	}
	writeJSON(w, http.StatusOK, banks)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
