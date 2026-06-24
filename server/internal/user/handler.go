package user

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/kai-zer-ru/buhgalter/internal/apperror"
	"github.com/kai-zer-ru/buhgalter/internal/audit"
	"github.com/kai-zer-ru/buhgalter/internal/auth"
	"github.com/kai-zer-ru/buhgalter/internal/db"
	"github.com/kai-zer-ru/buhgalter/internal/notify"
)

type Handler struct {
	Store *db.Handle
	Audit *audit.Logger
}

type settingsRequest struct {
	DisplayName string `json:"display_name"`
	Language    string `json:"language"`
	Currency    string `json:"currency"`
	Timezone    string `json:"timezone"`
	Theme       string `json:"theme"`
}

type passwordRequest struct {
	CurrentPassword    string `json:"current_password"`
	NewPassword        string `json:"new_password"`
	NewPasswordConfirm string `json:"new_password_confirm"`
	OldPassword        string `json:"old_password"`
}

type tokenListItem struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	TokenPrefix string  `json:"token_prefix"`
	ExpiresAt   *string `json:"expires_at"`
	LastUsedAt  *string `json:"last_used_at"`
	CreatedAt   string  `json:"created_at"`
}

type createTokenRequest struct {
	Name      string  `json:"name"`
	ExpiresAt *string `json:"expires_at"`
}

type createTokenResponse struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Token       string  `json:"token"`
	TokenPrefix string  `json:"token_prefix"`
	ExpiresAt   *string `json:"expires_at"`
	CreatedAt   string  `json:"created_at"`
}

type notificationTestRequest struct {
	Channel string `json:"channel"`
}

type notificationPreviewRequest struct {
	TriggerType string `json:"trigger_type"`
	Template    string `json:"template"`
}

type notificationResetRequest struct {
	TriggerType *string `json:"trigger_type"`
}

func (h *Handler) GetSettings(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"display_name": info.User.DisplayName,
		"language":     info.User.Language,
		"currency":     info.User.Currency,
		"timezone":     info.User.Timezone,
		"theme":        info.User.Theme,
	})
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

	if req.Language != "" && req.Language != "ru" && req.Language != "en" {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_LANGUAGE")
		return
	}
	if req.Currency != "" && req.Currency != "RUB" && req.Currency != "USD" && req.Currency != "EUR" {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_CURRENCY")
		return
	}
	if req.Theme != "" && req.Theme != "light" && req.Theme != "dark" {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_THEME")
		return
	}
	if req.Timezone != "" {
		if _, err := time.LoadLocation(req.Timezone); err != nil {
			apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_TIMEZONE")
			return
		}
	}

	displayName := strings.TrimSpace(req.DisplayName)
	if displayName == "" {
		displayName = info.User.DisplayName
	}
	language := req.Language
	if language == "" {
		language = info.User.Language
	}
	currency := req.Currency
	if currency == "" {
		currency = info.User.Currency
	}
	timezone := req.Timezone
	if timezone == "" {
		timezone = info.User.Timezone
	}
	theme := req.Theme
	if theme == "" {
		theme = info.User.Theme
	}

	_, err := h.Store.DB().ExecContext(r.Context(), `
		UPDATE users
		SET display_name = ?, language = ?, currency = ?, timezone = ?, theme = ?,
		    updated_at = datetime('now')
		WHERE id = ?`,
		displayName, language, currency, timezone, theme, info.User.ID,
	)
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"display_name": displayName,
		"language":     language,
		"currency":     currency,
		"timezone":     timezone,
		"theme":        theme,
	})
}

