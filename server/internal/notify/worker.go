package notify

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
	"github.com/kai-zer-ru/buhgalter/internal/timeutil"
)

type Worker struct {
	DB     *sql.DB
	Logger *slog.Logger

	stop    chan struct{}
	mu      sync.Mutex
	lastRun map[string]string
}

func NewWorker(db *sql.DB, logger *slog.Logger) *Worker {
	return &Worker{
		DB:      db,
		Logger:  logger,
		stop:    make(chan struct{}),
		lastRun: map[string]string{},
	}
}

func (w *Worker) Start() {
	go w.loop()
}

func (w *Worker) Stop() {
	close(w.stop)
}

func (w *Worker) loop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-w.stop:
			return
		case now := <-ticker.C:
			w.run(now)
		}
	}
}

func (w *Worker) run(now time.Time) {
	ctx := context.Background()
	q := sqlcdb.New(w.DB)
	users, err := q.ListUsersWithTimezone(ctx)
	if err != nil {
		w.Logger.Error("notify worker: list users", "err", err)
		return
	}
	for _, user := range users {
		if err := q.EnsureNotificationSettings(ctx, user.ID); err != nil {
			continue
		}
		settings, err := q.GetNotificationSettings(ctx, user.ID)
		if err != nil {
			continue
		}
		loc, err := time.LoadLocation(defaultTZ(user.Timezone))
		if err != nil {
			continue
		}
		localNow := now.In(loc)
		runHour, runMinute := parseNotificationSendTime(settings.NotificationTimeLocal)
		if localNow.Hour() != runHour || localNow.Minute() != runMinute {
			continue
		}
		dateKey := localNow.Format("2006-01-02")
		if !w.markRunOnce(user.ID, dateKey) {
			continue
		}
		if err := w.runForUser(ctx, user.ID, now, localNow, settings); err != nil {
			w.Logger.Error("notify worker: user run failed", "user_id", user.ID, "err", err)
		}
	}
}

func (w *Worker) markRunOnce(userID, dateKey string) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.lastRun[userID] == dateKey {
		return false
	}
	w.lastRun[userID] = dateKey
	return true
}

func (w *Worker) runForUser(ctx context.Context, userID string, nowUTC, nowLocal time.Time, settings sqlcdb.NotificationSetting) error {
	q := sqlcdb.New(w.DB)
	localeCode, timezone, currencyCode, err := userFormatting(ctx, w.DB, userID)
	if err != nil {
		return err
	}
	cutoff := nowLocal.In(time.UTC).Format("2006-01-02 15:04:05")
	if err := w.processPlanned(ctx, q, settings, userID, localeCode, timezone, currencyCode, cutoff, nowLocal); err != nil {
		return err
	}
	if err := w.processDebts(ctx, q, settings, userID, localeCode, timezone, currencyCode, nowLocal); err != nil {
		return err
	}
	if err := w.processCreditPayments(ctx, q, settings, userID, localeCode, timezone, currencyCode, nowUTC, nowLocal); err != nil {
		return err
	}
	return nil
}

func (w *Worker) processPlanned(
	ctx context.Context,
	q *sqlcdb.Queries,
	settings sqlcdb.NotificationSetting,
	userID, localeCode, timezone, currencyCode, cutoff string,
	nowLocal time.Time,
) error {
	rows, err := q.ListDueFutureTransactions(ctx, sqlcdb.ListDueFutureTransactionsParams{
		UserID:          userID,
		TransactionDate: cutoff,
	})
	if err != nil {
		return err
	}
	_, err = q.ActivateFutureTransactionsBefore(ctx, sqlcdb.ActivateFutureTransactionsBeforeParams{
		UpdatedAt:       time.Now().UTC().Format(time.RFC3339),
		UserID:          userID,
		TransactionDate: cutoff,
	})
	if err != nil {
		return err
	}
	if settings.TriggerPlanned != 1 {
		return nil
	}
	customTemplates, err := q.ListNotificationTemplates(ctx, userID)
	if err != nil {
		return err
	}
	customMap := toTemplateMap(customTemplates)
	for _, tx := range rows {
		dateKey := nowLocal.Format("2006-01-02")
		text, err := Format(TriggerPlannedOp, localeCode, customMap[TriggerPlannedOp], FormatData{
			"type":        localizedOperationType(localeCode, tx.Type),
			"amount":      FormatAmountDisplay(tx.Amount, currencyCode),
			"description": normalizeDescription(tx.Description),
			"date":        timeutil.FormatDisplayDateTimeShortInTimezone(tx.TransactionDate, timezone),
		})
		if err != nil {
			continue
		}
		w.sendByChannels(ctx, q, settings, userID, TriggerPlannedOp, tx.ID, dateKey, text)
	}
	return nil
}

