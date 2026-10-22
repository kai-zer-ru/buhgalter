package notify

import (
	"strings"
	"testing"
	"time"

	"github.com/kai-zer-ru/buhgalter/internal/timeutil"
)

func TestFormatDefaultsForAllTriggers(t *testing.T) {
	const base = "https://buhgalter.example"
	data := FormatData{
		"debtor":          "Денис",
		"amount":          "10 000.00 ₽",
		"due_date":        "01.07.2026",
		"days":            "2",
		"action":          "вернуть долг",
		"credit":          "Ипотека",
		"payment_date":    "02.07.2026",
		"when":            "завтра",
		"type":            "Расход",
		"description":     "Подписка",
		"date":            "01.07.2026 09:00",
		"requested_at":    "01.07.2026 09:00",
		"channel":         "telegram",
		"debt_url":        debtURLPlaceholderValue(base, "ru", previewDebtID),
		"credit_url":      creditURLPlaceholderValue(base, "ru", previewCreditID),
		"transaction_url": transactionURLPlaceholderValue(base, "ru", previewTransactionID),
		"settings_url":    settingsURLPlaceholderValue(base, "ru"),
		"reset_url":       resetURLPlaceholderValue(base, "ru", previewResetUserID),
		"moderation_url":  moderationURLPlaceholderValue(base, "ru", previewResetUserID),
	}
	for _, trigger := range triggerOrder {
		text, err := Format(trigger, "ru", nil, data)
		if err != nil {
			t.Fatalf("trigger %s: %v", trigger, err)
		}
		if text == "" {
			t.Fatalf("trigger %s: empty text", trigger)
		}
	}
}

func TestValidateTemplateUnknownPlaceholder(t *testing.T) {
	err := ValidateTemplate(TriggerDebtOverdue, "Просрочен: {debtor} {oops}")
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestCustomTemplateOverride(t *testing.T) {
	custom := "Долг {debtor}: {amount}"
	text, err := Format(TriggerDebtOverdue, "ru", &custom, FormatData{
		"debtor": "Иван",
		"amount": "1 000.00 ₽",
	})
	if err != nil {
		t.Fatal(err)
	}
	if text != "Долг Иван: 1 000.00 ₽" {
		t.Fatalf("unexpected text: %q", text)
	}
}

func TestRelativeWhen(t *testing.T) {
	now := mustTime("2026-06-01 12:00:00")
	if got := RelativeWhen("ru", "2026-06-01 13:00:00", now, "Europe/Moscow"); got != "сегодня" {
		t.Fatalf("today: %q", got)
	}
	if got := RelativeWhen("ru", "2026-06-02 13:00:00", now, "Europe/Moscow"); got != "завтра" {
		t.Fatalf("tomorrow: %q", got)
	}
}

func TestRelativeDays(t *testing.T) {
	if got := RelativeDays("ru", 0); got != "сегодня" {
		t.Fatalf("ru today: %q", got)
	}
	if got := RelativeDays("en", 0); got != "today" {
		t.Fatalf("en today: %q", got)
	}
	if got := RelativeDays("ru", 1); got != "завтра" {
		t.Fatalf("ru tomorrow: %q", got)
	}
	if got := RelativeDays("en", 1); got != "tomorrow" {
		t.Fatalf("en tomorrow: %q", got)
	}
	if got := RelativeDays("ru", 3); got != "через 3 дн." {
		t.Fatalf("ru in 3: %q", got)
	}
	if got := RelativeDays("en", 3); got != "in 3 days" {
		t.Fatalf("en in 3: %q", got)
	}
}

func TestDebtActionPhrase(t *testing.T) {
	if got := DebtActionPhrase("ru", "borrowed"); got != "вернуть долг" {
		t.Fatalf("borrowed ru: %q", got)
	}
	if got := DebtActionPhrase("ru", "lent"); got != "получить долг от" {
		t.Fatalf("lent ru: %q", got)
	}
	if got := DebtActionPhrase("en", "borrowed"); got != "repay debt to" {
		t.Fatalf("borrowed en: %q", got)
	}
	if got := DebtActionPhrase("en", "lent"); got != "collect debt from" {
		t.Fatalf("lent en: %q", got)
	}
}

func TestDebtDueSoonDefaultWordingByDirection(t *testing.T) {
	const debtURL = "https://example/debts"

	lentToday, err := Format(TriggerDebtDueSoon, "ru", nil, FormatData{
		"debtor":   "Настя сестра",
		"amount":   "100.00 ₽",
		"due_date": "15.07.2026",
		"days":     "0",
		"when":     RelativeDays("ru", 0),
		"action":   DebtActionPhrase("ru", "lent"),
		"debt_url": debtURL,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(lentToday, "получить долг от Настя сестра") {
		t.Fatalf("lent wording missing: %q", lentToday)
	}
	if strings.Contains(lentToday, "вернуть") {
		t.Fatalf("lent must not say вернуть: %q", lentToday)
	}
	if !strings.Contains(lentToday, "сегодня") {
		t.Fatalf("expected сегодня: %q", lentToday)
	}
	if strings.Contains(lentToday, "через 0") {
		t.Fatalf("must not say через 0: %q", lentToday)
	}

	borrowedSoon, err := Format(TriggerDebtDueSoon, "ru", nil, FormatData{
		"debtor":   "Денис",
		"amount":   "1 000.00 ₽",
		"due_date": "17.07.2026",
		"days":     "2",
		"when":     RelativeDays("ru", 2),
		"action":   DebtActionPhrase("ru", "borrowed"),
		"debt_url": debtURL,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(borrowedSoon, "вернуть долг Денис") {
		t.Fatalf("borrowed wording missing: %q", borrowedSoon)
	}
	if !strings.Contains(borrowedSoon, "через 2 дн.") {
		t.Fatalf("expected через 2 дн.: %q", borrowedSoon)
	}
}

func mustTime(s string) (resultTime time.Time) {
	tm, err := timeutil.ParseUTC(s)
	if err != nil {
		panic(err)
	}
	return tm
}
