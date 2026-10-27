package categoryrule

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/kai-zer-ru/buhgalter/internal/auth"
	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
	"github.com/kai-zer-ru/buhgalter/internal/scope"
)

const (
	MaxPerUser    = 100
	MaxPatternLen = 200

	KindCategoryAccount = "category_account"
	KindTextCategory    = "text_category"
)

type Rule struct {
	ID               string  `json:"id"`
	OwnerUserID      string  `json:"-"`
	Scope            string  `json:"scope,omitempty"`
	Kind             string  `json:"kind"`
	Pattern          *string `json:"pattern,omitempty"`
	CategoryID       string  `json:"category_id"`
	SubcategoryID    *string `json:"subcategory_id,omitempty"`
	DefaultAccountID *string `json:"default_account_id,omitempty"`
	Priority         int     `json:"priority"`
	CreatedAt        string  `json:"created_at"`
	UpdatedAt        string  `json:"updated_at"`
}

type Input struct {
	Kind             string
	Pattern          *string
	CategoryID       string
	SubcategoryID    *string
	DefaultAccountID *string
}

var (
	ErrNotFound           = errors.New("category rule not found")
	ErrLimitReached       = errors.New("category rule limit reached")
	ErrInvalidKind        = errors.New("invalid rule kind")
	ErrInvalidPattern     = errors.New("invalid rule pattern")
	ErrInvalidCategory    = errors.New("invalid rule category")
	ErrInvalidSubcategory = errors.New("invalid rule subcategory")
	ErrInvalidAccount     = errors.New("invalid rule account")
	ErrInvalidOrder       = errors.New("invalid rule order")
	ErrSystemCategory     = errors.New("system category not allowed for text rule")
)

func queries(db sqlcdb.DBTX) *sqlcdb.Queries {
	return sqlcdb.New(db)
}

func List(ctx context.Context, db *sql.DB, userID string) ([]Rule, error) {
	return ListByScope(ctx, db, auth.Scope{UserID: userID}, "")
}