func (w *Worker) processDebts(
	ctx context.Context,
	q *sqlcdb.Queries,
	settings sqlcdb.NotificationSetting,
	userID, localeCode, timezone, currencyCode string,
	nowLocal time.Time,
) error {
	if settings.TriggerDebt != 1 {
		return nil
	}
	rows, err := q.ListActiveDebtsByUser(ctx, userID)
	if err != nil {
		return err
	}
	customTemplates, err := q.ListNotificationTemplates(ctx, userID)
	if err != nil {
		return err
	}
	customMap := toTemplateMap(customTemplates)
	for _, row := range rows {
		diff := localDayDiff(nowLocal, row.DueDate, timezone)
		trigger, ok := pickDebtTrigger(settings, strings.TrimSpace(row.Direction), diff)
		if !ok {
			continue
		}
		text, err := Format(trigger, localeCode, customMap[trigger], FormatData{
			"debtor":   row.DebtorName,
			"amount":   FormatAmountDisplay(row.Amount, currencyCode),
			"due_date": timeutil.FormatDisplayDateInTimezone(row.DueDate, timezone),
			"days":     int64ToString(int64(max(diff, 0))),
		})
		if err != nil {
			continue
		}
		dateKey := nowLocal.Format("2006-01-02")
		w.sendByChannels(ctx, q, settings, userID, trigger, row.ID, dateKey, text)
	}
	return nil
}

func pickDebtTrigger(settings sqlcdb.NotificationSetting, direction string, dayDiff int) (string, bool) {
	switch direction {
	case "borrowed":
		// "Я должен": напоминания каждый день за N дней до срока, в день срока,
		// затем каждый день после срока, но не дольше лимита.
		if dayDiff >= 0 {
			if dayDiff <= int(settings.DebtDaysBefore) {
				return TriggerDebtDueSoon, true
			}
			return "", false
		}
		overdueDays := -dayDiff
		if settings.MyDebtOverdueDaysLimit <= 0 {
			return "", false
		}
		if overdueDays <= int(settings.MyDebtOverdueDaysLimit) {
			return TriggerDebtOverdue, true
		}
		return "", false
	case "lent":
		// "Мне должны": в день срока и далее ежедневные напоминания
		// после задержки в N дней, не дольше лимита.
		if dayDiff == 0 {
			return TriggerDebtDueSoon, true
		}
		if dayDiff > 0 {
			return "", false
		}
		overdueDays := -dayDiff
		startAfter := int(settings.OwedDebtOverdueStartAfterDays)
		if overdueDays <= startAfter {
			return "", false
		}
		if settings.OwedDebtOverdueDaysLimit <= 0 {
			return "", false
		}
		sentDays := overdueDays - startAfter
		if sentDays <= int(settings.OwedDebtOverdueDaysLimit) {
			return TriggerDebtOverdue, true
		}
		return "", false
	default:
		// На всякий случай для legacy-значений повторяем прежнее поведение.
		switch {
		case dayDiff < 0:
			return TriggerDebtOverdue, true
		case dayDiff == int(settings.DebtDaysBefore):
			return TriggerDebtDueSoon, true
		default:
			return "", false
		}
	}
}

