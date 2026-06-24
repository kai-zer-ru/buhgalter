package notify

const (
	TriggerDebtOverdue   = "debt_overdue"
	TriggerDebtDueSoon   = "debt_due_soon"
	TriggerCreditPayment = "credit_payment"
	TriggerPlannedOp     = "planned_operation"
	TriggerTest          = "test"
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
	TriggerTest,
}

var triggerPlaceholders = map[string][]string{
	TriggerDebtOverdue:   {"debtor", "amount", "due_date"},
	TriggerDebtDueSoon:   {"debtor", "amount", "due_date", "days"},
	TriggerCreditPayment: {"credit", "amount", "payment_date", "when"},
	TriggerPlannedOp:     {"type", "amount", "description", "date"},
	TriggerTest:          {"channel"},
}
