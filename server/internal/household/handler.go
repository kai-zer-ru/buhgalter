package household

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/kai-zer-ru/buhgalter/internal/apicache"
	"github.com/kai-zer-ru/buhgalter/internal/apperror"
	"github.com/kai-zer-ru/buhgalter/internal/audit"
	"github.com/kai-zer-ru/buhgalter/internal/auth"
	"github.com/kai-zer-ru/buhgalter/internal/db"
	appmw "github.com/kai-zer-ru/buhgalter/internal/middleware"
	"github.com/kai-zer-ru/buhgalter/internal/realtime"
	"github.com/kai-zer-ru/buhgalter/internal/settingscache"
)

type Handler struct {
	Store   *db.Handle
	Audit   *audit.Logger
	Cache   *apicache.Cache
	Limiter *appmw.IPRateLimiter
}

func (h *Handler) GetMe(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	resp, err := GetMe(r.Context(), h.Store.DB(), info.User.ID)
	if writeErr(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	var req struct {
		Name string `json:"name"`
		Kind string `json:"kind"`
	}
	if r.ContentLength != 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_INVALID_JSON")
			return
		}
	}
	resp, err := Create(r.Context(), h.Store.DB(), info.User, req.Name, req.Kind)
	if writeErr(w, r, err) {
		return
	}
	h.audit(r, info, "household.create", map[string]any{"household_id": resp.Household.ID})
	h.invalidate(r.Context(), []string{info.User.ID})
	writeJSON(w, http.StatusCreated, resp)
}

func (h *Handler) Rename(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_INVALID_JSON")
		return
	}
	resp, err := Rename(r.Context(), h.Store.DB(), info.Scope, req.Name)
	if writeErr(w, r, err) {
		return
	}
	ids, _ := MemberIDs(r.Context(), h.Store.DB(), info.Scope.HouseholdID)
	h.audit(r, info, "household.rename", map[string]any{"household_id": info.Scope.HouseholdID, "name": req.Name})
	h.invalidate(r.Context(), ids)
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) ListInvites(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	invites, err := ListInvites(r.Context(), h.Store.DB(), info.Scope, h.baseURL(r))
	if writeErr(w, r, err) {
		return
	}
	if invites == nil {
		invites = []Invite{}
	}
	writeJSON(w, http.StatusOK, invites)
}

func (h *Handler) CreateInvite(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	invite, err := CreateInvite(r.Context(), h.Store.DB(), info.Scope, h.baseURL(r))
	if writeErr(w, r, err) {
		return
	}
	h.audit(r, info, "household.invite.create", map[string]any{"invite_id": invite.ID})
	writeJSON(w, http.StatusCreated, invite)
}

func (h *Handler) RevokeInvite(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	id := chi.URLParam(r, "id")
	if err := RevokeInvite(r.Context(), h.Store.DB(), info.Scope, id); writeErr(w, r, err) {
		return
	}
	h.audit(r, info, "household.invite.revoke", map[string]any{"invite_id": id})
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Preview(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	if !h.allowInviteAttempt(w, r, info.User.ID) {
		return
	}
	var req struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_INVALID_JSON")
		return
	}
	preview, err := PreviewInvite(r.Context(), h.Store.DB(), info.User.ID, req.Code)
	if writeErr(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, preview)
}

