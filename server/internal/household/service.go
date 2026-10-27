package household

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/kai-zer-ru/buhgalter/internal/auth"
	"github.com/kai-zer-ru/buhgalter/internal/categoryseed"
	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
)

const (
	maxNameLen     = 64
	inviteTTL      = 7 * 24 * time.Hour
	inviteAttempts = 8
)

func nowUTC() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func NormalizeKind(raw string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", KindFamily:
		return KindFamily, nil
	case KindTeam:
		return KindTeam, nil
	default:
		return "", ErrInvalidKind
	}
}

func defaultHouseholdName(displayName, lang, kind string) string {
	name := strings.TrimSpace(displayName)
	if name == "" {
		name = "user"
	}
	var full string
	if lang == "en" {
		if kind == KindTeam {
			full = name + "'s team"
		} else {
			full = name + "'s family"
		}
	} else if kind == KindTeam {
		full = "Команда " + name
	} else {
		full = "Семья " + name
	}
	return clipName(full)
}

func clipName(name string) string {
	if utf8.RuneCountInString(name) <= maxNameLen {
		return name
	}
	runes := []rune(name)
	return string(runes[:maxNameLen])
}

func normalizeName(name string, required bool) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		if required {
			return "", ErrNameRequired
		}
		return "", nil
	}
	if utf8.RuneCountInString(name) > maxNameLen {
		return "", ErrNameTooLong
	}
	return name, nil
}

func GetMe(ctx context.Context, db *sql.DB, userID string) (MeResponse, error) {
	q := sqlcdb.New(db)
	row, err := q.GetHouseholdByUserID(ctx, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return MeResponse{Household: nil}, nil
	}
	if err != nil {
		return MeResponse{}, err
	}
	return buildMe(ctx, q, row.ID, row.Name, row.Kind, row.CreatedAt, row.Role)
}

func buildMe(ctx context.Context, q *sqlcdb.Queries, householdID, name, kind, createdAt, role string) (MeResponse, error) {
	rows, err := q.ListHouseholdMembers(ctx, householdID)
	if err != nil {
		return MeResponse{}, err
	}
	members := make([]Member, 0, len(rows))
	for _, m := range rows {
		members = append(members, Member{
			UserID:      m.UserID,
			DisplayName: m.DisplayName,
			Login:       m.Login,
			Role:        m.Role,
			JoinedAt:    m.JoinedAt,
		})
	}
	return MeResponse{
		Household: &Household{ID: householdID, Name: name, Kind: kind, CreatedAt: createdAt},
		Role:      role,
		Members:   members,
	}, nil
}

func Create(ctx context.Context, db *sql.DB, owner auth.User, name, kind string) (MeResponse, error) {
	name, err := normalizeName(name, false)
	if err != nil {
		return MeResponse{}, err
	}
	kind, err = NormalizeKind(kind)
	if err != nil {
		return MeResponse{}, err
	}
	q := sqlcdb.New(db)
	if _, err := q.GetHouseholdByUserID(ctx, owner.ID); err == nil {
		return MeResponse{}, ErrAlreadyMember
	} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return MeResponse{}, err
	}
	if name == "" {
		name = defaultHouseholdName(owner.DisplayName, owner.Language, kind)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return MeResponse{}, err
	}
	defer func() { _ = tx.Rollback() }()
	tq := sqlcdb.New(tx)

	householdID := uuid.NewString()
	principalID := uuid.NewString()
	if err := tq.InsertHouseholdPrincipal(ctx, sqlcdb.InsertHouseholdPrincipalParams{
		ID:          principalID,
		Login:       auth.HouseholdLoginPrefix + householdID,
		DisplayName: strPtr(name),
		Language:    owner.Language,
		Currency:    owner.Currency,
		Timezone:    owner.Timezone,
	}); err != nil {
		return MeResponse{}, err
	}
	if err := tq.InsertHousehold(ctx, sqlcdb.InsertHouseholdParams{
		ID:              householdID,
		Name:            name,
		Kind:            kind,
		OwnerUserID:     owner.ID,
		PrincipalUserID: principalID,
	}); err != nil {
		return MeResponse{}, err
	}
	if err := tq.InsertHouseholdMember(ctx, sqlcdb.InsertHouseholdMemberParams{
		HouseholdID: householdID,
		UserID:      owner.ID,
		Role:        "owner",
	}); err != nil {
		return MeResponse{}, err
	}
	if err := copyOwnerCatalog(ctx, tq, owner.ID, principalID); err != nil {
		return MeResponse{}, err
	}
	if err := categoryseed.EnsureSystemCategories(ctx, tx, principalID); err != nil {
		return MeResponse{}, err
	}
	if err := tx.Commit(); err != nil {
		return MeResponse{}, err
	}
	return GetMe(ctx, db, owner.ID)
}

