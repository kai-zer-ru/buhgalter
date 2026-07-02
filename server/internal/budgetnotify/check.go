package budgetnotify

import (
	"context"
	"database/sql"
	"strconv"
	"time"

	"github.com/kai-zer-ru/buhgalter/internal/budget"
	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
	"github.com/kai-zer-ru/buhgalter/internal/notify"
	"github.com/kai-zer-ru/buhgalter/internal/timeutil"
)

// CheckThresholdsAfterTx re-evaluates budget alert thresholds after an expense mutation.
func CheckThresholdsAfterTx(ctx context.Context, db *sql.DB, userID string) {
	_ = CheckThresholdsForUser(ctx, db, userID)
}

// CheckThresholdsForUser sends budget_threshold notifications for crossed thresholds.
func CheckThresholdsForUser(ctx context.Context, db *sql.DB, userID string) error {
	q := sqlcdb.New(db)
	if err := q.EnsureNotificationSettings(ctx, userID); err != nil {
		return err
	}
	settings, err := q.GetNotificationSettings(ctx, userID)
	if err != nil {
		return err
	}
	if settings.TriggerBudget != 1 {
		return nil
	}
	month, err := budget.CurrentMonthQuery(ctx, db, userID)
	if err != nil {
		return err
	}
	items, err := budget.Summary(ctx, db, userID, month)
	if err != nil {
		return err
	}
	periodStart, _, err := budget.MonthBounds(ctx, db, userID, month)
	if err != nil {
		return err
	}
	localeCode, _, currencyCode, err := notify.UserFormatting(ctx, db, userID)
	if err != nil {
		return err
	}
	externalURL := notify.ResolveExternalURL(ctx, db)
	customTemplates, err := q.ListNotificationTemplates(ctx, userID)
	if err != nil {
		return err
	}
	customMap := notify.ToTemplateMap(customTemplates)
	for _, item := range items.Items {
		for _, th := range alertThresholds(item.AlertAtPercent) {
			if !thresholdCrossed(item, th) {
				continue
			}
			sent, err := q.HasBudgetAlertSent(ctx, sqlcdb.HasBudgetAlertSentParams{
				BudgetID: item.ID, PeriodStart: periodStart, ThresholdPercent: th,
			})
			if err != nil || sent > 0 {
				continue
			}
			text, err := notify.Format(notify.TriggerBudgetThreshold, localeCode, customMap[notify.TriggerBudgetThreshold], notify.FormatData{
				"name":       item.Name,
				"spent":      notify.FormatAmountDisplay(item.Spent, currencyCode),
				"planned":    notify.FormatAmountDisplay(item.Planned, currencyCode),
				"percent":    strconv.Itoa(item.Percent),
				"budget_url": notify.BudgetURLPlaceholderValue(externalURL, localeCode),
			})
			if err != nil {
				continue
			}
			notify.Deliver(ctx, db, settings, userID, notify.TriggerBudgetThreshold, item.ID, periodStart, text)
			_ = q.InsertBudgetAlertSent(ctx, sqlcdb.InsertBudgetAlertSentParams{
				BudgetID: item.ID, PeriodStart: periodStart, ThresholdPercent: th,
				SentAt: timeutil.FormatUTC(time.Now().UTC()),
			})
		}
	}
	return nil
}

func alertThresholds(alertAt int64) []int64 {
	out := []int64{100}
	if alertAt > 0 && alertAt < 100 {
		out = append([]int64{alertAt}, out...)
	}
	return out
}

func thresholdCrossed(item budget.SummaryItem, threshold int64) bool {
	if threshold == 100 {
		return item.Spent > item.Planned
	}
	return int64(item.Percent) >= threshold
}
