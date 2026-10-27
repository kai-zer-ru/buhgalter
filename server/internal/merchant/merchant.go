package merchant

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kai-zer-ru/buhgalter/internal/auth"
	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
	"github.com/kai-zer-ru/buhgalter/internal/scope"
)

type Merchant struct {
	ID          string   `json:"id"`
	OwnerUserID string   `json:"-"`
	Scope       string   `json:"scope,omitempty"`
	Name        string   `json:"name"`
	Icon        string   `json:"icon"`
	Aliases     []string `json:"aliases"`
	CreatedAt   string   `json:"created_at"`
}

var (
	ErrNotFound     = errors.New("merchant not found")
	ErrNameTaken    = errors.New("merchant name already exists")
	ErrInvalidName  = errors.New("invalid merchant name")
	ErrInvalidMerge = errors.New("invalid merchant merge")
)

func queries(db sqlcdb.DBTX) *sqlcdb.Queries {
	return sqlcdb.New(db)
}

func normalizeIcon(icon string) string {
	icon = strings.TrimSpace(icon)
	if icon == "" {
		return "default"
	}
	return icon
}

func containsFold(list []string, name string) bool {
	for _, item := range list {
		if strings.EqualFold(item, name) {
			return true
		}
	}
	return false
}

func sameFold(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !strings.EqualFold(a[i], b[i]) {
			return false
		}
	}
	return true
}

// normalizeAliases trims, drops empty values and the primary name, and dedupes with EqualFold.
func normalizeAliases(raw []string, primary string) []string {
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		name := strings.TrimSpace(item)
		if name == "" || strings.EqualFold(name, primary) {
			continue
		}
		if containsFold(out, name) {
			continue
		}
		out = append(out, name)
	}
	return out
}

