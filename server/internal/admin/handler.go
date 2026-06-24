package admin

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/kai-zer-ru/buhgalter/internal/apperror"
	"github.com/kai-zer-ru/buhgalter/internal/audit"
	"github.com/kai-zer-ru/buhgalter/internal/auth"
	"github.com/kai-zer-ru/buhgalter/internal/config"
	"github.com/kai-zer-ru/buhgalter/internal/db"
	"github.com/kai-zer-ru/buhgalter/internal/notify"
)

type Handler struct {
	Store   *db.Handle
	Audit   *audit.Logger
	Config  config.Config
	Started time.Time
}

type settingsResponse struct {
	RegistrationEnabled bool   `json:"registration_enabled"`
	ExternalURL         string `json:"external_url"`
	SecretKeySet        bool   `json:"secret_key_set"`
}

type settingsRequest struct {
	RegistrationEnabled bool   `json:"registration_enabled"`
	ExternalURL         string `json:"external_url"`
}

type notificationSecretKeyRequest struct {
	NotificationSecretKey string `json:"notification_secret_key"`
}

type userItem struct {
	ID          string `json:"id"`
	Login       string `json:"login"`
	DisplayName string `json:"display_name"`
	IsAdmin     bool   `json:"is_admin"`
	CreatedAt   string `json:"created_at"`
}

type createUserRequest struct {
	Login           string `json:"login"`
	Password        string `json:"password"`
	PasswordConfirm string `json:"password_confirm"`
	DisplayName     string `json:"display_name"`
	IsAdmin         bool   `json:"is_admin"`
}

type diagnosticsResponse struct {
	AppVersion         string            `json:"app_version"`
	BuildCommit        string            `json:"build_commit"`
	BuildTime          string            `json:"build_time"`
	DBMigrationVersion int64             `json:"db_migration_version"`
	InstallMethod      string            `json:"install_method"`
	PreviousAppVersion *string           `json:"previous_app_version"`
	GoVersion          string            `json:"go_version"`
	OS                 string            `json:"os"`
	Arch               string            `json:"arch"`
	UptimeSeconds      int64             `json:"uptime_seconds"`
	DBSizeBytes        int64             `json:"db_size_bytes"`
	UsersCount         int64             `json:"users_count"`
	DataDir            string            `json:"data_dir"`
	LogDir             string            `json:"log_dir"`
	Addr               string            `json:"addr"`
	StaticEmbed        bool              `json:"static_embed"`
	ExternalURL        string            `json:"external_url"`
	Env                map[string]string `json:"env"`
}

func (h *Handler) GetSettings(w http.ResponseWriter, r *http.Request) {
	var reg int
	var externalURL sql.NullString
	var secretKey sql.NullString
	err := h.Store.DB().QueryRowContext(r.Context(), `
		SELECT registration_enabled, external_url, notification_secret_key FROM system_settings WHERE id = 1`,
	).Scan(&reg, &externalURL, &secretKey)
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}

	resp := settingsResponse{RegistrationEnabled: reg == 1}
	if externalURL.Valid {
		resp.ExternalURL = externalURL.String
	}
	resp.SecretKeySet = secretKey.Valid && strings.TrimSpace(secretKey.String) != ""
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) PutSettings(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}

	var req settingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_INVALID_JSON")
		return
	}

	externalURL := strings.TrimSpace(req.ExternalURL)
	if externalURL != "" && !strings.HasPrefix(externalURL, "http://") && !strings.HasPrefix(externalURL, "https://") {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_EXTERNAL_URL")
		return
	}

	reg := 0
	if req.RegistrationEnabled {
		reg = 1
	}
	var external any
	if externalURL == "" {
		external = nil
	} else {
		external = externalURL
	}
	_, err := h.Store.DB().ExecContext(r.Context(), `
		UPDATE system_settings
		SET registration_enabled = ?, external_url = ?, updated_at = datetime('now')
		WHERE id = 1`, reg, external,
	)
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}

	ip := auth.ClientIP(r)
	_ = h.Audit.Log("admin.settings.update", info.User.ID, info.User.Login, ip, map[string]any{
		"registration_enabled": req.RegistrationEnabled,
		"external_url_set":     externalURL != "",
	})

	var currentSecret string
	_ = h.Store.DB().QueryRowContext(r.Context(), `SELECT notification_secret_key FROM system_settings WHERE id = 1`).Scan(&currentSecret)
	writeJSON(w, http.StatusOK, settingsResponse{
		RegistrationEnabled: req.RegistrationEnabled,
		ExternalURL:         externalURL,
		SecretKeySet:        strings.TrimSpace(currentSecret) != "",
	})
}

