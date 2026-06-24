package notify

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kai-zer-ru/buhgalter/internal/db"
	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
)

func TestDedupExistsCompositeKey(t *testing.T) {
	t.Parallel()

	manager, err := db.NewManager(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()

	ctx := context.Background()
	q := sqlcdb.New(manager.DB())
	userID := "u-dedup"
	now := time.Now().UTC().Format(time.RFC3339)

	_, err = manager.DB().ExecContext(ctx, `
		INSERT INTO users (id, login, password_hash, display_name, is_admin, language, currency, timezone, theme, created_at, updated_at)
		VALUES (?, 'dedup', 'hash', 'Dedup', 0, 'ru', 'RUB', 'Europe/Moscow', 'light', ?, ?)`,
		userID, now, now)
	if err != nil {
		t.Fatal(err)
	}

	entityID := "debt-1"
	dedupDate := "2026-06-24"
	message := "msg"
	if err := q.InsertNotificationLog(ctx, sqlcdb.InsertNotificationLogParams{
		ID:          uuid.NewString(),
		UserID:      userID,
		TriggerType: TriggerDebtOverdue,
		Channel:     ChannelTelegram,
		EntityID:    &entityID,
		DedupDate:   &dedupDate,
		Status:      "sent",
		Message:     &message,
		CreatedAt:   now,
	}); err != nil {
		t.Fatal(err)
	}

	exists, err := DedupExists(ctx, q, userID, TriggerDebtOverdue, ChannelTelegram, entityID, dedupDate)
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Fatal("expected dedup hit for exact composite key")
	}

	cases := []struct {
		name       string
		trigger    string
		channel    string
		entityID   string
		dedupDate  string
	}{
		{name: "different trigger", trigger: TriggerDebtDueSoon, channel: ChannelTelegram, entityID: entityID, dedupDate: dedupDate},
		{name: "different channel", trigger: TriggerDebtOverdue, channel: ChannelMax, entityID: entityID, dedupDate: dedupDate},
		{name: "different entity", trigger: TriggerDebtOverdue, channel: ChannelTelegram, entityID: "debt-2", dedupDate: dedupDate},
		{name: "different date", trigger: TriggerDebtOverdue, channel: ChannelTelegram, entityID: entityID, dedupDate: "2026-06-25"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			found, err := DedupExists(ctx, q, userID, tc.trigger, tc.channel, tc.entityID, tc.dedupDate)
			if err != nil {
				t.Fatal(err)
			}
			if found {
				t.Fatalf("expected no dedup hit for case %q", tc.name)
			}
		})
	}
}