func withTx(ctx context.Context, db sqlcdb.DBTX, fn func(*sql.Tx) error) error {
	if tx, ok := db.(*sql.Tx); ok {
		return fn(tx)
	}
	sqlDB, ok := db.(*sql.DB)
	if !ok {
		return errors.New("merchant: unsupported db")
	}
	tx, err := sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func isUnique(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed")
}

func fromRow(row sqlcdb.Merchant) Merchant {
	return Merchant{
		ID: row.ID, OwnerUserID: row.UserID, Name: row.Name, Icon: row.Icon,
		Aliases: []string{}, CreatedAt: row.CreatedAt,
	}
}

func attachAliases(ctx context.Context, db sqlcdb.DBTX, ownerIDs []string, items []Merchant) ([]Merchant, error) {
	if items == nil {
		items = []Merchant{}
	}
	rows, err := queries(db).ListMerchantAliasesByUser(ctx, ownerIDs)
	if err != nil {
		return nil, err
	}
	byID := make(map[string][]string, len(items))
	for _, row := range rows {
		byID[row.MerchantID] = append(byID[row.MerchantID], row.Name)
	}
	for i := range items {
		aliases := byID[items[i].ID]
		if aliases == nil {
			aliases = []string{}
		}
		items[i].Aliases = aliases
	}
	return items, nil
}

func List(ctx context.Context, db sqlcdb.DBTX, userID string) ([]Merchant, error) {
	return ListByScope(ctx, db, auth.Scope{UserID: userID}, "")
}

func ListByScope(ctx context.Context, db sqlcdb.DBTX, sc auth.Scope, view string) ([]Merchant, error) {
	ids := scope.FilterOwnerIDs(sc, view)
	if len(ids) == 0 {
		return []Merchant{}, nil
	}
	rows, err := queries(db).ListMerchantsByUser(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make([]Merchant, 0, len(rows))
	for _, row := range rows {
		m := fromRow(row)
		m.Scope = scope.AccountScope(sc, m.OwnerUserID)
		out = append(out, m)
	}
	return attachAliases(ctx, db, ids, out)
}

func Get(ctx context.Context, db sqlcdb.DBTX, userID, id string) (Merchant, error) {
	return GetByScope(ctx, db, auth.Scope{UserID: userID}, id)
}

func GetByScope(ctx context.Context, db sqlcdb.DBTX, sc auth.Scope, id string) (Merchant, error) {
	ids := sc.OwnerIDs()
	if len(ids) == 0 {
		return Merchant{}, ErrNotFound
	}
	row, err := queries(db).GetMerchantByID(ctx, sqlcdb.GetMerchantByIDParams{ID: id, OwnerIds: ids})
	if errors.Is(err, sql.ErrNoRows) {
		return Merchant{}, ErrNotFound
	}
	if err != nil {
		return Merchant{}, err
	}
	m := fromRow(row)
	m.Scope = scope.AccountScope(sc, m.OwnerUserID)
	items, err := attachAliases(ctx, db, ids, []Merchant{m})
	if err != nil {
		return Merchant{}, err
	}
	return items[0], nil
}

func findExisting(ctx context.Context, db sqlcdb.DBTX, userID, name string) (Merchant, bool, error) {
	row, err := queries(db).GetMerchantByName(ctx, sqlcdb.GetMerchantByNameParams{UserID: userID, Name: name})
	if err == nil {
		m, gerr := Get(ctx, db, userID, row.ID)
		return m, true, gerr
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return Merchant{}, false, err
	}
	rows, err := queries(db).ListMerchantsByUser(ctx, []string{userID})
	if err != nil {
		return Merchant{}, false, err
	}
	for _, merchant := range rows {
		if strings.EqualFold(merchant.Name, name) {
			m, gerr := Get(ctx, db, userID, merchant.ID)
			return m, true, gerr
		}
	}
	aliases, err := queries(db).ListMerchantAliasesByUser(ctx, []string{userID})
	if err != nil {
		return Merchant{}, false, err
	}
	for _, alias := range aliases {
		if strings.EqualFold(alias.Name, name) {
			m, gerr := Get(ctx, db, userID, alias.MerchantID)
			return m, true, gerr
		}
	}
	return Merchant{}, false, nil
}

func nameTaken(ctx context.Context, db sqlcdb.DBTX, userID, name string, allow map[string]struct{}) (bool, error) {
	rows, err := queries(db).ListMerchantsByUser(ctx, []string{userID})
	if err != nil {
		return false, err
	}
	for _, row := range rows {
		if _, ok := allow[row.ID]; ok {
			continue
		}
		if strings.EqualFold(row.Name, name) {
			return true, nil
		}
	}
	aliases, err := queries(db).ListMerchantAliasesByUser(ctx, []string{userID})
	if err != nil {
		return false, err
	}
	for _, alias := range aliases {
		if _, ok := allow[alias.MerchantID]; ok {
			continue
		}
		if strings.EqualFold(alias.Name, name) {
			return true, nil
		}
	}
	return false, nil
}

func writeAliases(ctx context.Context, db sqlcdb.DBTX, userID, merchantID string, aliases []string) error {
	if err := queries(db).DeleteMerchantAliasesByMerchant(ctx, sqlcdb.DeleteMerchantAliasesByMerchantParams{
		MerchantID: merchantID, UserID: userID,
	}); err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	for _, name := range aliases {
		err := queries(db).InsertMerchantAlias(ctx, sqlcdb.InsertMerchantAliasParams{
			ID: uuid.NewString(), UserID: userID, MerchantID: merchantID, Name: name, CreatedAt: now,
		})
		if isUnique(err) {
			return ErrNameTaken
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func Create(ctx context.Context, db sqlcdb.DBTX, userID, name, icon string) (Merchant, error) {
	return CreateWithAliases(ctx, db, userID, name, icon, nil)
}

// CreateWithAliases inserts a merchant and its aliases.
// A match on the primary name or any alias returns that merchant unchanged.
// nil aliases is an empty list. Empty strings and the primary name are dropped.
func CreateWithAliases(ctx context.Context, db sqlcdb.DBTX, userID, name, icon string, aliases []string) (Merchant, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Merchant{}, ErrInvalidName
	}
	icon = normalizeIcon(icon)
	if existing, ok, err := findExisting(ctx, db, userID, name); err != nil {
		return Merchant{}, err
	} else if ok {
		return existing, nil
	}
	normalized := normalizeAliases(aliases, name)
	for _, alias := range normalized {
		taken, err := nameTaken(ctx, db, userID, alias, nil)
		if err != nil {
			return Merchant{}, err
		}
		if taken {
			return Merchant{}, ErrNameTaken
		}
	}
	id := uuid.NewString()
	now := time.Now().UTC().Format(time.RFC3339)
	err := withTx(ctx, db, func(tx *sql.Tx) error {
		err := queries(tx).InsertMerchant(ctx, sqlcdb.InsertMerchantParams{
			ID: id, UserID: userID, Name: name, Icon: icon, CreatedAt: now,
		})
		if isUnique(err) {
			return ErrNameTaken
		}
		if err != nil {
			return err
		}
		return writeAliases(ctx, tx, userID, id, normalized)
	})
	if errors.Is(err, ErrNameTaken) {
		if existing, ok, ferr := findExisting(ctx, db, userID, name); ferr == nil && ok {
			return existing, nil
		}
		return Merchant{}, err
	}
	if err != nil {
		return Merchant{}, err
	}
	return Get(ctx, db, userID, id)
}

// Update changes the name and icon. aliases == nil leaves the list untouched;
// a non-nil slice replaces it. Renaming onto this merchant's own alias consumes that alias.
func Update(ctx context.Context, db sqlcdb.DBTX, userID, id, name, icon string, aliases *[]string) (Merchant, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Merchant{}, ErrInvalidName
	}
	existing, err := Get(ctx, db, userID, id)
	if err != nil {
		return Merchant{}, err
	}
	icon = strings.TrimSpace(icon)
	if icon == "" {
		icon = existing.Icon
	}
	if icon == "" {
		icon = "default"
	}
	allow := map[string]struct{}{id: {}}
	taken, err := nameTaken(ctx, db, userID, name, allow)
	if err != nil {
		return Merchant{}, err
	}
	if taken {
		return Merchant{}, ErrNameTaken
	}
	next := normalizeAliases(existing.Aliases, name)
	if aliases != nil {
		next = normalizeAliases(*aliases, name)
	}
	for _, alias := range next {
		taken, err := nameTaken(ctx, db, userID, alias, allow)
		if err != nil {
			return Merchant{}, err
		}
		if taken {
			return Merchant{}, ErrNameTaken
		}
	}
	write := aliases != nil || !sameFold(existing.Aliases, next)
	err = withTx(ctx, db, func(tx *sql.Tx) error {
		n, err := queries(tx).UpdateMerchant(ctx, sqlcdb.UpdateMerchantParams{
			Name: name, Icon: icon, ID: id, UserID: userID,
		})
		if isUnique(err) {
			return ErrNameTaken
		}
		if err != nil {
			return err
		}
		if n == 0 {
			return ErrNotFound
		}
		if !write {
			return nil
		}
		return writeAliases(ctx, tx, userID, id, next)
	})
	if err != nil {
		return Merchant{}, err
	}
	return Get(ctx, db, userID, id)
}

func Delete(ctx context.Context, db sqlcdb.DBTX, userID, id string) error {
	if _, err := Get(ctx, db, userID, id); err != nil {
		return err
	}
	n, err := queries(db).DeleteMerchant(ctx, sqlcdb.DeleteMerchantParams{ID: id, UserID: userID})
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// Resolve returns merchant id from id or name (get-or-create). Empty both → nil.
// A name that matches an alias resolves to that merchant and does not create another.
func Resolve(ctx context.Context, db sqlcdb.DBTX, userID string, merchantID, merchantName *string) (*string, error) {
	if merchantID != nil && *merchantID != "" {
		if _, err := Get(ctx, db, userID, *merchantID); err != nil {
			return nil, err
		}
		return merchantID, nil
	}
	if merchantName != nil && strings.TrimSpace(*merchantName) != "" {
		m, err := Create(ctx, db, userID, *merchantName, "default")
		if err != nil && !errors.Is(err, ErrNameTaken) {
			return nil, err
		}
		if errors.Is(err, ErrNameTaken) {
			found, ok, ferr := findExisting(ctx, db, userID, strings.TrimSpace(*merchantName))
			if ferr != nil {
				return nil, ferr
			}
			if !ok {
				return nil, err
			}
			return &found.ID, nil
		}
		return &m.ID, nil
	}
	return nil, nil
}

func dedupeIDs(ids []string) []string {
	out := make([]string, 0, len(ids))
	seen := map[string]struct{}{}
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

// Merge folds sources into target. Target name and icon stay. Source names and aliases
// become aliases (copies of the target primary name are dropped). Transactions and
// templates move to the target, then sources are deleted. Any name owned by a third
// merchant aborts the whole transaction.
func Merge(ctx context.Context, db *sql.DB, userID, targetID string, sourceIDs []string) (Merchant, error) {
	ids := dedupeIDs(sourceIDs)
	if strings.TrimSpace(targetID) == "" || len(ids) == 0 {
		return Merchant{}, ErrInvalidMerge
	}
	for _, id := range ids {
		if id == targetID {
			return Merchant{}, ErrInvalidMerge
		}
	}
	err := withTx(ctx, db, func(tx *sql.Tx) error {
		target, err := Get(ctx, tx, userID, targetID)
		if err != nil {
			return err
		}
		allow := map[string]struct{}{targetID: {}}
		sources := make([]Merchant, 0, len(ids))
		for _, id := range ids {
			src, err := Get(ctx, tx, userID, id)
			if err != nil {
				return err
			}
			sources = append(sources, src)
			allow[id] = struct{}{}
		}
		seen := append([]string{}, target.Name)
		seen = append(seen, target.Aliases...)
		var toAdd []string
		for _, src := range sources {
			candidates := make([]string, 0, 1+len(src.Aliases))
			candidates = append(candidates, src.Name)
			candidates = append(candidates, src.Aliases...)
			for _, raw := range candidates {
				name := strings.TrimSpace(raw)
				if name == "" || containsFold(seen, name) {
					continue
				}
				taken, err := nameTaken(ctx, tx, userID, name, allow)
				if err != nil {
					return err
				}
				if taken {
					return ErrNameTaken
				}
				seen = append(seen, name)
				toAdd = append(toAdd, name)
			}
		}
		q := queries(tx)
		for _, src := range sources {
			targetRef := target.ID
			sourceRef := src.ID
			if err := q.ReassignTransactionsMerchant(ctx, sqlcdb.ReassignTransactionsMerchantParams{
				TargetID: &targetRef, UserID: userID, SourceID: &sourceRef,
			}); err != nil {
				return err
			}
			if err := q.ReassignTemplateMerchant(ctx, sqlcdb.ReassignTemplateMerchantParams{
				TargetID: &targetRef, UserID: userID, SourceID: &sourceRef,
			}); err != nil {
				return err
			}
		}
		for _, src := range sources {
			n, err := q.DeleteMerchant(ctx, sqlcdb.DeleteMerchantParams{ID: src.ID, UserID: userID})
			if err != nil {
				return err
			}
			if n == 0 {
				return ErrNotFound
			}
		}
		now := time.Now().UTC().Format(time.RFC3339)
		for _, name := range toAdd {
			err := q.InsertMerchantAlias(ctx, sqlcdb.InsertMerchantAliasParams{
				ID: uuid.NewString(), UserID: userID, MerchantID: target.ID, Name: name, CreatedAt: now,
			})
			if isUnique(err) {
				return ErrNameTaken
			}
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return Merchant{}, err
	}
	return Get(ctx, db, userID, targetID)
}