func (w *Worker) processCreditPayments(
	ctx context.Context,
	q *sqlcdb.Queries,
	settings sqlcdb.NotificationSetting,
	userID, localeCode, timezone, currencyCode string,
	nowUTC, nowLocal time.Time,
) error {
	if settings.TriggerCredit != 1 {
		return nil
	}
	rows, err := q.CreditPaymentsUnappliedByUser(ctx, userID)
	if err != nil {
		return err
	}
	customTemplates, err := q.ListNotificationTemplates(ctx, userID)
	if err != nil {
		return err
	}
	customMap := toTemplateMap(customTemplates)
	for _, row := range rows {
		diff := localDayDiff(nowLocal, row.PaymentDate, timezone)
		if diff != 0 && diff != int(settings.CreditDaysBefore) {
			continue
		}
		creditName := "Кредит"
		if row.CreditName != nil && *row.CreditName != "" {
			creditName = *row.CreditName
		}
		text, err := Format(TriggerCreditPayment, localeCode, customMap[TriggerCreditPayment], FormatData{
			"credit":       creditName,
			"amount":       FormatAmountDisplay(row.Amount, currencyCode),
			"payment_date": timeutil.FormatDisplayDateInTimezone(row.PaymentDate, timezone),
			"when":         RelativeWhen(localeCode, row.PaymentDate, nowUTC, timezone),
		})
		if err != nil {
			continue
		}
		dateKey := nowLocal.Format("2006-01-02")
		w.sendByChannels(ctx, q, settings, userID, TriggerCreditPayment, row.ID, dateKey, text)
	}
	return nil
}

func (w *Worker) sendByChannels(ctx context.Context, q *sqlcdb.Queries, settings sqlcdb.NotificationSetting, userID, triggerType, entityID, dateKey, text string) {
	secret, err := ResolveSecretKey(ctx, w.DB)
	if err != nil {
		w.Logger.Warn("notify worker: cannot resolve secret key", "user_id", userID, "err", err)
		return
	}
	box, err := NewSecretBox(secret)
	if err != nil {
		w.Logger.Warn("notify worker: secret key is not configured", "user_id", userID, "err", err)
		return
	}
	channels := activeChannels(settings)
	for _, channel := range channels {
		exists, err := DedupExists(ctx, q, userID, triggerType, channel, entityID, dateKey)
		if err != nil || exists {
			continue
		}
		notifier, recipient, err := buildNotifier(settings, channel, box)
		if err != nil {
			_ = appendLog(ctx, q, userID, triggerType, channel, &entityID, &dateKey, "error", text)
			continue
		}
		if err := notifier.Send(ctx, recipient, text); err != nil {
			_ = appendLog(ctx, q, userID, triggerType, channel, &entityID, &dateKey, "error", text)
			continue
		}
		_ = appendLog(ctx, q, userID, triggerType, channel, &entityID, &dateKey, "sent", text)
	}
}

func activeChannels(settings sqlcdb.NotificationSetting) []string {
	var channels []string
	if settings.TelegramEnabled == 1 {
		channels = append(channels, ChannelTelegram)
	}
	if settings.MaxEnabled == 1 {
		channels = append(channels, ChannelMax)
	}
	return channels
}

func toTemplateMap(items []sqlcdb.NotificationTemplate) map[string]*string {
	out := make(map[string]*string, len(items))
	for _, item := range items {
		value := item.Template
		out[item.TriggerType] = &value
	}
	return out
}

func normalizeDescription(value *string) string {
	if value == nil || strings.TrimSpace(*value) == "" {
		return "—"
	}
	return strings.TrimSpace(*value)
}

func localDayDiff(nowLocal time.Time, targetUTC, timezone string) int {
	target, err := time.ParseInLocation("2006-01-02 15:04:05", targetUTC, time.UTC)
	if err != nil {
		return 9999
	}
	loc, err := time.LoadLocation(defaultTZ(timezone))
	if err != nil {
		loc = time.UTC
	}
	a := nowLocal.In(loc)
	b := target.In(loc)
	aa := time.Date(a.Year(), a.Month(), a.Day(), 0, 0, 0, 0, loc)
	bb := time.Date(b.Year(), b.Month(), b.Day(), 0, 0, 0, 0, loc)
	return int(bb.Sub(aa).Hours() / 24)
}

func int64ToString(value int64) string {
	return fmt.Sprintf("%d", value)
}

func defaultTZ(value string) string {
	if strings.TrimSpace(value) == "" {
		return "Europe/Moscow"
	}
	return value
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func parseNotificationSendTime(value string) (int, int) {
	normalized := normalizeNotificationTimeLocal(value)
	parsed, err := time.Parse("15:04", normalized)
	if err != nil {
		return 0, 0
	}
	return parsed.Hour(), parsed.Minute()
}