func Rename(ctx context.Context, db *sql.DB, scope auth.Scope, name string) (MeResponse, error) {
	if !scope.InHousehold() {
		return MeResponse{}, ErrNotMember
	}
	if !scope.IsOwner() {
		return MeResponse{}, ErrOwnerOnly
	}
	name, err := normalizeName(name, true)
	if err != nil {
		return MeResponse{}, err
	}
	q := sqlcdb.New(db)
	if err := q.UpdateHouseholdName(ctx, sqlcdb.UpdateHouseholdNameParams{Name: name, ID: scope.HouseholdID}); err != nil {
		return MeResponse{}, err
	}
	return GetMe(ctx, db, scope.UserID)
}

func ListInvites(ctx context.Context, db *sql.DB, scope auth.Scope, baseURL string) ([]Invite, error) {
	if !scope.InHousehold() {
		return nil, ErrNotMember
	}
	if !scope.IsOwner() {
		return nil, ErrOwnerOnly
	}
	rows, err := sqlcdb.New(db).ListPendingHouseholdInvites(ctx, sqlcdb.ListPendingHouseholdInvitesParams{
		HouseholdID: scope.HouseholdID,
		ExpiresAt:   nowUTC(),
	})
	if err != nil {
		return nil, err
	}
	out := make([]Invite, 0, len(rows))
	for _, row := range rows {
		out = append(out, Invite{
			ID:        row.ID,
			Code:      formatInviteCode(row.Code),
			URL:       inviteURL(baseURL, row.Code),
			CreatedAt: row.CreatedAt,
			ExpiresAt: row.ExpiresAt,
		})
	}
	return out, nil
}

func CreateInvite(ctx context.Context, db *sql.DB, scope auth.Scope, baseURL string) (Invite, error) {
	if !scope.InHousehold() {
		return Invite{}, ErrNotMember
	}
	if !scope.IsOwner() {
		return Invite{}, ErrOwnerOnly
	}
	q := sqlcdb.New(db)
	expires := time.Now().UTC().Add(inviteTTL).Format(time.RFC3339)
	var last error
	for i := 0; i < inviteAttempts; i++ {
		code, err := generateInviteCode()
		if err != nil {
			return Invite{}, err
		}
		id := uuid.NewString()
		err = q.InsertHouseholdInvite(ctx, sqlcdb.InsertHouseholdInviteParams{
			ID:          id,
			HouseholdID: scope.HouseholdID,
			Code:        code,
			CreatedBy:   scope.UserID,
			ExpiresAt:   expires,
		})
		if err == nil {
			return Invite{
				ID:        id,
				Code:      formatInviteCode(code),
				URL:       inviteURL(baseURL, code),
				ExpiresAt: expires,
			}, nil
		}
		last = err
		if !strings.Contains(strings.ToLower(err.Error()), "unique") {
			return Invite{}, err
		}
	}
	return Invite{}, last
}

func RevokeInvite(ctx context.Context, db *sql.DB, scope auth.Scope, inviteID string) error {
	if !scope.InHousehold() {
		return ErrNotMember
	}
	if !scope.IsOwner() {
		return ErrOwnerOnly
	}
	n, err := sqlcdb.New(db).RevokeHouseholdInvite(ctx, sqlcdb.RevokeHouseholdInviteParams{
		ID:          inviteID,
		HouseholdID: scope.HouseholdID,
	})
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrInviteInvalid
	}
	return nil
}

