package auth

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/kai-zer-ru/buhgalter/internal/apperror"
	"github.com/kai-zer-ru/buhgalter/internal/audit"
	"github.com/kai-zer-ru/buhgalter/internal/db"
	appmw "github.com/kai-zer-ru/buhgalter/internal/middleware"
)

type Handler struct {
	Store        *db.Handle
	Audit        *audit.Logger
	Logger       *slog.Logger
	LoginLimiter *appmw.IPRateLimiter
}

type loginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

type registerRequest struct {
	Login           string `json:"login"`
	Password        string `json:"password"`
	PasswordConfirm string `json:"password_confirm"`
	DisplayName     string `json:"display_name"`
}

type verifyResponse struct {
	Valid bool `json:"valid"`
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	ip := ClientIP(r)
	if !h.LoginLimiter.Allow(ip) {
		w.Header().Set("Retry-After", "60")
		apperror.WriteR(w, r, http.StatusTooManyRequests, apperror.RateLimited)
		return
	}

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_INVALID_JSON")
		return
	}

	login := strings.TrimSpace(req.Login)
	if login == "" || req.Password == "" {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_LOGIN_PASSWORD_REQUIRED")
		return
	}

	user, hash, err := LoadUserByLogin(r.Context(), h.Store.DB(), login)
	if err != nil {
		h.logFailedLogin(ip, login)
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.InvalidCredentials)
		return
	}

	ok, err := VerifyPassword(hash, req.Password)
	if err != nil || !ok {
		h.logFailedLogin(ip, login)
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.InvalidCredentials)
		return
	}

	token, err := CreateSession(r.Context(), h.Store.DB(), user.ID, ip, r.UserAgent())
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}

	SetSessionCookie(w, r, token)
	_ = h.Audit.Log("auth.login.success", user.ID, user.Login, ip, nil)

	writeJSON(w, http.StatusOK, loginResponse{Token: token, User: *user})
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	info, ok := FromContext(r.Context())
	ip := ClientIP(r)
	if ok && info.Token != "" {
		_ = DeleteSessionByToken(r.Context(), h.Store.DB(), info.Token)
		_ = h.Audit.Log("auth.logout", info.User.ID, info.User.Login, ip, nil)
	}
	ClearSessionCookie(w, r)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var enabled int
	if err := h.Store.DB().QueryRowContext(r.Context(), `
		SELECT registration_enabled FROM system_settings WHERE id = 1`).Scan(&enabled); err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}
	if enabled != 1 {
		apperror.WriteR(w, r, http.StatusForbidden, apperror.RegistrationDisabled)
		return
	}

	var req registerRequest
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
	if err := ValidatePassword(req.Password, login); err != nil {
		msgKey := apperror.PasswordTooWeak
		fallback := "пароль должен содержать минимум одну букву и одну цифру и не совпадать с логином"
		if err == ErrPasswordTooShort {
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

	hash, err := HashPassword(req.Password)
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}

	userID, err := CreateUser(r.Context(), h.Store.DB(), login, hash, displayName, false)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			apperror.WriteR(w, r, http.StatusConflict, apperror.ValidationError, "CONFLICT_LOGIN_TAKEN")
			return
		}
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}

	user, err := LoadUser(r.Context(), h.Store.DB(), userID)
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}

	ip := ClientIP(r)
	token, err := CreateSession(r.Context(), h.Store.DB(), user.ID, ip, r.UserAgent())
	if err != nil {
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
		return
	}

	SetSessionCookie(w, r, token)
	writeJSON(w, http.StatusCreated, loginResponse{Token: token, User: *user})
}

func (h *Handler) Verify(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimSpace(r.URL.Query().Get("token"))
	valid := false
	if token != "" {
		valid = VerifyToken(r.Context(), h.Store.DB(), token)
	}
	writeJSON(w, http.StatusOK, verifyResponse{Valid: valid})
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	info, ok := FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	writeJSON(w, http.StatusOK, info.User)
}

func (h *Handler) logFailedLogin(ip, login string) {
	h.Logger.Warn("login failed", "ip", ip, "login", login)
	_ = h.Audit.Log("auth.login.failed", "", login, ip, nil)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
