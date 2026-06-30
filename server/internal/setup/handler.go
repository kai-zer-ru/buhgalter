package setup

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/kai-zer-ru/buhgalter/internal/apperror"
	"github.com/kai-zer-ru/buhgalter/internal/audit"
	"github.com/kai-zer-ru/buhgalter/internal/auth"
	"github.com/kai-zer-ru/buhgalter/internal/backup"
	"github.com/kai-zer-ru/buhgalter/internal/bank"
	"github.com/kai-zer-ru/buhgalter/internal/categoryseed"
	"github.com/kai-zer-ru/buhgalter/internal/db"
	"github.com/kai-zer-ru/buhgalter/internal/settingscache"

	"github.com/google/uuid"
)

type Handler struct {
	DataDir string
	Store   *db.Handle
	Audit   *audit.Logger
	Backup  *backup.Service
}

type statusResponse struct {
	Configured          bool   `json:"configured"`
	Database            string `json:"database"`
	RegistrationEnabled bool   `json:"registration_enabled"`
	ExternalURL         string `json:"external_url"`
}

type setupRequest struct {
	AdminLogin           string `json:"admin_login"`
	AdminDisplayName     string `json:"admin_display_name"`
	AdminPassword        string `json:"admin_password"`
	AdminPasswordConfirm string `json:"admin_password_confirm"`
	RegistrationEnabled  bool   `json:"registration_enabled"`
	ExternalURL          string `json:"external_url"`
}

type setupResponse struct {
	Message string `json:"message"`
}

type restoreResponse struct {
	Message    string `json:"message"`
	Configured bool   `json:"configured"`
}

func (h *Handler) Status(w http.ResponseWriter, r *http.Request) {
	configured := syncConfiguredMarker(h.DataDir, h.Store.DB())

	var regEnabled int
	var externalURL sql.NullString
	_ = h.Store.DB().QueryRowContext(r.Context(), `
		SELECT registration_enabled, external_url FROM system_settings WHERE id = 1`,
	).Scan(&regEnabled, &externalURL)

	resp := statusResponse{
		Configured:          configured,
		Database:            "SQLite",
		RegistrationEnabled: regEnabled == 1,
	}
	if externalURL.Valid {
		resp.ExternalURL = externalURL.String
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) Setup(w http.ResponseWriter, r *http.Request) {
	if syncConfiguredMarker(h.DataDir, h.Store.DB()) {
		apperror.WriteR(w, r, http.StatusConflict, apperror.AlreadyConfigured)
		return
	}
	if hasAdmin, err := adminUserExists(r.Context(), h.Store.DB()); err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	} else if hasAdmin {
		apperror.WriteR(w, r, http.StatusConflict, apperror.AlreadyConfigured)
		return
	}

	var req setupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_INVALID_JSON")
		return
	}

	login := strings.TrimSpace(req.AdminLogin)
	if len(login) < 3 || len(login) > 32 {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_LOGIN_LENGTH")
		return
	}
	displayName := strings.TrimSpace(req.AdminDisplayName)
	if displayName == "" {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_DISPLAY_NAME_REQUIRED")
		return
	}
	if len(displayName) > 64 {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_DISPLAY_NAME_LENGTH")
		return
	}
	if err := auth.ValidatePassword(req.AdminPassword, login); err != nil {
		msgKey := apperror.PasswordTooWeak
		fallback := "пароль должен содержать минимум одну букву и одну цифру и не совпадать с логином"
		if err == auth.ErrPasswordTooShort {
			msgKey = apperror.PasswordTooShort
			fallback = "пароль должен быть не короче 8 символов"
		}
		apperror.WriteL(w, r, http.StatusBadRequest, apperror.ValidationError, msgKey, fallback)
		return
	}
	if req.AdminPassword != req.AdminPasswordConfirm {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError)
		return
	}

	externalURL := strings.TrimSpace(req.ExternalURL)
	if externalURL != "" && !strings.HasPrefix(externalURL, "http://") && !strings.HasPrefix(externalURL, "https://") {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_EXTERNAL_URL")
		return
	}

	hash, err := auth.HashPassword(req.AdminPassword)
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}

	userID := uuid.NewString()
	tx, err := h.Store.DB().Begin()
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}
	defer func() { _ = tx.Rollback() }()

	_, err = tx.Exec(`
		INSERT INTO users (id, login, password_hash, display_name, is_admin, status)
		VALUES (?, ?, ?, ?, 1, 'active')`,
		userID, login, hash, displayName,
	)
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}

	if err := categoryseed.SeedDefaults(r.Context(), tx, userID); err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}

	var external any
	if externalURL == "" {
		external = nil
	} else {
		external = externalURL
	}

	reg := 0
	if req.RegistrationEnabled {
		reg = 1
	}

	_, err = tx.Exec(`
		UPDATE system_settings
		SET is_configured = 1,
		    external_url = ?,
		    registration_enabled = ?,
		    updated_at = datetime('now')
		WHERE id = 1`,
		external, reg,
	)
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}

	if err := tx.Commit(); err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}

	if err := MarkConfigured(h.DataDir); err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}

	if err := bank.SeedIfEmpty(r.Context(), h.Store.DB()); err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}
	settingscache.Invalidate()

	ip := r.RemoteAddr
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		ip = strings.Split(fwd, ",")[0]
	}
	_ = h.Audit.Log("setup.complete", userID, login, strings.TrimSpace(ip), map[string]any{
		"registration_enabled": req.RegistrationEnabled,
		"external_url_set":     externalURL != "",
	})

	writeJSON(w, http.StatusCreated, setupResponse{Message: "setup_complete"})
}