func PreviewInvite(ctx context.Context, db *sql.DB, userID, code string) (InvitePreview, error) {
	if _, err := sqlcdb.New(db).GetHouseholdByUserID(ctx, userID); err == nil {
		return InvitePreview{}, ErrAlreadyMember
	} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return InvitePreview{}, err
	}
	raw, ok := normalizeInviteCode(code)
	if !ok {
		return InvitePreview{}, ErrInviteInvalid
	}
	q := sqlcdb.New(db)
	row, err := q.GetHouseholdInviteByCode(ctx, raw)
	if err != nil {
		return InvitePreview{}, ErrInviteInvalid
	}
	if row.Status != "pending" || row.ExpiresAt <= nowUTC() {
		return InvitePreview{}, ErrInviteInvalid
	}
	owner, err := q.GetUserByID(ctx, row.OwnerUserID)
	if err != nil {
		return InvitePreview{}, ErrInviteInvalid
	}
	return InvitePreview{
		HouseholdName:    row.HouseholdName,
		OwnerDisplayName: owner.DisplayName,
		ExpiresAt:        row.ExpiresAt,
	}, nil
}

func AcceptInvite(ctx context.Context, db *sql.DB, userID, code string) (MeResponse, error) {
	if _, err := sqlcdb.New(db).GetHouseholdByUserID(ctx, userID); err == nil {
		return MeResponse{}, ErrAlreadyMember
	} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return MeResponse{}, err
	}
	raw, ok := normalizeInviteCode(code)
	if !ok {
		return MeResponse{}, ErrInviteInvalid
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return MeResponse{}, err
	}
	defer func() { _ = tx.Rollback() }()
	tq := sqlcdb.New(tx)

	row, err := tq.GetHouseholdInviteByCode(ctx, raw)
	if err != nil {
		return MeResponse{}, ErrInviteInvalid
	}
	n, err := tq.ConsumeHouseholdInvite(ctx, sqlcdb.ConsumeHouseholdInviteParams{
		UsedBy:    strPtr(userID),
		ID:        row.ID,
		ExpiresAt: nowUTC(),
	})
	if err != nil {
		return MeResponse{}, err
	}
	if n != 1 {
		return MeResponse{}, ErrInviteInvalid
	}
	if err := tq.InsertHouseholdMember(ctx, sqlcdb.InsertHouseholdMemberParams{
		HouseholdID: row.HouseholdID,
		UserID:      userID,
		Role:        "member",
	}); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return MeResponse{}, ErrAlreadyMember
		}
		return MeResponse{}, err
	}
	if err := tx.Commit(); err != nil {
		return MeResponse{}, err
	}
	return GetMe(ctx, db, userID)
}

func Leave(ctx context.Context, db *sql.DB, scope auth.Scope) error {
	if !scope.InHousehold() {
		return ErrNotMember
	}
	if scope.IsOwner() {
		return ErrOwnerCannotLeave
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	q := sqlcdb.New(tx)
	if err := clearLeavingRefs(ctx, q, scope.UserID, scope.PrincipalID); err != nil {
		return err
	}
	n, err := q.DeleteHouseholdMember(ctx, sqlcdb.DeleteHouseholdMemberParams{
		HouseholdID: scope.HouseholdID,
		UserID:      scope.UserID,
	})
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotMember
	}
	return tx.Commit()
}

func Kick(ctx context.Context, db *sql.DB, scope auth.Scope, targetID string) error {
	if !scope.InHousehold() {
		return ErrNotMember
	}
	if !scope.IsOwner() {
		return ErrOwnerOnly
	}
	if targetID == scope.UserID {
		return ErrCannotKickSelf
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	q := sqlcdb.New(tx)
	member, err := q.GetHouseholdMember(ctx, sqlcdb.GetHouseholdMemberParams{
		HouseholdID: scope.HouseholdID,
		UserID:      targetID,
	})
	if err != nil {
		return ErrMemberNotFound
	}
	if member.Role == "owner" {
		return ErrMemberNotFound
	}
	if err := clearLeavingRefs(ctx, q, targetID, scope.PrincipalID); err != nil {
		return err
	}
	n, err := q.DeleteHouseholdMember(ctx, sqlcdb.DeleteHouseholdMemberParams{
		HouseholdID: scope.HouseholdID,
		UserID:      targetID,
	})
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrMemberNotFound
	}
	return tx.Commit()
}

