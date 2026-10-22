package transactiontemplate

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
	"github.com/kai-zer-ru/buhgalter/internal/tag"
)

const (
	MaxPerUser = 100
	MaxNameLen = 80
	MaxDescLen = 500
)

type Template struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Type          string    `json:"type"`
	AccountID     *string   `json:"account_id,omitempty"`
	ToAccountID   *string   `json:"to_account_id,omitempty"`
	CategoryID    *string   `json:"category_id,omitempty"`
	SubcategoryID *string   `json:"subcategory_id,omitempty"`
	Amount        *int64    `json:"amount,omitempty"`
	Description   *string   `json:"description,omitempty"`
	MerchantID    *string   `json:"merchant_id,omitempty"`
	Icon          *string   `json:"icon,omitempty"`
	SortOrder     int       `json:"sort_order"`
	Tags          []tag.Ref `json:"tags"`
	CreatedAt     string    `json:"created_at"`
	UpdatedAt     string    `json:"updated_at"`
}

type Input struct {
	Name          string
	Type          string
	AccountID     *string
	ToAccountID   *string
	CategoryID    *string
	SubcategoryID *string
	Amount        *int64
	Description   *string
	MerchantID    *string
	Icon          *string
	TagIDs        []string
}

var (
	ErrNotFound        = errors.New("transaction template not found")
	ErrInvalidName     = errors.New("invalid template name")
	ErrInvalidType     = errors.New("invalid template type")
	ErrInvalidAmount   = errors.New("invalid template amount")
	ErrInvalidAccount  = errors.New("invalid template account")
	ErrInvalidCategory = errors.New("invalid template category")
	ErrInvalidMerchant = errors.New("invalid template merchant")
	ErrInvalidTag      = errors.New("invalid template tag")
	ErrLimitReached    = errors.New("template limit reached")
	ErrInvalidReorder  = errors.New("invalid template reorder")
)

func queries(db sqlcdb.DBTX) *sqlcdb.Queries {
	return sqlcdb.New(db)
}

func List(ctx context.Context, db *sql.DB, userID string) ([]Template, error) {
	rows, err := queries(db).ListTransactionTemplatesByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]Template, 0, len(rows))
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		out = append(out, fromRow(row))
		ids = append(ids, row.ID)
	}
	if err := attachTags(ctx, db, out, ids); err != nil {
		return nil, err
	}
	return out, nil
}

