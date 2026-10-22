package transaction

import (
	"context"
	"testing"
	"time"

	"github.com/kai-zer-ru/buhgalter/internal/timeutil"
)

func TestListChangesCreateUpdateDelete(t *testing.T) {
	handle, env := seedEnvFull(t)
	database := handle.DB()
	ctx := context.Background()
	past := timeutil.NowUTC().Add(-24 * time.Hour)

	empty, err := ListChanges(ctx, database, env.userID, 0, "", 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(empty.Changes) != 0 || empty.SinceID != 0 || empty.HasMore {
		t.Fatalf("expected empty feed, got %+v", empty)
	}

	tx, err := Create(ctx, database, env.userID, CreateInput{
		AccountID:       env.accountID,
		Type:            "expense",
		Amount:          2500,
		CategoryID:      &env.expenseID,
		TransactionDate: past,
	})
	if err != nil {
		t.Fatal(err)
	}

	created, err := ListChanges(ctx, database, env.userID, 0, "", 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(created.Changes) != 1 {
		t.Fatalf("expected 1 change, got %+v", created.Changes)
	}
	if created.Changes[0].Action != "upsert" || created.Changes[0].EntityID != tx.ID {
		t.Fatalf("unexpected create change: %+v", created.Changes[0])
	}
	if created.Changes[0].Transaction == nil || created.Changes[0].Transaction.Amount != 2500 {
		t.Fatalf("expected transaction payload, got %+v", created.Changes[0].Transaction)
	}

	if _, err := Update(ctx, database, env.userID, tx.ID, UpdateInput{
		AccountID:       env.accountID,
		Type:            "expense",
		Amount:          3000,
		CategoryID:      &env.expenseID,
		TransactionDate: past,
	}); err != nil {
		t.Fatal(err)
	}

	updated, err := ListChanges(ctx, database, env.userID, 0, "", 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(updated.Changes) != 1 {
		t.Fatalf("expected collapsed update, got %+v", updated.Changes)
	}
	if updated.Changes[0].Action != "upsert" || updated.Changes[0].Transaction.Amount != 3000 {
		t.Fatalf("expected latest amount 3000, got %+v", updated.Changes[0])
	}

	if err := Delete(ctx, database, env.userID, tx.ID); err != nil {
		t.Fatal(err)
	}

	deleted, err := ListChanges(ctx, database, env.userID, 0, "", 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(deleted.Changes) != 1 || deleted.Changes[0].Action != "deleted" {
		t.Fatalf("expected deleted, got %+v", deleted.Changes)
	}
	if deleted.Changes[0].Transaction != nil {
		t.Fatalf("deleted change must not include payload")
	}

	caughtUp, err := ListChanges(ctx, database, env.userID, deleted.SinceID, "", 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(caughtUp.Changes) != 0 {
		t.Fatalf("expected no further changes, got %+v", caughtUp.Changes)
	}
}

func TestListChangesIsolatesUsersAndPaginates(t *testing.T) {
	handle, env := seedEnvFull(t)
	database := handle.DB()
	ctx := context.Background()
	past := timeutil.NowUTC().Add(-24 * time.Hour)

	first, err := Create(ctx, database, env.userID, CreateInput{
		AccountID:       env.accountID,
		Type:            "expense",
		Amount:          1000,
		CategoryID:      &env.expenseID,
		TransactionDate: past,
	})
	if err != nil {
		t.Fatal(err)
	}
	second, err := Create(ctx, database, env.userID, CreateInput{
		AccountID:       env.accountID,
		Type:            "expense",
		Amount:          2000,
		CategoryID:      &env.expenseID,
		TransactionDate: past,
	})
	if err != nil {
		t.Fatal(err)
	}

	page1, err := ListChanges(ctx, database, env.userID, 0, "", 1)
	if err != nil {
		t.Fatal(err)
	}
	if !page1.HasMore || len(page1.Changes) != 1 || page1.Changes[0].EntityID != first.ID {
		t.Fatalf("page1: %+v", page1)
	}
	page2, err := ListChanges(ctx, database, env.userID, page1.SinceID, "", 10)
	if err != nil {
		t.Fatal(err)
	}
	if page2.HasMore || len(page2.Changes) != 1 || page2.Changes[0].EntityID != second.ID {
		t.Fatalf("page2: %+v", page2)
	}

	other, err := ListChanges(ctx, database, "missing-user", 0, "", 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(other.Changes) != 0 {
		t.Fatalf("other user must not see events, got %+v", other.Changes)
	}
}
