package tag

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
)

type Tag struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
}

type Ref struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

var (
	ErrNotFound    = errors.New("tag not found")
	ErrNameTaken   = errors.New("tag name already exists")
	ErrInvalidName = errors.New("invalid tag name")
)

func queries(db sqlcdb.DBTX) *sqlcdb.Queries {
	return sqlcdb.New(db)
}

func List(ctx context.Context, db *sql.DB, userID, q string) ([]Tag, error) {
	q = strings.TrimSpace(q)
	qPtr := q
	rows, err := queries(db).ListTagsByUser(ctx, sqlcdb.ListTagsByUserParams{
		UserID:  userID,
		Column2: q,
		Column3: &qPtr,
	})
	if err != nil {
		return nil, err
	}
	out := make([]Tag, 0, len(rows))
	for _, row := range rows {
		out = append(out, fromRow(row))
	}
	return out, nil
}

func Create(ctx context.Context, db *sql.DB, userID, name string) (Tag, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Tag{}, ErrInvalidName
	}
	existing, err := queries(db).GetTagByName(ctx, sqlcdb.GetTagByNameParams{
		UserID: userID, Name: name,
	})
	if err == nil {
		return fromRow(existing), nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return Tag{}, err
	}
	// SQLite lower()/NOCASE — ASCII; для кириллицы ищем через EqualFold.
	if t, ok := findByEqualFold(ctx, db, userID, name); ok {
		return t, nil
	}

	id := uuid.NewString()
	now := time.Now().UTC().Format(time.RFC3339)
	if err := queries(db).InsertTag(ctx, sqlcdb.InsertTagParams{
		ID: id, UserID: userID, Name: name, CreatedAt: now,
	}); err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return Tag{}, ErrNameTaken
		}
		return Tag{}, err
	}
	return Get(ctx, db, userID, id)
}

func findByEqualFold(ctx context.Context, db *sql.DB, userID, name string) (Tag, bool) {
	rows, err := queries(db).ListTagsByUser(ctx, sqlcdb.ListTagsByUserParams{
		UserID: userID, Column2: "", Column3: strPtr(""),
	})
	if err != nil {
		return Tag{}, false
	}
	for _, row := range rows {
		if strings.EqualFold(row.Name, name) {
			return fromRow(row), true
		}
	}
	return Tag{}, false
}

func strPtr(s string) *string { return &s }

func Get(ctx context.Context, db *sql.DB, userID, id string) (Tag, error) {
	row, err := queries(db).GetTagByID(ctx, sqlcdb.GetTagByIDParams{ID: id, UserID: userID})
	if errors.Is(err, sql.ErrNoRows) {
		return Tag{}, ErrNotFound
	}
	if err != nil {
		return Tag{}, err
	}
	return fromRow(row), nil
}

func Update(ctx context.Context, db *sql.DB, userID, id, name string) (Tag, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Tag{}, ErrInvalidName
	}
	if _, err := Get(ctx, db, userID, id); err != nil {
		return Tag{}, err
	}
	n, err := queries(db).UpdateTagName(ctx, sqlcdb.UpdateTagNameParams{
		Name: name, ID: id, UserID: userID,
	})
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return Tag{}, ErrNameTaken
		}
		return Tag{}, err
	}
	if n == 0 {
		return Tag{}, ErrNotFound
	}
	return Get(ctx, db, userID, id)
}

func Delete(ctx context.Context, db *sql.DB, userID, id string) error {
	if _, err := Get(ctx, db, userID, id); err != nil {
		return err
	}
	n, err := queries(db).DeleteTag(ctx, sqlcdb.DeleteTagParams{ID: id, UserID: userID})
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// ResolveIDs merges tag_ids and tag_names (get-or-create), returns unique ids.
func ResolveIDs(ctx context.Context, db *sql.DB, userID string, tagIDs, tagNames []string) ([]string, error) {
	seen := make(map[string]struct{})
	var out []string
	add := func(id string) {
		if id == "" {
			return
		}
		if _, ok := seen[id]; ok {
			return
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	for _, id := range tagIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, err := Get(ctx, db, userID, id); err != nil {
			return nil, err
		}
		add(id)
	}
	for _, name := range tagNames {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		t, err := Create(ctx, db, userID, name)
		if err != nil && !errors.Is(err, ErrNameTaken) {
			return nil, err
		}
		if errors.Is(err, ErrNameTaken) {
			row, err := queries(db).GetTagByName(ctx, sqlcdb.GetTagByNameParams{
				UserID: userID, Name: name,
			})
			if err != nil {
				return nil, err
			}
			add(row.ID)
			continue
		}
		add(t.ID)
	}
	return out, nil
}

func SetForTransaction(ctx context.Context, db sqlcdb.DBTX, transactionID string, tagIDs []string) error {
	q := queries(db)
	if err := q.DeleteTransactionTags(ctx, transactionID); err != nil {
		return err
	}
	for _, id := range tagIDs {
		if err := q.InsertTransactionTag(ctx, sqlcdb.InsertTransactionTagParams{
			TransactionID: transactionID,
			TagID:         id,
		}); err != nil {
			return err
		}
	}
	return nil
}

func ListForTransaction(ctx context.Context, db *sql.DB, transactionID string) ([]Ref, error) {
	rows, err := queries(db).ListTagsByTransactionID(ctx, transactionID)
	if err != nil {
		return nil, err
	}
	out := make([]Ref, 0, len(rows))
	for _, row := range rows {
		out = append(out, Ref{ID: row.ID, Name: row.Name})
	}
	return out, nil
}

func MapForTransactions(ctx context.Context, db *sql.DB, transactionIDs []string) (map[string][]Ref, error) {
	out := make(map[string][]Ref, len(transactionIDs))
	if len(transactionIDs) == 0 {
		return out, nil
	}
	rows, err := queries(db).ListTagsForTransactions(ctx, transactionIDs)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.TransactionID] = append(out[row.TransactionID], Ref{ID: row.TagID, Name: row.TagName})
	}
	return out, nil
}

func fromRow(row sqlcdb.Tag) Tag {
	return Tag{ID: row.ID, Name: row.Name, CreatedAt: row.CreatedAt}
}
