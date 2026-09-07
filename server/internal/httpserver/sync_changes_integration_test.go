package httpserver_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strconv"
	"testing"
)

func TestTransactionChangesFeed(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	accID := createTestAccount(t, env, "Кошелёк")
	catID := getExpenseCategory(t, env)
	past := "2020-01-15 12:00:00"

	emptyResp, err := env.authedRequest(http.MethodGet, "/api/v1/sync/transaction-changes?since_id=0", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer emptyResp.Body.Close()
	if emptyResp.StatusCode != http.StatusOK {
		t.Fatalf("empty feed status %d", emptyResp.StatusCode)
	}
	var empty struct {
		SinceID int64 `json:"since_id"`
		Changes []any `json:"changes"`
	}
	_ = json.NewDecoder(emptyResp.Body).Decode(&empty)
	if len(empty.Changes) != 0 {
		t.Fatalf("expected no changes, got %+v", empty.Changes)
	}

	body, _ := json.Marshal(map[string]any{
		"account_id": accID, "type": "expense", "amount": "15.00",
		"category_id": catID, "transaction_date": past,
	})
	createResp, err := env.authedRequest(http.MethodPost, "/api/v1/transactions", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer createResp.Body.Close()
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("create status %d", createResp.StatusCode)
	}
	var created struct {
		ID string `json:"id"`
	}
	_ = json.NewDecoder(createResp.Body).Decode(&created)

	feedResp, err := env.authedRequest(http.MethodGet, "/api/v1/sync/transaction-changes?since_id=0", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer feedResp.Body.Close()
	var feed struct {
		ServerTime string `json:"server_time"`
		SinceID    int64  `json:"since_id"`
		Changes    []struct {
			Action      string `json:"action"`
			EntityID    string `json:"entity_id"`
			Transaction *struct {
				ID     string `json:"id"`
				Amount int64  `json:"amount"`
			} `json:"transaction"`
		} `json:"changes"`
	}
	_ = json.NewDecoder(feedResp.Body).Decode(&feed)
	if len(feed.Changes) != 1 || feed.Changes[0].Action != "upsert" || feed.Changes[0].EntityID != created.ID {
		t.Fatalf("unexpected feed: %+v", feed)
	}
	if feed.Changes[0].Transaction == nil || feed.Changes[0].Transaction.Amount != 1500 {
		t.Fatalf("expected upsert payload 1500, got %+v", feed.Changes[0].Transaction)
	}

	delResp, err := env.authedRequest(http.MethodDelete, "/api/v1/transactions/"+created.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	delResp.Body.Close()
	if delResp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete status %d", delResp.StatusCode)
	}

	afterCreate := feed.SinceID
	delFeedResp, err := env.authedRequest(http.MethodGet, "/api/v1/sync/transaction-changes?since_id="+strconv.FormatInt(afterCreate, 10), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer delFeedResp.Body.Close()
	var delFeed struct {
		Changes []struct {
			Action   string `json:"action"`
			EntityID string `json:"entity_id"`
		} `json:"changes"`
	}
	_ = json.NewDecoder(delFeedResp.Body).Decode(&delFeed)
	if len(delFeed.Changes) != 1 || delFeed.Changes[0].Action != "deleted" || delFeed.Changes[0].EntityID != created.ID {
		t.Fatalf("expected delete event, got %+v", delFeed.Changes)
	}
}