func Transfer(ctx context.Context, db *sql.DB, scope auth.Scope, targetID string) (MeResponse, error) {
	if !scope.InHousehold() {
		return MeResponse{}, ErrNotMember
	}
	if !scope.IsOwner() {
		return MeResponse{}, ErrOwnerOnly
	}
	if targetID == "" || targetID == scope.UserID {
		return MeResponse{}, ErrMemberNotFound
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return MeResponse{}, err
	}
	defer func() { _ = tx.Rollback() }()
	tq := sqlcdb.New(tx)

	member, err := tq.GetHouseholdMember(ctx, sqlcdb.GetHouseholdMemberParams{
		HouseholdID: scope.HouseholdID,
		UserID:      targetID,
	})
	if err != nil || member.Role != "member" {
		return MeResponse{}, ErrMemberNotFound
	}
	newOwner, err := tq.GetUserByID(ctx, targetID)
	if err != nil {
		return MeResponse{}, ErrMemberNotFound
	}
	if err := tq.UpdateHouseholdMemberRole(ctx, sqlcdb.UpdateHouseholdMemberRoleParams{
		Role:        "member",
		HouseholdID: scope.HouseholdID,
		UserID:      scope.UserID,
	}); err != nil {
		return MeResponse{}, err
	}
	if err := tq.UpdateHouseholdMemberRole(ctx, sqlcdb.UpdateHouseholdMemberRoleParams{
		Role:        "owner",
		HouseholdID: scope.HouseholdID,
		UserID:      targetID,
	}); err != nil {
		return MeResponse{}, err
	}
	if err := tq.UpdateHouseholdOwner(ctx, sqlcdb.UpdateHouseholdOwnerParams{
		OwnerUserID: targetID,
		ID:          scope.HouseholdID,
	}); err != nil {
		return MeResponse{}, err
	}
	if err := tq.UpdatePrincipalLocale(ctx, sqlcdb.UpdatePrincipalLocaleParams{
		Language: newOwner.Language,
		Timezone: newOwner.Timezone,
		Currency: newOwner.Currency,
		ID:       scope.PrincipalID,
	}); err != nil {
		return MeResponse{}, err
	}
	if err := tx.Commit(); err != nil {
		return MeResponse{}, err
	}
	return GetMe(ctx, db, scope.UserID)
}

func Dissolve(ctx context.Context, db *sql.DB, scope auth.Scope, confirmMembers bool) error {
	_, err := DissolveWithData(ctx, db, scope, confirmMembers)
	return err
}

func MemberIDs(ctx context.Context, db *sql.DB, householdID string) ([]string, error) {
	if householdID == "" {
		return nil, nil
	}
	return sqlcdb.New(db).ListHouseholdMemberIDs(ctx, householdID)
}

func SyncPrincipalLocale(ctx context.Context, db sqlcdb.DBTX, principalID, language, timezone, currency string) error {
	return sqlcdb.New(db).UpdatePrincipalLocale(ctx, sqlcdb.UpdatePrincipalLocaleParams{
		Language: language,
		Timezone: timezone,
		Currency: currency,
		ID:       principalID,
	})
}

func inviteURL(base, rawCode string) string {
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	if base == "" {
		return "/invite/" + formatInviteCode(rawCode)
	}
	return base + "/invite/" + formatInviteCode(rawCode)
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func InviteHouseholdID(ctx context.Context, db *sql.DB, code string) (string, error) {
	raw, ok := normalizeInviteCode(code)
	if !ok {
		return "", ErrInviteInvalid
	}
	row, err := sqlcdb.New(db).GetHouseholdInviteByCode(ctx, raw)
	if err != nil {
		return "", ErrInviteInvalid
	}
	return row.HouseholdID, nil
}
