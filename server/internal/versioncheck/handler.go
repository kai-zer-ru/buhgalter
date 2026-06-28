package versioncheck

import (
	"encoding/json"
	"net/http"
)

type Handler struct {
	Checker *Checker
}

func (h *Handler) Check(w http.ResponseWriter, r *http.Request) {
	result := h.Checker.Check(r.Context())
	writeJSON(w, http.StatusOK, result)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