func (h *Handler) Restore(w http.ResponseWriter, r *http.Request) {
	if h.Backup == nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}
	if syncConfiguredMarker(h.DataDir, h.Store.DB()) {
		apperror.WriteR(w, r, http.StatusConflict, apperror.AlreadyConfigured)
		return
	}
	if hasAdmin, err := adminUserExists(r.Context(), h.Store.DB()); err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	} else if hasAdmin {
		apperror.WriteR(w, r, http.StatusConflict, apperror.AlreadyConfigured)
		return
	}

	if err := r.ParseMultipartForm(64 << 20); err != nil {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_MULTIPART")
		return
	}
	if strings.TrimSpace(r.FormValue("confirm")) != "RESTORE" {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_RESTORE_CONFIRM")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_FILE_REQUIRED")
		return
	}
	defer file.Close()

	if strings.ToLower(filepath.Ext(header.Filename)) != ".db" {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_DB_FILE_REQUIRED")
		return
	}

	if err := h.Backup.Restore(file); err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}

	configured := syncConfiguredMarker(h.DataDir, h.Store.DB())
	_ = h.Audit.Log("setup.restore", "", "", auth.ClientIP(r), map[string]any{
		"filename":   header.Filename,
		"configured": configured,
	})

	writeJSON(w, http.StatusOK, restoreResponse{
		Message:    "restore_complete",
		Configured: configured,
	})
}

// syncConfiguredMarker returns true when setup has completed on this instance.
// If SQLite is already configured but the marker file is missing (e.g. setup
// failed after commit), the marker is recreated — same as SyncMarkerFromDB on startup.
func syncConfiguredMarker(dataDir string, sqlDB *sql.DB) bool {
	if IsConfigured(dataDir) {
		return true
	}
	configured, err := db.IsConfigured(sqlDB)
	if err != nil || !configured {
		return false
	}
	_ = MarkConfigured(dataDir)
	return true
}

func adminUserExists(ctx context.Context, sqlDB *sql.DB) (bool, error) {
	var n int
	err := sqlDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM users WHERE is_admin = 1`).Scan(&n)
	return n > 0, err
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
