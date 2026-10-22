package admin

import (
	"encoding/json"
	"net/http"

	"github.com/kai-zer-ru/buhgalter/internal/apperror"
	"github.com/kai-zer-ru/buhgalter/internal/auth"
	"github.com/kai-zer-ru/buhgalter/internal/features"
)

type adminFlagItem struct {
	Key            string `json:"key"`
	Enabled        bool   `json:"enabled"`
	TitleKey       string `json:"title_key"`
	DescriptionKey string `json:"description_key"`
	DefaultEnabled bool   `json:"default_enabled"`
}

// GetFeaturesSnapshot returns { key: bool } for authenticated clients.
func (h *Handler) GetFeaturesSnapshot(w http.ResponseWriter, r *http.Request) {
	snap, err := features.Snapshot(r.Context(), h.Store.DB())
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}
	writeJSON(w, http.StatusOK, snap)
}

// GetFeatures returns catalog metadata + enabled state for the admin form.
func (h *Handler) GetFeatures(w http.ResponseWriter, r *http.Request) {
	snap, err := features.Snapshot(r.Context(), h.Store.DB())
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}
	items := make([]adminFlagItem, 0, len(features.Registry))
	for _, f := range features.Registry {
		items = append(items, adminFlagItem{
			Key:            f.Key,
			Enabled:        snap[f.Key],
			TitleKey:       f.TitleKey,
			DescriptionKey: f.DescriptionKey,
			DefaultEnabled: f.DefaultEnabled,
		})
	}
	writeJSON(w, http.StatusOK, items)
}

// PutFeatures upserts overrides from a partial or full map { key: bool }.
func (h *Handler) PutFeatures(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}

	var req map[string]bool
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_INVALID_JSON")
		return
	}
	if len(req) == 0 {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_FEATURES_EMPTY")
		return
	}
	for key := range req {
		if !features.IsKnown(key) {
			apperror.WriteField(w, http.StatusBadRequest, apperror.ValidationError, "unknown feature flag", "features")
			return
		}
	}

	if err := features.SetMany(r.Context(), h.Store.DB(), req); err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}

	ip := auth.ClientIP(r)
	_ = h.Audit.Log("admin.features.update", info.User.ID, info.User.Login, ip, map[string]any{
		"updates": req,
	})

	snap, err := features.Snapshot(r.Context(), h.Store.DB())
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}
	writeJSON(w, http.StatusOK, snap)
}
