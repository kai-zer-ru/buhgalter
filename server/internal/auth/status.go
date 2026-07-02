package auth

import (
	"net/http"

	"github.com/kai-zer-ru/buhgalter/internal/apperror"
)

type UserStatus string

const (
	UserStatusActive  UserStatus = "active"
	UserStatusPending UserStatus = "pending"
	UserStatusBanned  UserStatus = "banned"
)

func (s UserStatus) Valid() bool {
	switch s {
	case UserStatusActive, UserStatusPending, UserStatusBanned:
		return true
	default:
		return false
	}
}

func IsActive(status string) bool {
	return status == string(UserStatusActive)
}

func CanTransition(from, to UserStatus) bool {
	if from == to {
		return false
	}
	switch from {
	case UserStatusPending:
		return to == UserStatusActive || to == UserStatusBanned
	case UserStatusActive:
		return to == UserStatusBanned
	case UserStatusBanned:
		return to == UserStatusActive
	default:
		return false
	}
}

func RejectIfNotActive(w http.ResponseWriter, r *http.Request, status string) bool {
	switch status {
	case string(UserStatusPending):
		apperror.WriteR(w, r, http.StatusForbidden, apperror.UserPendingModeration)
		return true
	case string(UserStatusBanned):
		apperror.WriteR(w, r, http.StatusForbidden, apperror.UserBanned)
		return true
	default:
		return false
	}
}
