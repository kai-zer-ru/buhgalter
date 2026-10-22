package user

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kai-zer-ru/buhgalter/internal/auth"
	"github.com/kai-zer-ru/buhgalter/internal/categoryseed"
	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
)

func TestResetUserDataWipesLedgerAndReseedsCategories(t *testing.T) {
	h, handle, user := testEnv(t)
	ctx := context.Background()
	db := handle.DB()

	_, err := db.ExecContext(ctx, `
		INSERT INTO accounts (id, user_id, name, type, initial_balance, current_balance, status, created_at, updated_at)
		VALUES ('acc-wipe', ?, 'Кошелёк', 'cash', 100000, 90000, 'active', datetime('now'), datetime('now'))`,
		user.ID)
	if err != nil {
		t.Fatal(err)
	}
	cats, err := sqlcdb.New(db).ListCategoriesByUser(ctx, user.ID)
	if err != nil || len(cats) == 0 {
		t.Fatalf("categories: %v n=%d", err, len(cats))
	}
	var expenseID string
	for _, c := range cats {
		if c.Type == "expense" && c.IsSystem == 0 {
			expenseID = c.ID
			break
		}
	}
	if expenseID == "" {
		t.Fatal("no expense category")
	}
	_, err = db.ExecContext(ctx, `
		INSERT INTO transactions (id, user_id, account_id, type, kind, amount, category_id, transaction_date, created_at, updated_at)
		VALUES ('tx-wipe', ?, 'acc-wipe', 'expense', 'manual', 10000, ?, '2020-01-15 12:00:00', datetime('now'), datetime('now'))`,
		user.ID, expenseID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.ExecContext(ctx, `
		INSERT INTO merchants (id, user_id, name, created_at) VALUES ('m-wipe', ?, 'Shop', datetime('now'))`,
		user.ID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.ExecContext(ctx, `
		INSERT INTO categories (id, user_id, name, type, icon, sort_order, is_primary, is_system, created_at)
		VALUES ('cat-custom', ?, 'Кастом', 'expense', 'default', 50, 0, 0, datetime('now'))`,
		user.ID)
	if err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	h.DeleteData(rec, withUser(t, user, http.MethodDelete, "/user/data", nil))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	q := sqlcdb.New(db)
	accounts, err := q.ListAccountsByUserActive(ctx, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(accounts) != 0 {
		t.Fatalf("accounts left: %d", len(accounts))
	}
	var txCount int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM transactions WHERE user_id = ?`, user.ID).Scan(&txCount); err != nil {
		t.Fatal(err)
	}
	if txCount != 0 {
		t.Fatalf("transactions left: %d", txCount)
	}
	var merchantCount int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM merchants WHERE user_id = ?`, user.ID).Scan(&merchantCount); err != nil {
		t.Fatal(err)
	}
	if merchantCount != 0 {
		t.Fatalf("merchants left: %d", merchantCount)
	}
	after, err := q.ListCategoriesByUser(ctx, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(after) != categoryseed.DefaultCount {
		t.Fatalf("categories %d want %d", len(after), categoryseed.DefaultCount)
	}
	for _, c := range after {
		if c.Name == "Кастом" {
			t.Fatal("custom category survived")
		}
	}
	loaded, err := q.GetUserByID(ctx, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Login != user.Login {
		t.Fatalf("login %q", loaded.Login)
	}
}

func TestResetUserDataDoesNotTouchOtherUser(t *testing.T) {
	_, handle, user := testEnv(t)
	ctx := context.Background()
	db := handle.DB()

	otherID, err := auth.CreateUser(ctx, db, "otherwipe", "hash", "Other", false, auth.UserStatusActive)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.ExecContext(ctx, `
		INSERT INTO accounts (id, user_id, name, type, initial_balance, current_balance, status, created_at, updated_at)
		VALUES ('acc-other', ?, 'Чужой', 'cash', 1, 1, 'active', datetime('now'), datetime('now'))`,
		otherID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.ExecContext(ctx, `
		INSERT INTO accounts (id, user_id, name, type, initial_balance, current_balance, status, created_at, updated_at)
		VALUES ('acc-self', ?, 'Свой', 'cash', 1, 1, 'active', datetime('now'), datetime('now'))`,
		user.ID)
	if err != nil {
		t.Fatal(err)
	}

	if err := ResetUserData(ctx, db, user.ID); err != nil {
		t.Fatal(err)
	}

	q := sqlcdb.New(db)
	self, err := q.ListAccountsByUserActive(ctx, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(self) != 0 {
		t.Fatalf("self accounts %d", len(self))
	}
	other, err := q.ListAccountsByUserActive(ctx, otherID)
	if err != nil {
		t.Fatal(err)
	}
	if len(other) != 1 || other[0].Name != "Чужой" {
		t.Fatalf("other accounts %+v", other)
	}
}
