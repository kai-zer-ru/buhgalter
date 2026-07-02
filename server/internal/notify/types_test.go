package notify

import (
	"testing"

	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
)

func TestTemplateSettingEnabled(t *testing.T) {
	t.Parallel()

	allOn := sqlcdb.NotificationSetting{
		TriggerDebt:              1,
		TriggerCredit:            1,
		TriggerPlanned:           1,
		TriggerNegativeBalance:   1,
		TriggerBudget:            1,
		TriggerAutoTopupDisabled: 1,
		TriggerPasswordReset:     1,
		TriggerUserRegistration:  1,
	}
	allOff := sqlcdb.NotificationSetting{}

	cases := []struct {
		trigger string
		on      bool
		off     bool
	}{
		{TriggerDebtOverdue, true, false},
		{TriggerDebtDueSoon, true, false},
		{TriggerCreditPayment, true, false},
		{TriggerPlannedOp, true, false},
		{TriggerBalanceShortfall, true, false},
		{TriggerBudgetThreshold, true, false},
		{TriggerAutoTopupDisabled, true, false},
		{TriggerPasswordReset, true, false},
		{TriggerUserRegistration, true, false},
		{TriggerTest, true, true},
	}
	for _, tc := range cases {
		if TemplateSettingEnabled(allOn, tc.trigger) != tc.on {
			t.Fatalf("%s: expected enabled when settings on", tc.trigger)
		}
		if TemplateSettingEnabled(allOff, tc.trigger) != tc.off {
			t.Fatalf("%s: expected disabled when settings off", tc.trigger)
		}
	}
}

func TestPolicySettingEnabled(t *testing.T) {
	t.Parallel()

	if !PolicySettingEnabled(true, true, true, "debt_days_before") {
		t.Fatal("debt_days_before expected enabled when trigger_debt on")
	}
	if PolicySettingEnabled(false, true, true, "debt_days_before") {
		t.Fatal("debt_days_before expected disabled when trigger_debt off")
	}
	if !PolicySettingEnabled(false, true, false, "credit_days_before") {
		t.Fatal("credit_days_before expected enabled when trigger_credit on")
	}
	if PolicySettingEnabled(false, false, false, "notification_time_local") {
		t.Fatal("notification_time_local expected disabled when all scheduled triggers off")
	}
	if !PolicySettingEnabled(false, false, true, "notification_time_local") {
		t.Fatal("notification_time_local expected enabled when trigger_planned on")
	}
}
