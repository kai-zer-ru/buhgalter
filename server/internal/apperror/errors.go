package apperror

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/kai-zer-ru/buhgalter/internal/locale"
)

const (
	Unauthorized         = "UNAUTHORIZED"
	Forbidden            = "FORBIDDEN"
	InvalidCredentials   = "INVALID_CREDENTIALS"
	RegistrationDisabled = "REGISTRATION_DISABLED"
	RateLimited          = "RATE_LIMITED"
	ValidationError      = "VALIDATION_ERROR"
	PasswordsMismatch    = "PASSWORDS_MISMATCH"
	InvalidCurrentPassword = "INVALID_CURRENT_PASSWORD"
	PasswordTooShort       = "PASSWORD_TOO_SHORT"
	PasswordTooWeak        = "PASSWORD_TOO_WEAK"
	PasswordUnchanged      = "PASSWORD_UNCHANGED"
	InternalError          = "INTERNAL_ERROR"
	NotFound               = "NOT_FOUND"
	Conflict               = "CONFLICT"
	AlreadyConfigured      = "ALREADY_CONFIGURED"
	ServiceUnavailable     = "SERVICE_UNAVAILABLE"
)

type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type Response struct {
	Error ErrorBody `json:"error"`
}

func Write(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(Response{
		Error: ErrorBody{Code: code, Message: message},
	})
}

func WriteL(w http.ResponseWriter, r *http.Request, status int, code, msgKey, fallback string) {
	Write(w, status, code, locale.T(r, msgKey, fallback))
}

// WriteR localizes message by msgKey (defaults to code).
func WriteR(w http.ResponseWriter, r *http.Request, status int, code string, msgKey ...string) {
	key := code
	if len(msgKey) > 0 && strings.TrimSpace(msgKey[0]) != "" {
		key = msgKey[0]
	}
	WriteL(w, r, status, code, key, key)
}

// WriteDetail localizes msgKey; detail is shown when the key is missing.
func WriteDetail(w http.ResponseWriter, r *http.Request, status int, code, msgKey, detail string) {
	WriteL(w, r, status, code, msgKey, detail)
}
