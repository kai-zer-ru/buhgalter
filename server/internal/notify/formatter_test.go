package notify

import (
	"testing"
	"time"

	"github.com/kai-zer-ru/buhgalter/internal/timeutil"
)

func TestFormatDefaultsForAllTriggers(t *testing.T) {
	data := FormatData{
		"debtor":       "Денис",
		"amount":       "10 000.00 ₽",
		"due_date":     "01.07.2026",
		"days":         "2",
		"credit":       "Ипотека",
		"payment_date": "02.07.2026",
		"when":         "завтра",
		"type":         "Расход",
		"description":  "Подписка",
		"date":         "01.07.2026 09:00",
		"requested_at": "01.07.2026 09:00",
		"channel":      "telegram",
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

func mustTime(s string) (resultTime time.Time) {
	tm, err := timeutil.ParseUTC(s)
	if err != nil {
		panic(err)
	}
	return tm
}
