package notify

const (
	TriggerDebtOverdue      = "debt_overdue"
	TriggerDebtDueSoon      = "debt_due_soon"
	TriggerCreditPayment    = "credit_payment"
	TriggerPlannedOp        = "planned_operation"
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
	TriggerUserRegistration,
	TriggerPasswordReset,
	TriggerTest,
}

var triggerPlaceholders = map[string][]string{
	TriggerDebtOverdue:      {"debtor", "amount", "due_date"},
	TriggerDebtDueSoon:      {"debtor", "amount", "due_date", "days"},
	TriggerCreditPayment:    {"credit", "amount", "payment_date", "when"},
	TriggerPlannedOp:        {"type", "amount", "description", "date"},
	TriggerUserRegistration: {"login", "display_name", "registered_at", "moderation_url"},
	TriggerPasswordReset:    {"login", "display_name", "requested_at", "reset_url"},
	TriggerTest:             {"channel"},
}

func IsAdminOnlyTrigger(triggerType string) bool {
	return triggerType == TriggerPasswordReset || triggerType == TriggerUserRegistration
}

func RequiresRegistrationEnabled(triggerType string) bool {
	return triggerType == TriggerUserRegistration
}
