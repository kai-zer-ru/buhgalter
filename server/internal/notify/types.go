package notify

import sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"

const (
	TriggerDebtOverdue      = "debt_overdue"
	TriggerDebtDueSoon      = "debt_due_soon"
	TriggerCreditPayment    = "credit_payment"
	TriggerPlannedOp        = "planned_operation"
	TriggerBalanceShortfall = "balance_shortfall"
	TriggerUserRegistration = "user_registration"
	TriggerPasswordReset    = "password_reset"
	TriggerTest             = "test"
)

const (
	ChannelTelegram = "telegram"
	ChannelMax      = "max"
)

const (
	MaxProviderA161     = "a161"
	MaxProviderOfficial = "official"
)

var triggerOrder = []string{
	TriggerDebtOverdue,
	TriggerDebtDueSoon,
	TriggerCreditPayment,
	TriggerPlannedOp,
	TriggerBalanceShortfall,
	TriggerUserRegistration,
	TriggerPasswordReset,
	TriggerTest,
}

var triggerPlaceholders = map[string][]string{
	TriggerDebtOverdue:      {"debtor", "amount", "due_date", "debt_url"},
	TriggerDebtDueSoon:      {"debtor", "amount", "due_date", "days", "debt_url"},
	TriggerCreditPayment:    {"credit", "amount", "payment_date", "when", "credit_url"},
	TriggerPlannedOp:        {"type", "amount", "description", "date", "transaction_url"},
	TriggerBalanceShortfall: {"amount"},
	TriggerUserRegistration: {"login", "display_name", "registered_at", "moderation_url"},
	TriggerPasswordReset:    {"login", "display_name", "requested_at", "reset_url"},
	TriggerTest:             {"channel", "settings_url"},
}

func IsAdminOnlyTrigger(triggerType string) bool {
	return triggerType == TriggerPasswordReset || triggerType == TriggerUserRegistration
}

func RequiresRegistrationEnabled(triggerType string) bool {
	return triggerType == TriggerUserRegistration
}

func TemplateSettingEnabled(settings sqlcdb.NotificationSetting, triggerType string) bool {
	switch triggerType {
	case TriggerDebtOverdue, TriggerDebtDueSoon:
		return settings.TriggerDebt == 1
	case TriggerCreditPayment:
		return settings.TriggerCredit == 1
	case TriggerPlannedOp:
		return settings.TriggerPlanned == 1
	case TriggerBalanceShortfall:
		return settings.TriggerNegativeBalance == 1
	case TriggerPasswordReset:
		return settings.TriggerPasswordReset == 1
	case TriggerUserRegistration:
		return settings.TriggerUserRegistration == 1
	default:
		return true
	}
}

func PolicySettingEnabled(triggerDebt, triggerCredit, triggerPlanned bool, field string) bool {
	switch field {
	case "debt_days_before", "my_debt_overdue_days_limit", "owed_debt_overdue_start_after_days", "owed_debt_overdue_days_limit":
		return triggerDebt
	case "credit_days_before":
		return triggerCredit
	case "notification_time_local":
		return triggerDebt || triggerCredit || triggerPlanned
	default:
		return true
	}
}
