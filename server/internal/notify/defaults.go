package notify

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/kai-zer-ru/buhgalter/internal/locale"
)

type localeCatalog struct {
	Notifications struct {
		Templates map[string]string `json:"templates"`
	} `json:"notifications"`
}

var fallbackTemplates = map[string]map[string]string{
	"ru": {
		TriggerDebtOverdue:   "Просрочен долг: {debtor} — {amount}. (Срок: {due_date})",
		TriggerDebtDueSoon:   "Напоминание: вернуть долг {debtor} — {amount}. (Срок: {due_date}, через {days} дн.)",
		TriggerCreditPayment: "Платёж по кредиту «{credit}»: {amount}. Дата: {when}",
		TriggerPlannedOp:     "Плановая операция: {type} на {amount} — {description}",
		TriggerPasswordReset: "Запрос на восстановление пароля: {display_name} (@{login}), время: {requested_at}.",
		TriggerTest:          "Тестовое уведомление «Бухгалтер». Канал: {channel}.",
	},
	"en": {
		TriggerDebtOverdue:   "Debt overdue: {debtor} — {amount} (due {due_date})",
		TriggerDebtDueSoon:   "Reminder: repay debt to {debtor} — {amount} (due {due_date}, in {days} days)",
		TriggerCreditPayment: "Credit payment \"{credit}\": {amount} {when}",
		TriggerPlannedOp:     "Planned transaction: {type} {amount} — {description}",
		TriggerPasswordReset: "Password reset requested by {display_name} (@{login}) at {requested_at}.",
		TriggerTest:          "Buhgalter test notification. Channel: {channel}.",
	},
}

var operationTypeLocalized = map[string]map[string]string{
	"ru": {
		"expense":  "Расход",
		"income":   "Доход",
		"transfer": "Перевод",
	},
	"en": {
		"expense":  "Expense",
		"income":   "Income",
		"transfer": "Transfer",
	},
}

func defaultTemplate(localeCode, trigger string) string {
	localeCode = normalizeLocale(localeCode)
	if value := loadLocaleTemplate(localeCode, trigger); value != "" {
		return value
	}
	if value := fallbackTemplates[localeCode][trigger]; value != "" {
		return value
	}
	return fallbackTemplates["ru"][trigger]
}

func localizedOperationType(localeCode, txType string) string {
	localeCode = normalizeLocale(localeCode)
	if value := operationTypeLocalized[localeCode][txType]; value != "" {
		return value
	}
	return txType
}

func normalizeLocale(localeCode string) string {
	if localeCode == "en" {
		return "en"
	}
	return "ru"
}

func loadLocaleTemplate(localeCode, trigger string) string {
	for _, path := range localeFileCandidates(localeCode) {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var cat localeCatalog
		if err := json.Unmarshal(data, &cat); err != nil {
			var flat map[string]string
			if errFlat := json.Unmarshal(data, &flat); errFlat != nil {
				continue
			}
			if value := flat["notifications.templates."+trigger]; value != "" {
				return value
			}
			continue
		}
		if value := cat.Notifications.Templates[trigger]; value != "" {
			return value
		}
		var flat map[string]string
		if err := json.Unmarshal(data, &flat); err == nil {
			if value := flat["notifications.templates."+trigger]; value != "" {
				return value
			}
		}
	}
	return ""
}

func localeFileCandidates(localeCode string) []string {
	if dir := locale.Dir(); dir != "" {
		return []string{filepath.Join(dir, localeCode+".json")}
	}
	return []string{
		filepath.Join("locales", localeCode+".json"),
		filepath.Join("server", "locales", localeCode+".json"),
	}
}
