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
		TriggerDebtOverdue:       "Просрочен долг: {debtor} — {amount}. (Срок: {due_date})\n{debt_url}",
		TriggerDebtDueSoon:       "Напоминание: {action} {debtor} — {amount}. (Срок: {due_date}, {when})\n{debt_url}",
		TriggerCreditPayment:     "Платёж по кредиту «{credit}»: {amount}. Дата: {when}\n{credit_url}",
		TriggerPlannedOp:         "Плановая операция: {type} на {amount} — {description}\n{transaction_url}",
		TriggerBalanceShortfall:  "На балансе не хватает {amount}!",
		TriggerBudgetThreshold:   "{name}: потрачено {spent} из {planned} ({percent}%)\n{budget_url}",
		TriggerAutoTopupDisabled: "Автопополнение счёта «{account}» отключено: на «{source_account}» не хватает {amount} (остаток {source_balance}).\n{account_url}",
		TriggerUserRegistration:  "Новая регистрация: {display_name} (@{login}), {registered_at}. Модерация: {moderation_url}",
		TriggerPasswordReset:     "Запрос на восстановление пароля: {display_name} (@{login}), время: {requested_at}.\n{reset_url}",
		TriggerTest:              "Тестовое уведомление «Бухгалтер». Канал: {channel}.\n{settings_url}",
	},
	"en": {
		TriggerDebtOverdue:       "Debt overdue: {debtor} — {amount} (due {due_date})\n{debt_url}",
		TriggerDebtDueSoon:       "Reminder: {action} {debtor} — {amount} (due {due_date}, {when})\n{debt_url}",
		TriggerCreditPayment:     "Credit payment \"{credit}\": {amount} {when}\n{credit_url}",
		TriggerPlannedOp:         "Planned transaction: {type} {amount} — {description}\n{transaction_url}",
		TriggerBalanceShortfall:  "Insufficient balance: {amount} short!",
		TriggerBudgetThreshold:   "{name}: spent {spent} of {planned} ({percent}%)\n{budget_url}",
		TriggerAutoTopupDisabled: "Auto top-up for \"{account}\" disabled: \"{source_account}\" is short by {amount} (balance {source_balance}).\n{account_url}",
		TriggerUserRegistration:  "New registration: {display_name} (@{login}), {registered_at}. Moderation: {moderation_url}",
		TriggerPasswordReset:     "Password reset requested by {display_name} (@{login}) at {requested_at}.\n{reset_url}",
		TriggerTest:              "Buhgalter test notification. Channel: {channel}.\n{settings_url}",
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
