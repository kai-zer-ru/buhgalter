// Package currency serves the currency allowlist from internal/money.
package currency

import (
	"encoding/json"
	"net/http"

	"github.com/kai-zer-ru/buhgalter/internal/money"
)

// List handles GET /api/v1/currencies: [{code, symbol}] in allowlist order, no auth.
func List(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(money.Currencies())
}
