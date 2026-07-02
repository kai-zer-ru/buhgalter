package notify

import "testing"

func TestExternalURLPlaceholderValue(t *testing.T) {
	t.Parallel()

	const base = "https://buhgalter.example"
	if got := debtURLPlaceholderValue(base, "ru", "debt-1"); got != base+"/debts" {
		t.Fatalf("debt url: %q", got)
	}
	if got := creditURLPlaceholderValue(base, "ru", "cr-1"); got != base+"/credits/cr-1" {
		t.Fatalf("credit url: %q", got)
	}
	if got := transactionURLPlaceholderValue(base, "en", "tx-1"); got != base+"/transactions" {
		t.Fatalf("transaction url: %q", got)
	}
	if got := settingsURLPlaceholderValue(base, "ru"); got != base+"/settings?tab=notifications" {
		t.Fatalf("settings url: %q", got)
	}
	if got := debtURLPlaceholderValue("", "ru", "debt-1"); got != externalURLMissingHint("ru") {
		t.Fatalf("missing external url: %q", got)
	}
}

func TestModerationURLPlaceholderValue(t *testing.T) {
	t.Parallel()

	const base = "https://buhgalter.example"
	userID := "00000000-0000-0000-0000-000000000099"
	want := base + "/admin/users?moderate=" + userID
	if got := moderationURLPlaceholderValue(base, "ru", userID); got != want {
		t.Fatalf("moderation url: %q", got)
	}
	if got := moderationURLPlaceholderValue("", "en", userID); got != "configure external URL in admin settings" {
		t.Fatalf("missing moderation hint: %q", got)
	}
}

func TestApplyPreviewURLs(t *testing.T) {
	t.Parallel()

	const base = "https://host.example"
	data := FormatData{}
	applyPreviewURLs(TriggerCreditPayment, data, base, "ru", "RUB")
	if data["credit_url"] != base+"/credits/"+previewCreditID {
		t.Fatalf("preview credit url: %q", data["credit_url"])
	}
}
