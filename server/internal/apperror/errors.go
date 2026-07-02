package apperror

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/kai-zer-ru/buhgalter/internal/locale"
)

const (
	Unauthorized           = "UNAUTHORIZED"
	Forbidden              = "FORBIDDEN"
	InvalidCredentials     = "INVALID_CREDENTIALS"
	RegistrationDisabled   = "REGISTRATION_DISABLED"
	RateLimited            = "RATE_LIMITED"
	ValidationError        = "VALIDATION_ERROR"
	PasswordsMismatch      = "PASSWORDS_MISMATCH"
	InvalidCurrentPassword = "INVALID_CURRENT_PASSWORD"
	PasswordTooShort       = "PASSWORD_TOO_SHORT"
	PasswordTooWeak        = "PASSWORD_TOO_WEAK"
	PasswordUnchanged      = "PASSWORD_UNCHANGED"
	InternalError          = "INTERNAL_ERROR"
	NotFound               = "NOT_FOUND"
	Conflict               = "CONFLICT"
	AlreadyConfigured      = "ALREADY_CONFIGURED"
	ServiceUnavailable     = "SERVICE_UNAVAILABLE"
	UserPendingModeration  = "USER_PENDING_MODERATION"
	UserBanned             = "USER_BANNED"
	UserStatusInvalid      = "USER_STATUS_INVALID"
	UserStatusTransition   = "USER_STATUS_TRANSITION"
)

type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Field   string `json:"field,omitempty"`
}

type Response struct {
	Error ErrorBody `json:"error"`
}

func Write(w http.ResponseWriter, status int, code, message string) {
	WriteField(w, status, code, message, "")
}

func WriteField(w http.ResponseWriter, status int, code, message, field string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(Response{
		Error: ErrorBody{Code: code, Message: message, Field: field},
	})
}

func WriteL(w http.ResponseWriter, r *http.Request, status int, code, msgKey, fallback string) {
	WriteField(w, status, code, locale.T(r, msgKey, fallback), inferField(msgKey))
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
	WriteField(w, status, code, locale.T(r, msgKey, detail), inferField(msgKey))
}

func inferField(msgKey string) string {
	switch msgKey {
	case "ERR_INVALID_JSON":
		return "body"
	case "ERR_INVALID_AMOUNT", "ERR_TX_AMOUNT_POSITIVE", "ERR_SETTLE_AMOUNT":
		return "amount"
	case "ERR_INVALID_DUE_DATE":
		return "due_date"
	case "ERR_INVALID_DEBT_DATE":
		return "debt_date"
	case "ERR_PERIOD_DATE":
		return "period"
	case "ERR_GROUP_BY":
		return "group_by"
	case "ERR_ACCOUNT_NOT_FOUND", "ERR_ACCOUNT_ARCHIVED", "ERR_ACCOUNT_ARCHIVED_EDIT", "ERR_ACCOUNT_PRIMARY_ARCHIVED", "ERR_CREDIT_CARD_ARCHIVE_NOT_FULLY_PAID":
		return "account_id"
	case "ERR_ACCOUNT_BANK_REQUIRED", "ERR_ACCOUNT_BANK_FORBIDDEN", "ERR_ACCOUNT_BANK_NOT_FOUND":
		return "bank_id"
	case "ERR_CREDIT_INVALID_TERM":
		return "term_months"
	case "ERR_CREDIT_INVALID_INTERVAL":
		return "payment_interval"
	case "ERR_CREDIT_INVALID_PAYMENT":
		return "monthly_payment"
	case "ERR_CREDIT_INVALID_DEBIT_TIME":
		return "debit_time_local"
	case "ERR_CREDIT_INVALID_KIND":
		return "credit_kind"
	case "ERR_CREDIT_INVALID_MORTGAGE":
		return "property_price"
	case "ERR_CREDIT_COMPLETE_DATE":
		return "payment_date"
	case "ERR_CREDIT_INVALID_STATUS":
		return "status"
	case "CONFLICT_LOGIN_TAKEN", "ERR_LOGIN_LENGTH", "ERR_LOGIN_PASSWORD_REQUIRED":
		return "login"
	case "PASSWORD_TOO_SHORT", "PASSWORD_TOO_WEAK", "PASSWORDS_MISMATCH",
		"INVALID_CURRENT_PASSWORD", "PASSWORD_UNCHANGED":
		return "password"
	default:
		return ""
	}
}