func Get(ctx context.Context, db *sql.DB, userID, id string) (Template, error) {
	row, err := queries(db).GetTransactionTemplateByID(ctx, sqlcdb.GetTransactionTemplateByIDParams{
		ID: id, UserID: userID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return Template{}, ErrNotFound
	}
	if err != nil {
		return Template{}, err
	}
	t := fromRow(row)
	tags, err := listTags(ctx, db, id)
	if err != nil {
		return Template{}, err
	}
	t.Tags = tags
	return t, nil
}

func Create(ctx context.Context, db *sql.DB, userID string, in Input) (Template, error) {
	count, err := queries(db).CountTransactionTemplatesByUser(ctx, userID)
	if err != nil {
		return Template{}, err
	}
	if count >= MaxPerUser {
		return Template{}, ErrLimitReached
	}
	norm, err := normalizeInput(ctx, db, userID, in)
	if err != nil {
		return Template{}, err
	}
	maxOrder, err := maxSortOrder(ctx, queries(db), userID)
	if err != nil {
		return Template{}, err
	}
	id := uuid.NewString()
	now := time.Now().UTC().Format(time.RFC3339)
	if err := queries(db).InsertTransactionTemplate(ctx, sqlcdb.InsertTransactionTemplateParams{
		ID:            id,
		UserID:        userID,
		Name:          norm.Name,
		Type:          norm.Type,
		AccountID:     norm.AccountID,
		ToAccountID:   norm.ToAccountID,
		CategoryID:    norm.CategoryID,
		SubcategoryID: norm.SubcategoryID,
		Amount:        norm.Amount,
		Description:   norm.Description,
		MerchantID:    norm.MerchantID,
		Icon:          norm.Icon,
		SortOrder:     maxOrder + 1,
		CreatedAt:     now,
		UpdatedAt:     now,
	}); err != nil {
		return Template{}, err
	}
	if err := replaceTags(ctx, db, userID, id, norm.TagIDs); err != nil {
		_, _ = queries(db).DeleteTransactionTemplate(ctx, sqlcdb.DeleteTransactionTemplateParams{ID: id, UserID: userID})
		return Template{}, err
	}
	return Get(ctx, db, userID, id)
}

func Update(ctx context.Context, db *sql.DB, userID, id string, in Input) (Template, error) {
	if _, err := Get(ctx, db, userID, id); err != nil {
		return Template{}, err
	}
	norm, err := normalizeInput(ctx, db, userID, in)
	if err != nil {
		return Template{}, err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	n, err := queries(db).UpdateTransactionTemplate(ctx, sqlcdb.UpdateTransactionTemplateParams{
		Name:          norm.Name,
		Type:          norm.Type,
		AccountID:     norm.AccountID,
		ToAccountID:   norm.ToAccountID,
		CategoryID:    norm.CategoryID,
		SubcategoryID: norm.SubcategoryID,
		Amount:        norm.Amount,
		Description:   norm.Description,
		MerchantID:    norm.MerchantID,
		Icon:          norm.Icon,
		UpdatedAt:     now,
		ID:            id,
		UserID:        userID,
	})
	if err != nil {
		return Template{}, err
	}
	if n == 0 {
		return Template{}, ErrNotFound
	}
	if err := replaceTags(ctx, db, userID, id, norm.TagIDs); err != nil {
		return Template{}, err
	}
	return Get(ctx, db, userID, id)
}

func Delete(ctx context.Context, db *sql.DB, userID, id string) error {
	n, err := queries(db).DeleteTransactionTemplate(ctx, sqlcdb.DeleteTransactionTemplateParams{
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
		return ErrInvalidReorder
	}
	seen := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		if id == "" {
			return ErrInvalidReorder
		}
		if _, ok := seen[id]; ok {
			return ErrInvalidReorder
		}
		seen[id] = struct{}{}
	}
	for _, t := range existing {
		if _, ok := seen[t.ID]; !ok {
			return ErrInvalidReorder
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
		if err := q.UpdateTransactionTemplateSortOrder(ctx, sqlcdb.UpdateTransactionTemplateSortOrderParams{
			SortOrder: int64(i + 1),
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
	name := strings.TrimSpace(in.Name)
	if name == "" || utf8.RuneCountInString(name) > MaxNameLen {
		return Input{}, ErrInvalidName
	}
	typ := strings.TrimSpace(in.Type)
	if typ != "income" && typ != "expense" && typ != "transfer" {
		return Input{}, ErrInvalidType
	}
	if in.Amount != nil && *in.Amount <= 0 {
		return Input{}, ErrInvalidAmount
	}

	var desc *string
	if in.Description != nil {
		d := strings.TrimSpace(*in.Description)
		if utf8.RuneCountInString(d) > MaxDescLen {
			return Input{}, ErrInvalidName
		}
		if d != "" {
			desc = &d
		}
	}

	var icon *string
	if in.Icon != nil {
		ic := strings.TrimSpace(*in.Icon)
		if ic != "" {
			icon = &ic
		}
	}

	out := Input{
		Name:        name,
		Type:        typ,
		Amount:      in.Amount,
		Description: desc,
		Icon:        icon,
		TagIDs:      uniqueNonEmpty(in.TagIDs),
	}

	if typ == "transfer" {
		from := trimPtr(in.AccountID)
		to := trimPtr(in.ToAccountID)
		if from == nil || to == nil || *from == *to {
			return Input{}, ErrInvalidAccount
		}
		if err := requireActiveAccount(ctx, db, userID, *from); err != nil {
			return Input{}, err
		}
		if err := requireActiveAccount(ctx, db, userID, *to); err != nil {
			return Input{}, err
		}
		out.AccountID = from
		out.ToAccountID = to
		if len(out.TagIDs) > 0 {
			return Input{}, ErrInvalidTag
		}
		return out, nil
	}

	acc := trimPtr(in.AccountID)
	if acc != nil {
		if err := requireActiveAccount(ctx, db, userID, *acc); err != nil {
			return Input{}, err
		}
		out.AccountID = acc
	}
	cat := trimPtr(in.CategoryID)
	if cat != nil {
		row, err := queries(db).GetCategoryByID(ctx, sqlcdb.GetCategoryByIDParams{ID: *cat, UserID: userID})
		if errors.Is(err, sql.ErrNoRows) {
			return Input{}, ErrInvalidCategory
		}
		if err != nil {
			return Input{}, err
		}
		if row.Type != typ || row.IsSystem != 0 {
			return Input{}, ErrInvalidCategory
		}
		out.CategoryID = cat
	}
	sub := trimPtr(in.SubcategoryID)
	if sub != nil {
		if cat == nil {
			return Input{}, ErrInvalidCategory
		}
		row, err := queries(db).GetSubcategoryByID(ctx, sqlcdb.GetSubcategoryByIDParams{ID: *sub, UserID: userID})
		if errors.Is(err, sql.ErrNoRows) {
			return Input{}, ErrInvalidCategory
		}
		if err != nil {
			return Input{}, err
		}
		if row.CategoryID != *cat {
			return Input{}, ErrInvalidCategory
		}
		out.SubcategoryID = sub
	}
	merch := trimPtr(in.MerchantID)
	if merch != nil {
		if _, err := queries(db).GetMerchantByID(ctx, sqlcdb.GetMerchantByIDParams{ID: *merch, UserID: userID}); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return Input{}, ErrInvalidMerchant
			}
			return Input{}, err
		}
		out.MerchantID = merch
	}
	for _, tagID := range out.TagIDs {
		if _, err := queries(db).GetTagByID(ctx, sqlcdb.GetTagByIDParams{ID: tagID, UserID: userID}); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return Input{}, ErrInvalidTag
			}
			return Input{}, err
		}
	}
	return out, nil
}

func requireActiveAccount(ctx context.Context, db *sql.DB, userID, accountID string) error {
	row, err := queries(db).GetAccountByID(ctx, sqlcdb.GetAccountByIDParams{ID: accountID, UserID: userID})
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

func replaceTags(ctx context.Context, db *sql.DB, userID, templateID string, tagIDs []string) error {
	if err := queries(db).DeleteTransactionTemplateTags(ctx, templateID); err != nil {
		return err
	}
	for _, tagID := range tagIDs {
		if _, err := queries(db).GetTagByID(ctx, sqlcdb.GetTagByIDParams{ID: tagID, UserID: userID}); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrInvalidTag
			}
			return err
		}
		if err := queries(db).InsertTransactionTemplateTag(ctx, sqlcdb.InsertTransactionTemplateTagParams{
			TemplateID: templateID, TagID: tagID,
		}); err != nil {
			return err
		}
	}
	return nil
}

func listTags(ctx context.Context, db *sql.DB, templateID string) ([]tag.Ref, error) {
	rows, err := queries(db).ListTagsByTemplateID(ctx, templateID)
	if err != nil {
		return nil, err
	}
	out := make([]tag.Ref, 0, len(rows))
	for _, row := range rows {
		out = append(out, tag.Ref{ID: row.ID, Name: row.Name})
	}
	return out, nil
}

func attachTags(ctx context.Context, db *sql.DB, items []Template, ids []string) error {
	for i := range items {
		items[i].Tags = []tag.Ref{}
	}
	if len(ids) == 0 {
		return nil
	}
	rows, err := queries(db).ListTagsForTemplates(ctx, ids)
	if err != nil {
		return err
	}
	byID := make(map[string]int, len(items))
	for i, item := range items {
		byID[item.ID] = i
	}
	for _, row := range rows {
		i, ok := byID[row.TemplateID]
		if !ok {
			continue
		}
		items[i].Tags = append(items[i].Tags, tag.Ref{ID: row.TagID, Name: row.TagName})
	}
	return nil
}

func fromRow(row sqlcdb.TransactionTemplate) Template {
	t := Template{
		ID:            row.ID,
		Name:          row.Name,
		Type:          row.Type,
		AccountID:     row.AccountID,
		ToAccountID:   row.ToAccountID,
		CategoryID:    row.CategoryID,
		SubcategoryID: row.SubcategoryID,
		Amount:        row.Amount,
		MerchantID:    row.MerchantID,
		SortOrder:     int(row.SortOrder),
		Tags:          []tag.Ref{},
		CreatedAt:     row.CreatedAt,
		UpdatedAt:     row.UpdatedAt,
	}
	if row.Description != nil && *row.Description != "" {
		t.Description = row.Description
	}
	if row.Icon != nil && *row.Icon != "" {
		t.Icon = row.Icon
	}
	return t
}

func maxSortOrder(ctx context.Context, q *sqlcdb.Queries, userID string) (int64, error) {
	v, err := q.MaxTransactionTemplateSortOrder(ctx, userID)
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

func uniqueNonEmpty(ids []string) []string {
	seen := make(map[string]struct{}, len(ids))
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}