func (h *Handler) PutNotificationSecretKey(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}

	var req notificationSecretKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_INVALID_JSON")
		return
	}
	secret := strings.TrimSpace(req.NotificationSecretKey)
	if secret == "" {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_SECRET_KEY_EMPTY")
		return
	}
	if _, err := notify.NewSecretBox(secret); err != nil {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_SECRET_KEY_INVALID")
		return
	}

	_, err := h.Store.DB().ExecContext(r.Context(), `
		UPDATE system_settings
		SET notification_secret_key = ?, updated_at = datetime('now')
		WHERE id = 1`, secret,
	)
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}

	ip := auth.ClientIP(r)
	_ = h.Audit.Log("admin.settings.secret_key.update", info.User.ID, info.User.Login, ip, map[string]any{
		"secret_key_set": true,
	})

	var reg int
	var externalURL sql.NullString
	var secretRaw sql.NullString
	if err := h.Store.DB().QueryRowContext(r.Context(), `
		SELECT registration_enabled, external_url, notification_secret_key FROM system_settings WHERE id = 1`,
	).Scan(&reg, &externalURL, &secretRaw); err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}
	resp := settingsResponse{RegistrationEnabled: reg == 1, SecretKeySet: strings.TrimSpace(secretRaw.String) != ""}
	if externalURL.Valid {
		resp.ExternalURL = externalURL.String
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	rows, err := h.Store.DB().QueryContext(r.Context(), `
		SELECT id, login, COALESCE(display_name, ''), is_admin, created_at
		FROM users ORDER BY created_at ASC`,
	)
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}
	defer rows.Close()

	var users []userItem
	for rows.Next() {
		var u userItem
		var isAdmin int
		if err := rows.Scan(&u.ID, &u.Login, &u.DisplayName, &isAdmin, &u.CreatedAt); err != nil {
			apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
			return
		}
		u.IsAdmin = isAdmin == 1
		users = append(users, u)
	}
	if users == nil {
		users = []userItem{}
	}
	writeJSON(w, http.StatusOK, users)
}

func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}

	var req createUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_INVALID_JSON")
		return
	}

	login := strings.TrimSpace(req.Login)
	displayName := strings.TrimSpace(req.DisplayName)
	if displayName == "" {
		displayName = login
	}
	if len(login) < 3 || len(login) > 32 {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_LOGIN_LENGTH")
		return
	}
	if err := auth.ValidatePassword(req.Password, login); err != nil {
		msgKey := apperror.PasswordTooWeak
		fallback := "пароль должен содержать минимум одну букву и одну цифру и не совпадать с логином"
		if err == auth.ErrPasswordTooShort {
			msgKey = apperror.PasswordTooShort
			fallback = "пароль должен быть не короче 8 символов"
		}
		apperror.WriteL(w, r, http.StatusBadRequest, apperror.ValidationError, msgKey, fallback)
		return
	}
	if req.Password != req.PasswordConfirm {
		apperror.WriteL(w, r, http.StatusBadRequest, apperror.ValidationError, apperror.PasswordsMismatch, "пароли не совпадают")
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}

	userID, err := auth.CreateUser(r.Context(), h.Store.DB(), login, hash, displayName, req.IsAdmin)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			apperror.WriteR(w, r, http.StatusConflict, apperror.ValidationError, "CONFLICT_LOGIN_TAKEN")
			return
		}
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}

	var createdAt string
	_ = h.Store.DB().QueryRowContext(r.Context(), `SELECT created_at FROM users WHERE id = ?`, userID).Scan(&createdAt)

	ip := auth.ClientIP(r)
	_ = h.Audit.Log("admin.user.create", info.User.ID, info.User.Login, ip, map[string]any{
		"target_login": login,
	})

	writeJSON(w, http.StatusCreated, userItem{
		ID:          userID,
		Login:       login,
		DisplayName: displayName,
		IsAdmin:     req.IsAdmin,
		CreatedAt:   createdAt,
	})
}

