package ui

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/kai-zer-ru/buhgalter/internal/apperror"
	"github.com/kai-zer-ru/buhgalter/internal/auth"
	uilocales "github.com/kai-zer-ru/buhgalter/ui_locales"
)

type I18nResponse struct {
	Version  string            `json:"version"`
	Lang     string            `json:"lang"`
	Messages map[string]string `json:"messages"`
}

var (
	catalogOnce sync.Once
	catalogs    map[string]map[string]string
	catalogErr  error
)

func loadCatalogs() {
	catalogOnce.Do(func() {
		catalogs = make(map[string]map[string]string, 2)
		for _, lang := range []string{"ru", "en"} {
			raw, err := uilocales.Files.ReadFile(lang + ".json")
			if err != nil {
				catalogErr = err
				return
			}
			var messages map[string]string
			if err := json.Unmarshal(raw, &messages); err != nil {
				catalogErr = err
				return
			}
			catalogs[lang] = messages
		}
	})
}

// I18n returns the UI string catalog for Android remote sync when the APK is behind the server.
func (h *Handler) I18n(w http.ResponseWriter, r *http.Request) {
	if _, ok := auth.FromContext(r.Context()); !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}

	lang := strings.ToLower(strings.TrimSpace(chi.URLParam(r, "lang")))
	if lang != "ru" && lang != "en" {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_LANGUAGE")
		return
	}

	loadCatalogs()
	if catalogErr != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}
	messages := catalogs[lang]
	if messages == nil {
		apperror.WriteR(w, r, http.StatusNotFound, apperror.NotFound)
		return
	}

	version := h.Version
	if version == "" {
		version = "0.0.0"
	}
	writeJSON(w, http.StatusOK, I18nResponse{
		Version:  version,
		Lang:     lang,
		Messages: messages,
	})
}
