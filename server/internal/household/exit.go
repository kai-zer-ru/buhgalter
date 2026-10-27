package household

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/kai-zer-ru/buhgalter/internal/auth"
	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
)

func GetMember(ctx context.Context, db *sql.DB, householdID, userID string) (sqlcdb.HouseholdMember, error) {
	row, err := sqlcdb.New(db).GetHouseholdMember(ctx, sqlcdb.GetHouseholdMemberParams{
		HouseholdID: householdID, UserID: userID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return sqlcdb.HouseholdMember{}, ErrMemberNotFound
	}
	return row, err
}

func clearLeavingRefs(ctx context.Context, q *sqlcdb.Queries, userID, principalID string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	if err := q.DeleteUserChangeEventsForPrincipalTransactions(ctx, sqlcdb.DeleteUserChangeEventsForPrincipalTransactionsParams{
		UserID: userID, UserID_2: principalID,
	}); err != nil {
		return err
	}
	if err := q.ClearPersonalRuleDefaultsOnPrincipalAccounts(ctx, sqlcdb.ClearPersonalRuleDefaultsOnPrincipalAccountsParams{
		UpdatedAt: now, UserID: userID, UserID_2: principalID,
	}); err != nil {
		return err
	}
	if err := q.ClearPersonalTemplateAccountsOnPrincipal(ctx, sqlcdb.ClearPersonalTemplateAccountsOnPrincipalParams{
		UpdatedAt: now, UserID: userID, UserID_2: principalID,
	}); err != nil {
		return err
	}
	if err := q.ClearPersonalTemplateToAccountsOnPrincipal(ctx, sqlcdb.ClearPersonalTemplateToAccountsOnPrincipalParams{
		UpdatedAt: now, UserID: userID, UserID_2: principalID,
	}); err != nil {
		return err
	}
	if err := q.DeleteInterceptBankBindingsOnPrincipalAccounts(ctx, sqlcdb.DeleteInterceptBankBindingsOnPrincipalAccountsParams{
		UserID: userID, UserID_2: principalID,
	}); err != nil {
		return err
	}
	return q.DeleteInterceptCardBindingsOnPrincipalAccounts(ctx, sqlcdb.DeleteInterceptCardBindingsOnPrincipalAccountsParams{
		UserID: userID, UserID_2: principalID,
	})
}

func wipePrincipalRemainder(ctx context.Context, q *sqlcdb.Queries, userID string) error {
	steps := []func() error{
		func() error { return q.DeleteUserCreditPayments(ctx, userID) },
		func() error { return q.DeleteUserDebtTransactions(ctx, userID) },
		func() error { return q.DeleteUserCredits(ctx, userID) },
		func() error { return q.DeleteUserDebts(ctx, userID) },
		func() error { return q.DeleteUserDebtors(ctx, userID) },
		func() error { return q.DeleteUserSubscriptions(ctx, userID) },
		func() error { return q.DeleteUserRecurringOperations(ctx, userID) },
		func() error { return q.DeleteUserBudgets(ctx, userID) },
		func() error { return q.DeleteUserTransactionTemplates(ctx, userID) },
		func() error { return q.DeleteUserCategoryRules(ctx, userID) },
		func() error { return q.DeleteUserTransactions(ctx, userID) },
		func() error { return q.DeleteUserMerchantAliases(ctx, userID) },
		func() error { return q.DeleteUserMerchants(ctx, userID) },
		func() error { return q.DeleteUserTags(ctx, userID) },
		func() error { return q.ClearUserAccountSelfRefs(ctx, userID) },
		func() error { return q.DeleteUserAccounts(ctx, userID) },
		func() error { return q.DeleteUserReceiptImports(ctx, userID) },
		func() error { return q.DeleteUserReceiptScanJobs(ctx, userID) },
		func() error { return q.DeleteUserReceiptFiscalPreviews(ctx, userID) },
		func() error { return q.DeleteUserStores(ctx, userID) },
		func() error { return q.DeleteUserCategories(ctx, userID) },
		func() error { return q.DeleteUserImportJobs(ctx, userID) },
		func() error { return q.DeleteUserImportIdempotency(ctx, userID) },
		func() error { return q.DeleteUserChangeEvents(ctx, userID) },
		func() error { return q.DeleteUserNotificationLog(ctx, userID) },
		func() error { return q.DeleteUserNotificationSettings(ctx, userID) },
		func() error { return q.DeleteUserNotificationTemplates(ctx, userID) },
		func() error { return q.DeleteUserReceiptImportItemCategories(ctx, userID) },
		func() error { return q.DeleteNotificationInterceptBankBindings(ctx, userID) },
		func() error { return q.DeleteNotificationInterceptCardBindings(ctx, userID) },
	}
	for _, fn := range steps {
		if err := fn(); err != nil {
			return err
		}
	}
	return nil
}

func dissolveRecipient(ctx context.Context, q *sqlcdb.Queries, householdID, dissolvingOwner string, original *string) string {
	if original == nil || *original == "" {
		return dissolvingOwner
	}
	_, err := q.GetHouseholdMember(ctx, sqlcdb.GetHouseholdMemberParams{HouseholdID: householdID, UserID: *original})
	if err != nil {
		return dissolvingOwner
	}
	return *original
}

type DissolveResult struct {
	AccountIDs []string
	Moved      MovedCounts
}

func PrepareAdminDelete(ctx context.Context, db *sql.DB, targetID string) error {
	q := sqlcdb.New(db)
	hh, err := q.GetHouseholdByUserID(ctx, targetID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	count, err := q.CountHouseholdMembers(ctx, hh.ID)
	if err != nil {
		return err
	}
	if hh.Role == "owner" {
		if count > 1 {
			return ErrOwnerDeleteBlocked
		}
		return Dissolve(ctx, db, auth.Scope{
			UserID: targetID, HouseholdID: hh.ID, PrincipalID: hh.PrincipalUserID, Role: "owner",
		}, true)
	}
	ownerScope := auth.Scope{
		UserID: hh.OwnerUserID, HouseholdID: hh.ID, PrincipalID: hh.PrincipalUserID, Role: "owner",
	}
	if err := Kick(ctx, db, ownerScope, targetID); err != nil {
		return err
	}
	return q.ReassignInviteCreatedBy(ctx, sqlcdb.ReassignInviteCreatedByParams{
		CreatedBy: hh.OwnerUserID, HouseholdID: hh.ID, CreatedBy_2: targetID,
	})
}

func DissolveWithData(ctx context.Context, db *sql.DB, scope auth.Scope, confirmMembers bool) (DissolveResult, error) {
	var out DissolveResult
	if !scope.InHousehold() {
		return out, ErrNotMember
	}
	if !scope.IsOwner() {
		return out, ErrOwnerOnly
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return out, err
	}
	defer func() { _ = tx.Rollback() }()
	q := sqlcdb.New(tx)

	count, err := q.CountHouseholdMembers(ctx, scope.HouseholdID)
	if err != nil {
		return out, err
	}
	if count > 1 && !confirmMembers {
		return out, ErrHasMembers
	}

	accs, err := q.ListAccountsForMove(ctx, scope.PrincipalID)
	if err != nil {
		return out, err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	for _, acc := range accs {
		to := dissolveRecipient(ctx, q, scope.HouseholdID, scope.UserID, acc.OriginalOwnerUserID)
		moved, err := MoveAccount(ctx, tx, acc.ID, scope.PrincipalID, to, MoveOpts{AllowDeleted: acc.Status == "deleted"})
		if err != nil {
			return out, err
		}
		out.AccountIDs = append(out.AccountIDs, acc.ID)
		out.Moved.Transactions += moved.Transactions
		out.Moved.Credits += moved.Credits
		out.Moved.Subscriptions += moved.Subscriptions
		out.Moved.Recurring += moved.Recurring
		out.Moved.Budgets += moved.Budgets
		out.Moved.Debts += moved.Debts
		out.Moved.Templates += moved.Templates
	}

	rm := newRemapper(tx, q, scope.PrincipalID, scope.UserID, now)
	if err := rm.remapAccountless(ctx); err != nil {
		return out, err
	}
	if err := wipePrincipalRemainder(ctx, q, scope.PrincipalID); err != nil {
		return out, err
	}
	if table, err := principalLeftoverTable(ctx, tx, scope.PrincipalID); err != nil {
		return out, err
	} else if table != "" {
		slog.Error("household dissolve leftover", "table", table, "principal_id", scope.PrincipalID)
		return out, fmt.Errorf("principal leftover in %s", table)
	}

	if err := q.DeletePendingHouseholdInvites(ctx, scope.HouseholdID); err != nil {
		return out, err
	}
	if err := q.DeleteHouseholdMembersByHousehold(ctx, scope.HouseholdID); err != nil {
		return out, err
	}
	if err := q.DeleteHousehold(ctx, scope.HouseholdID); err != nil {
		return out, err
	}
	if err := q.DeleteUser(ctx, scope.PrincipalID); err != nil {
		return out, err
	}
	if err := tx.Commit(); err != nil {
		return out, err
	}
	return out, nil
}