func (h *Handler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}

	targetID := chi.URLParam(r, "id")
	if targetID == info.User.ID {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_CANNOT_DELETE_SELF")
		return
	}

	var login string
	err := h.Store.DB().QueryRowContext(r.Context(), `SELECT login FROM users WHERE id = ?`, targetID).Scan(&login)
	if err != nil {
		apperror.WriteR(w, r, http.StatusNotFound, apperror.NotFound)
		return
	}

	_, err = h.Store.DB().ExecContext(r.Context(), `DELETE FROM users WHERE id = ?`, targetID)
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}

	ip := auth.ClientIP(r)
	_ = h.Audit.Log("admin.user.delete", info.User.ID, info.User.Login, ip, map[string]any{
		"target_login": login,
	})
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) GetDiagnostics(w http.ResponseWriter, r *http.Request) {
	var externalURL, previousVersion sql.NullString
	if err := h.Store.DB().QueryRowContext(r.Context(), `
		SELECT external_url, previous_app_version
		FROM system_settings
		WHERE id = 1`,
	).Scan(&externalURL, &previousVersion); err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}

	var usersCount int64
	if err := h.Store.DB().QueryRowContext(r.Context(), `SELECT COUNT(*) FROM users`).Scan(&usersCount); err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}

	dbMigrationVersion, err := h.loadMigrationVersion(r.Context())
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}

	resp := diagnosticsResponse{
		AppVersion:         h.Config.Version,
		BuildCommit:        strings.TrimSpace(h.Config.BuildCommit),
		BuildTime:          strings.TrimSpace(h.Config.BuildTime),
		DBMigrationVersion: dbMigrationVersion,
		InstallMethod:      h.resolveInstallMethod(),
		GoVersion:          runtime.Version(),
		OS:                 runtime.GOOS,
		Arch:               runtime.GOARCH,
		UptimeSeconds:      h.uptimeSeconds(),
		DBSizeBytes:        h.dbSizeBytes(),
		UsersCount:         usersCount,
		DataDir:            h.Config.DataDir,
		LogDir:             h.Config.LogDir,
		Addr:               h.Config.Addr,
		StaticEmbed:        h.Config.StaticEmbed,
		Env:                h.publicEnv(),
	}
	if externalURL.Valid {
		resp.ExternalURL = externalURL.String
	}
	if previousVersion.Valid && strings.TrimSpace(previousVersion.String) != "" {
		v := strings.TrimSpace(previousVersion.String)
		resp.PreviousAppVersion = &v
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) loadMigrationVersion(ctx context.Context) (int64, error) {
	var version int64
	err := h.Store.DB().QueryRowContext(ctx, `
		SELECT COALESCE(MAX(version_id), 0)
		FROM goose_db_version
		WHERE is_applied = 1`,
	).Scan(&version)
	return version, err
}

func (h *Handler) resolveInstallMethod() string {
	method := strings.TrimSpace(h.Config.InstallMethod)
	if method == "" {
		return "dev"
	}
	return method
}

func (h *Handler) uptimeSeconds() int64 {
	if h.Started.IsZero() {
		return 0
	}
	return int64(time.Since(h.Started).Seconds())
}

func (h *Handler) dbSizeBytes() int64 {
	total := int64(0)
	for _, path := range []string{
		h.Config.DBPath,
		h.Config.DBPath + "-wal",
		h.Config.DBPath + "-shm",
	} {
		if stat, err := os.Stat(path); err == nil {
			total += stat.Size()
		}
	}
	return total
}

func (h *Handler) publicEnv() map[string]string {
	return map[string]string{
		"BUHGALTER_ADDR":         h.Config.Addr,
		"BUHGALTER_DB_PATH":      filepath.Clean(h.Config.DBPath),
		"BUHGALTER_DATA_DIR":     filepath.Clean(h.Config.DataDir),
		"BUHGALTER_LOG_DIR":      filepath.Clean(h.Config.LogDir),
		"BUHGALTER_CORS_ORIGINS": strings.Join(h.Config.CORSOrigins, ","),
		"BUHGALTER_STATIC_EMBED": strconv.FormatBool(h.Config.StaticEmbed),
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
