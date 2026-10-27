package transaction

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/kai-zer-ru/buhgalter/internal/auth"
	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
	"github.com/kai-zer-ru/buhgalter/internal/scope"
)

type PersonRef struct {
	UserID      string `json:"user_id"`
	DisplayName string `json:"display_name"`
}

var ErrScopeMismatch = errors.New("scope mismatch")
var ErrTransferForeign = errors.New("transfer foreign account")

func withRowMeta(t Transaction, ownerUserID string, createdByID *string, createdByName string) Transaction {
	t.OwnerUserID = ownerUserID
	if createdByID != nil && *createdByID != "" {
		t.CreatedBy = &PersonRef{UserID: *createdByID, DisplayName: createdByName}
	}
	return t
}

func viewerFrom(actorID string, ownerIDs []string) auth.Scope {
	sc := auth.Scope{UserID: actorID}
	for _, id := range ownerIDs {
		if id != "" && id != actorID {
			sc.PrincipalID = id
			break
		}
	}
	return sc
}

func decorateVisible(ctx context.Context, db sqlcdb.DBTX, actorID string, ownerIDs []string, t *Transaction) {
	t.Scope = scope.AccountScope(viewerFrom(actorID, ownerIDs), t.OwnerUserID)
	maskTransferAccount(ctx, db, ownerIDs, t)
}

func maskTransferAccount(ctx context.Context, db sqlcdb.DBTX, ownerIDs []string, t *Transaction) {
	if t.TransferAccountID == nil || *t.TransferAccountID == "" {
		return
	}
	otherID := *t.TransferAccountID
	_, err := queries(db).GetAccountByID(ctx, sqlcdb.GetAccountByIDParams{ID: otherID, OwnerIds: ownerIDs})
	if err == nil {
		return
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return
	}
	label := "Личный счёт"
	row, err := queries(db).GetAccountOwnerDisplay(ctx, otherID)
	if err == nil && row.DisplayName != "" {
		label = fmt.Sprintf("Личный счёт %s", row.DisplayName)
	}
	t.TransferAccountID = nil
	t.TransferAccountName = ""
	t.TransferAccountStatus = ""
	t.TransferAccountMasked = true
	t.TransferAccountLabel = label
}

func assertCatalogOwner(ctx context.Context, db sqlcdb.DBTX, principalID, categoryID string) error {
	if categoryID == "" {
		return nil
	}
	uid, err := queries(db).GetCategoryUserID(ctx, categoryID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrInvalidCategory
	}
	if err != nil {
		return err
	}
	if uid != principalID {
		return ErrScopeMismatch
	}
	return nil
}

func assertMerchantOwner(ctx context.Context, db sqlcdb.DBTX, principalID, merchantID string) error {
	if merchantID == "" {
		return nil
	}
	uid, err := queries(db).GetMerchantUserID(ctx, merchantID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrInvalidMerchant
	}
	if err != nil {
		return err
	}
	if uid != principalID {
		return ErrScopeMismatch
	}
	return nil
}

func assertTagOwner(ctx context.Context, db sqlcdb.DBTX, principalID, tagID string) error {
	if tagID == "" {
		return nil
	}
	uid, err := queries(db).GetTagUserID(ctx, tagID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrInvalidTag
	}
	if err != nil {
		return err
	}
	if uid != principalID {
		return ErrScopeMismatch
	}
	return nil
}

func assertStoreOwner(ctx context.Context, db sqlcdb.DBTX, principalID, storeID string) error {
	if storeID == "" {
		return nil
	}
	uid, err := queries(db).GetStoreUserID(ctx, storeID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrInvalidAccount
	}
	if err != nil {
		return err
	}
	if uid != principalID {
		return ErrScopeMismatch
	}
	return nil
}

func assertSubcategoryOwner(ctx context.Context, db sqlcdb.DBTX, principalID, subID string) error {
	if subID == "" {
		return nil
	}
	row, err := queries(db).GetSubcategoryOwner(ctx, subID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrInvalidSubcategory
	}
	if err != nil {
		return err
	}
	if row.UserID != principalID {
		return ErrScopeMismatch
	}
	return nil
}

func requireVisibleAccounts(ctx context.Context, db sqlcdb.DBTX, ownerIDs []string, accountIDs ...string) error {
	for _, id := range uniqueAccountIDs(accountIDs...) {
		_, err := queries(db).GetAccountByID(ctx, sqlcdb.GetAccountByIDParams{ID: id, OwnerIds: ownerIDs})
		if errors.Is(err, sql.ErrNoRows) {
			return ErrTransferForeign
		}
		if err != nil {
			return err
		}
	}
	return nil
}