func ListByScope(ctx context.Context, db *sql.DB, sc auth.Scope, view string) ([]Rule, error) {
	ids := scope.FilterOwnerIDs(sc, view)
	if len(ids) == 0 {
		return []Rule{}, nil
	}
	rows, err := queries(db).ListCategoryRulesByUser(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make([]Rule, 0, len(rows))
	for _, row := range rows {
		r := fromRow(row)
		r.Scope = scope.AccountScope(sc, r.OwnerUserID)
		out = append(out, r)
	}
	return out, nil
}

func Get(ctx context.Context, db *sql.DB, userID, id string) (Rule, error) {
	return GetByScope(ctx, db, auth.Scope{UserID: userID}, id)
}

func GetByScope(ctx context.Context, db *sql.DB, sc auth.Scope, id string) (Rule, error) {
	ids := sc.OwnerIDs()
	if len(ids) == 0 {
		return Rule{}, ErrNotFound
	}
	row, err := queries(db).GetCategoryRuleByID(ctx, sqlcdb.GetCategoryRuleByIDParams{
		ID: id, OwnerIds: ids,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return Rule{}, ErrNotFound
	}
	if err != nil {
		return Rule{}, err
	}
	r := fromRow(row)
	r.Scope = scope.AccountScope(sc, r.OwnerUserID)
	return r, nil
}

func Create(ctx context.Context, db *sql.DB, userID string, in Input) (Rule, error) {
	count, err := queries(db).CountCategoryRulesByUser(ctx, userID)
	if err != nil {
		return Rule{}, err
	}
	if count >= MaxPerUser {
		return Rule{}, ErrLimitReached
	}
	norm, err := normalizeInput(ctx, db, userID, in)
	if err != nil {
		return Rule{}, err
	}
	maxPri, err := maxPriority(ctx, queries(db), userID)
	if err != nil {
		return Rule{}, err
	}
	id := uuid.NewString()
	now := time.Now().UTC().Format(time.RFC3339)
	if err := queries(db).InsertCategoryRule(ctx, sqlcdb.InsertCategoryRuleParams{
		ID:               id,
		UserID:           userID,
		Kind:             norm.Kind,
		Pattern:          norm.Pattern,
		CategoryID:       norm.CategoryID,
		SubcategoryID:    norm.SubcategoryID,
		DefaultAccountID: norm.DefaultAccountID,
		Priority:         maxPri + 1,
		CreatedAt:        now,
		UpdatedAt:        now,
	}); err != nil {
		return Rule{}, err
	}
	return Get(ctx, db, userID, id)
}

func Update(ctx context.Context, db *sql.DB, userID, id string, in Input) (Rule, error) {
	if _, err := Get(ctx, db, userID, id); err != nil {
		return Rule{}, err
	}
	norm, err := normalizeInput(ctx, db, userID, in)
	if err != nil {
		return Rule{}, err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	n, err := queries(db).UpdateCategoryRule(ctx, sqlcdb.UpdateCategoryRuleParams{
		Kind:             norm.Kind,
		Pattern:          norm.Pattern,
		CategoryID:       norm.CategoryID,
		SubcategoryID:    norm.SubcategoryID,
		DefaultAccountID: norm.DefaultAccountID,
		UpdatedAt:        now,
		ID:               id,
		UserID:           userID,
	})
	if err != nil {
		return Rule{}, err
	}
	if n == 0 {
		return Rule{}, ErrNotFound
	}
	return Get(ctx, db, userID, id)
}

func Delete(ctx context.Context, db *sql.DB, userID, id string) error {
	n, err := queries(db).DeleteCategoryRule(ctx, sqlcdb.DeleteCategoryRuleParams{
		ID: id, UserID: userID,
	})
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func Reorder(ctx context.Context, db *sql.DB, userID string, ids []string) error {
	existing, err := List(ctx, db, userID)
	if err != nil {
		return err
	}
	if len(ids) != len(existing) {
		return ErrInvalidOrder
	}
	seen := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		if id == "" {
			return ErrInvalidOrder
		}
		if _, ok := seen[id]; ok {
			return ErrInvalidOrder
		}
		seen[id] = struct{}{}
	}
	for _, r := range existing {
		if _, ok := seen[r.ID]; !ok {
			return ErrInvalidOrder
		}
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	q := queries(tx)
	now := time.Now().UTC().Format(time.RFC3339)
	for i, id := range ids {
		if err := q.UpdateCategoryRulePriority(ctx, sqlcdb.UpdateCategoryRulePriorityParams{
			Priority:  int64(i + 1),
			UpdatedAt: now,
			ID:        id,
			UserID:    userID,
		}); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func normalizeInput(ctx context.Context, db *sql.DB, userID string, in Input) (Input, error) {
	kind := strings.TrimSpace(in.Kind)
	if kind != KindCategoryAccount && kind != KindTextCategory {
		return Input{}, ErrInvalidKind
	}
	catID := strings.TrimSpace(in.CategoryID)
	if catID == "" {
		return Input{}, ErrInvalidCategory
	}
	catRow, err := queries(db).GetCategoryByID(ctx, sqlcdb.GetCategoryByIDParams{ID: catID, OwnerIds: []string{userID}})
	if errors.Is(err, sql.ErrNoRows) {
		return Input{}, ErrInvalidCategory
	}
	if err != nil {
		return Input{}, err
	}
	if kind == KindTextCategory && catRow.IsSystem != 0 {
		return Input{}, ErrSystemCategory
	}

	sub := trimPtr(in.SubcategoryID)
	if sub != nil {
		subRow, err := queries(db).GetSubcategoryByID(ctx, sqlcdb.GetSubcategoryByIDParams{ID: *sub, OwnerIds: []string{userID}})
		if errors.Is(err, sql.ErrNoRows) {
			return Input{}, ErrInvalidSubcategory
		}
		if err != nil {
			return Input{}, err
		}
		if subRow.CategoryID != catID {
			return Input{}, ErrInvalidSubcategory
		}
	}

	out := Input{
		Kind:          kind,
		CategoryID:    catID,
		SubcategoryID: sub,
	}

	switch kind {
	case KindCategoryAccount:
		if in.Pattern != nil && strings.TrimSpace(*in.Pattern) != "" {
			return Input{}, ErrInvalidPattern
		}
		acc := trimPtr(in.DefaultAccountID)
		if acc == nil {
			return Input{}, ErrInvalidAccount
		}
		if err := requireActiveAccount(ctx, db, userID, *acc); err != nil {
			return Input{}, err
		}
		out.DefaultAccountID = acc
	case KindTextCategory:
		if in.DefaultAccountID != nil && trimPtr(in.DefaultAccountID) != nil {
			return Input{}, ErrInvalidAccount
		}
		pat := normalizePattern(in.Pattern)
		if pat == nil {
			return Input{}, ErrInvalidPattern
		}
		out.Pattern = pat
	}
	return out, nil
}

func requireActiveAccount(ctx context.Context, db *sql.DB, userID, accountID string) error {
	row, err := queries(db).GetAccountByID(ctx, sqlcdb.GetAccountByIDParams{ID: accountID, OwnerIds: []string{userID}})
	if errors.Is(err, sql.ErrNoRows) {
		return ErrInvalidAccount
	}
	if err != nil {
		return err
	}
	if row.Status != "active" {
		return ErrInvalidAccount
	}
	return nil
}

func normalizePattern(p *string) *string {
	if p == nil {
		return nil
	}
	s := strings.TrimSpace(*p)
	s = strings.Join(strings.Fields(s), " ")
	if s == "" || utf8.RuneCountInString(s) > MaxPatternLen {
		return nil
	}
	return &s
}

func fromRow(row sqlcdb.CategoryRule) Rule {
	r := Rule{
		ID:               row.ID,
		OwnerUserID:      row.UserID,
		Kind:             row.Kind,
		CategoryID:       row.CategoryID,
		SubcategoryID:    row.SubcategoryID,
		DefaultAccountID: row.DefaultAccountID,
		Priority:         int(row.Priority),
		CreatedAt:        row.CreatedAt,
		UpdatedAt:        row.UpdatedAt,
	}
	if row.Pattern != nil && *row.Pattern != "" {
		r.Pattern = row.Pattern
	}
	return r
}

func maxPriority(ctx context.Context, q *sqlcdb.Queries, userID string) (int64, error) {
	v, err := q.MaxCategoryRulePriority(ctx, userID)
	if err != nil {
		return 0, err
	}
	switch n := v.(type) {
	case int64:
		return n, nil
	case int:
		return int64(n), nil
	case int32:
		return int64(n), nil
	case float64:
		return int64(n), nil
	default:
		return 0, nil
	}
}

func trimPtr(p *string) *string {
	if p == nil {
		return nil
	}
	s := strings.TrimSpace(*p)
	if s == "" {
		return nil
	}
	return &s
}