func (h *Handler) GetNotifications(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	view, err := notify.GetSettings(r.Context(), h.Store.DB(), info.User.ID)
	if err != nil {
		if db.IsBusy(err) {
			slog.Warn("get notifications: database busy", "user_id", info.User.ID, "err", err)
			apperror.WriteR(w, r, http.StatusServiceUnavailable, apperror.ServiceUnavailable)
			return
		}
		slog.Warn("get notifications failed", "user_id", info.User.ID, "err", err)
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func (h *Handler) PutNotifications(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	var req notify.UpdateSettingsInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_INVALID_JSON")
		return
	}
	box, err := h.secretBoxForUpdate(r.Context(), req)
	if err != nil {
		apperror.WriteDetail(w, r, http.StatusBadRequest, apperror.ValidationError, apperror.ValidationError, err.Error())
		return
	}
	view, err := notify.UpdateSettings(r.Context(), h.Store.DB(), info.User.ID, req, box)
	if err != nil {
		apperror.WriteDetail(w, r, http.StatusBadRequest, apperror.ValidationError, apperror.ValidationError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func (h *Handler) SendNotificationTest(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	var req notificationTestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_INVALID_JSON")
		return
	}
	channel := strings.TrimSpace(req.Channel)
	if channel != notify.ChannelTelegram && channel != notify.ChannelMax {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_CHANNEL")
		return
	}
	box, err := h.mustSecretBox(r.Context())
	if err != nil {
		apperror.WriteDetail(w, r, http.StatusBadRequest, apperror.ValidationError, apperror.ValidationError, err.Error())
		return
	}
	if err := notify.SendTest(r.Context(), h.Store.DB(), info.User.ID, channel, box); err != nil {
		if errors.Is(err, notify.ErrInvalidConfig) {
			apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_NOTIFICATION_CHANNEL")
			return
		}
		apperror.WriteDetail(w, r, http.StatusBadRequest, apperror.ValidationError, apperror.ValidationError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "sent"})
}

func (h *Handler) PreviewNotificationTemplate(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	var req notificationPreviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_INVALID_JSON")
		return
	}
	text, err := notify.PreviewTemplate(r.Context(), h.Store.DB(), info.User.ID, req.TriggerType, req.Template)
	if err != nil {
		apperror.WriteDetail(w, r, http.StatusBadRequest, apperror.ValidationError, apperror.ValidationError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"text": text})
}

func (h *Handler) ResetNotificationTemplates(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	var req notificationResetRequest
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}
	if err := notify.ResetTemplates(r.Context(), h.Store.DB(), info.User.ID, req.TriggerType); err != nil {
		apperror.WriteDetail(w, r, http.StatusBadRequest, apperror.ValidationError, apperror.ValidationError, err.Error())
		return
	}
	view, err := notify.GetSettings(r.Context(), h.Store.DB(), info.User.ID)
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}

	var req passwordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_INVALID_JSON")
		return
	}
	if err := auth.ValidatePassword(req.NewPassword, info.User.Login); err != nil {
		msgKey := apperror.PasswordTooWeak
		fallback := "пароль должен содержать минимум одну букву и одну цифру и не совпадать с логином"
		if err == auth.ErrPasswordTooShort {
			msgKey = apperror.PasswordTooShort
			fallback = "новый пароль должен быть не короче 8 символов"
		}
		apperror.WriteL(w, r, http.StatusBadRequest, apperror.ValidationError, msgKey, fallback)
		return
	}
	if req.NewPassword != req.NewPasswordConfirm {
		apperror.WriteL(w, r, http.StatusBadRequest, apperror.ValidationError, apperror.PasswordsMismatch, "пароли не совпадают")
		return
	}

	currentPassword := req.CurrentPassword
	if currentPassword == "" {
		currentPassword = req.OldPassword
	}

	_, hash, err := auth.LoadUserByLogin(r.Context(), h.Store.DB(), info.User.Login)
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}
	okPass, err := auth.VerifyPassword(hash, currentPassword)
	if err != nil || !okPass {
		apperror.WriteL(w, r, http.StatusBadRequest, apperror.ValidationError, apperror.InvalidCurrentPassword, "неверный текущий пароль")
		return
	}

	samePassword, err := auth.VerifyPassword(hash, req.NewPassword)
	if err == nil && samePassword {
		apperror.WriteL(w, r, http.StatusBadRequest, apperror.ValidationError, apperror.PasswordUnchanged, "новый пароль должен отличаться от текущего")
		return
	}

	newHash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}

	_, err = h.Store.DB().ExecContext(r.Context(), `
		UPDATE users SET password_hash = ?, updated_at = datetime('now') WHERE id = ?`,
		newHash, info.User.ID,
	)
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}

	ip := auth.ClientIP(r)
	_ = h.Audit.Log("user.password.change", info.User.ID, info.User.Login, ip, nil)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ListTokens(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}

	rows, err := h.Store.DB().QueryContext(r.Context(), `
		SELECT id, name, token_prefix, expires_at, last_used_at, created_at
		FROM api_tokens WHERE user_id = ? ORDER BY created_at DESC`, info.User.ID,
	)
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}
	defer rows.Close()

	var items []tokenListItem
	for rows.Next() {
		var item tokenListItem
		var expiresAt, lastUsedAt sql.NullString
		if err := rows.Scan(&item.ID, &item.Name, &item.TokenPrefix, &expiresAt, &lastUsedAt, &item.CreatedAt); err != nil {
			apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
			return
		}
		if expiresAt.Valid {
			item.ExpiresAt = &expiresAt.String
		}
		if lastUsedAt.Valid {
			item.LastUsedAt = &lastUsedAt.String
		}
		items = append(items, item)
	}
	if items == nil {
		items = []tokenListItem{}
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *Handler) CreateToken(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}

	var req createTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_INVALID_JSON")
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_TOKEN_NAME_REQUIRED")
		return
	}

	var expiresVal any
	var expiresPtr *string
	if req.ExpiresAt != nil && *req.ExpiresAt != "" {
		if _, err := time.Parse(time.RFC3339, *req.ExpiresAt); err != nil {
			apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_TOKEN_EXPIRES")
			return
		}
		expiresVal = *req.ExpiresAt
		expiresPtr = req.ExpiresAt
	}

	raw, hash, prefix, err := generateAPIToken()
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}

	id := uuid.NewString()
	_, err = h.Store.DB().ExecContext(r.Context(), `
		INSERT INTO api_tokens (id, user_id, name, token_hash, token_prefix, expires_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		id, info.User.ID, name, hash, prefix, expiresVal,
	)
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}

	var createdAt string
	_ = h.Store.DB().QueryRowContext(r.Context(), `SELECT created_at FROM api_tokens WHERE id = ?`, id).Scan(&createdAt)

	ip := auth.ClientIP(r)
	_ = h.Audit.Log("user.token.create", info.User.ID, info.User.Login, ip, map[string]any{
		"token_name":   name,
		"token_prefix": prefix,
	})

	writeJSON(w, http.StatusCreated, createTokenResponse{
		ID:          id,
		Name:        name,
		Token:       raw,
		TokenPrefix: prefix,
		ExpiresAt:   expiresPtr,
		CreatedAt:   createdAt,
	})
}

func (h *Handler) DeleteToken(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}

	id := chi.URLParam(r, "id")
	var name, prefix string
	err := h.Store.DB().QueryRowContext(r.Context(), `
		SELECT name, token_prefix FROM api_tokens WHERE id = ? AND user_id = ?`,
		id, info.User.ID,
	).Scan(&name, &prefix)
	if err != nil {
		apperror.WriteR(w, r, http.StatusNotFound, apperror.NotFound)
		return
	}

	_, err = h.Store.DB().ExecContext(r.Context(), `DELETE FROM api_tokens WHERE id = ? AND user_id = ?`, id, info.User.ID)
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}

	ip := auth.ClientIP(r)
	_ = h.Audit.Log("user.token.revoke", info.User.ID, info.User.Login, ip, map[string]any{
		"token_name":   name,
		"token_prefix": prefix,
	})
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) secretBoxForUpdate(ctx context.Context, req notify.UpdateSettingsInput) (*notify.SecretBox, error) {
	if req.TelegramBotToken == nil && req.MaxToken == nil {
		return nil, nil
	}
	telegramTokenChanged := req.TelegramBotToken != nil && strings.TrimSpace(*req.TelegramBotToken) != ""
	maxTokenChanged := req.MaxToken != nil && strings.TrimSpace(*req.MaxToken) != ""
	if !telegramTokenChanged && !maxTokenChanged {
		return nil, nil
	}
	return h.mustSecretBox(ctx)
}

func (h *Handler) mustSecretBox(ctx context.Context) (*notify.SecretBox, error) {
	secret, err := notify.ResolveSecretKey(ctx, h.Store.DB())
	if err != nil {
		return nil, err
	}
	return notify.NewSecretBox(secret)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
