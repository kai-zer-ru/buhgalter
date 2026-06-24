//go:build !embedstatic

package static

import (
	"net/http"
)

func Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "frontend not built: run make build", http.StatusServiceUnavailable)
	})
}