func (h *Handler) Accept(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	if !h.allowInviteAttempt(w, r, info.User.ID) {
		return
	}
	var req struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_INVALID_JSON")
		return
	}
	hhID, _ := InviteHouseholdID(r.Context(), h.Store.DB(), req.Code)
	before, _ := MemberIDs(r.Context(), h.Store.DB(), hhID)
	resp, err := AcceptInvite(r.Context(), h.Store.DB(), info.User.ID, req.Code)
	if writeErr(w, r, err) {
		return
	}
	ids := append(before, info.User.ID)
	h.audit(r, info, "household.invite.accept", map[string]any{"household_id": resp.Household.ID})
	h.invalidate(r.Context(), ids)
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) Leave(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	ids, _ := MemberIDs(r.Context(), h.Store.DB(), info.Scope.HouseholdID)
	if err := Leave(r.Context(), h.Store.DB(), info.Scope); writeErr(w, r, err) {
		return
	}
	h.audit(r, info, "household.leave", map[string]any{"household_id": info.Scope.HouseholdID})
	h.invalidate(r.Context(), ids)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Kick(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	target := chi.URLParam(r, "user_id")
	ids, _ := MemberIDs(r.Context(), h.Store.DB(), info.Scope.HouseholdID)
	if err := Kick(r.Context(), h.Store.DB(), info.Scope, target); writeErr(w, r, err) {
		return
	}
	h.audit(r, info, "household.kick", map[string]any{"household_id": info.Scope.HouseholdID, "user_id": target})
	h.invalidate(r.Context(), ids)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Transfer(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	var req struct {
		UserID string `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_INVALID_JSON")
		return
	}
	resp, err := Transfer(r.Context(), h.Store.DB(), info.Scope, strings.TrimSpace(req.UserID))
	if writeErr(w, r, err) {
		return
	}
	ids, _ := MemberIDs(r.Context(), h.Store.DB(), info.Scope.HouseholdID)
	h.audit(r, info, "household.transfer", map[string]any{"household_id": info.Scope.HouseholdID, "user_id": req.UserID})
	h.invalidate(r.Context(), ids)
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) Dissolve(w http.ResponseWriter, r *http.Request) {
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}
	var req struct {
		ConfirmMembers bool `json:"confirm_members"`
	}
	if r.ContentLength != 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_INVALID_JSON")
			return
		}
	}
	ids, _ := MemberIDs(r.Context(), h.Store.DB(), info.Scope.HouseholdID)
	res, err := DissolveWithData(r.Context(), h.Store.DB(), info.Scope, req.ConfirmMembers)
	if writeErr(w, r, err) {
		return
	}
	h.audit(r, info, "household.dissolve", map[string]any{
		"household_id": info.Scope.HouseholdID,
		"account_ids":  res.AccountIDs,
		"moved":        res.Moved,
	})
	h.invalidate(r.Context(), ids)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) allowInviteAttempt(w http.ResponseWriter, r *http.Request, userID string) bool {
	if h.Limiter == nil || h.Limiter.Allow(userID) {
		return true
	}
	apperror.WriteR(w, r, http.StatusTooManyRequests, apperror.RateLimited)
	return false
}

func (h *Handler) baseURL(r *http.Request) string {
	if ext, err := settingscache.ExternalURL(r.Context(), h.Store.DB()); err == nil && ext.Valid {
		if u := strings.TrimRight(strings.TrimSpace(ext.String), "/"); u != "" {
			return u
		}
	}
	scheme := "http"
	if r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}

func (h *Handler) audit(r *http.Request, info auth.AuthInfo, action string, details map[string]any) {
	if h.Audit == nil {
		return
	}
	_ = h.Audit.Log(action, info.User.ID, info.User.Login, auth.ClientIP(r), details)
}

func (h *Handler) invalidate(_ context.Context, userIDs []string) {
	seen := make(map[string]struct{}, len(userIDs))
	for _, id := range userIDs {
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		if h.Cache != nil {
			h.Cache.InvalidateUser(id)
		} else {
			realtime.NotifyUserDataChanged(id)
		}
	}
}

func writeErr(w http.ResponseWriter, r *http.Request, err error) bool {
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, ErrAlreadyMember):
		apperror.WriteR(w, r, http.StatusConflict, "ERR_HOUSEHOLD_ALREADY_MEMBER")
	case errors.Is(err, ErrNotMember):
		apperror.WriteR(w, r, http.StatusNotFound, "ERR_HOUSEHOLD_NOT_MEMBER")
	case errors.Is(err, ErrOwnerOnly):
		apperror.WriteR(w, r, http.StatusForbidden, "ERR_HOUSEHOLD_OWNER_ONLY")
	case errors.Is(err, ErrOwnerCannotLeave):
		apperror.WriteR(w, r, http.StatusConflict, "ERR_HOUSEHOLD_OWNER_CANNOT_LEAVE")
	case errors.Is(err, ErrHasMembers):
		apperror.WriteR(w, r, http.StatusConflict, "ERR_HOUSEHOLD_HAS_MEMBERS")
	case errors.Is(err, ErrHasSharedData):
		apperror.WriteR(w, r, http.StatusConflict, "ERR_HOUSEHOLD_HAS_SHARED_DATA")
	case errors.Is(err, ErrNoOriginalOwner):
		apperror.WriteR(w, r, http.StatusConflict, "ERR_UNSHARE_NO_ORIGINAL_OWNER")
	case errors.Is(err, ErrOwnerDeleteBlocked):
		apperror.WriteR(w, r, http.StatusConflict, "ERR_HOUSEHOLD_OWNER_DELETE")
	case errors.Is(err, ErrInviteInvalid):
		apperror.WriteR(w, r, http.StatusBadRequest, "ERR_INVITE_INVALID")
	case errors.Is(err, ErrMemberNotFound):
		apperror.WriteR(w, r, http.StatusNotFound, "ERR_HOUSEHOLD_MEMBER_NOT_FOUND")
	case errors.Is(err, ErrCannotKickSelf):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_HOUSEHOLD_CANNOT_KICK_SELF")
	case errors.Is(err, ErrNameRequired), errors.Is(err, ErrNameTooLong):
		apperror.WriteR(w, r, http.StatusBadRequest, apperror.ValidationError, "ERR_HOUSEHOLD_NAME")
	case errors.Is(err, ErrInvalidKind):
		apperror.WriteR(w, r, http.StatusBadRequest, "ERR_HOUSEHOLD_KIND")
	default:
		apperror.WriteR(w, r, http.StatusInternalServerError, apperror.InternalError)
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
