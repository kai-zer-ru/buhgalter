package transaction

import (
	"context"
	"testing"
	"time"

	"github.com/kai-zer-ru/buhgalter/internal/store"
)

func TestCreate_merchantDoesNotCreateStore(t *testing.T) {
	handle, env := seedEnvFull(t)
	database := handle.DB()
	ctx := context.Background()
	name := "Пятёрочка"
	tx, err := Create(ctx, database, env.userID, CreateInput{
		AccountID:       env.accountID,
		Type:            "expense",
		Amount:          2500,
		CategoryID:      &env.expenseID,
		MerchantName:    &name,
		TransactionDate: time.Now().UTC().Add(-time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	if tx.MerchantID == nil || tx.StoreID != nil {
		t.Fatalf("expected merchant without store, got merchant %v store %v", tx.MerchantID, tx.StoreID)
	}
	stores, err := store.List(ctx, database, env.userID)
	if err != nil {
		t.Fatal(err)
	}
	if len(stores) != 0 {
		t.Fatalf("saving an expense created a store: %+v", stores)
	}
}
