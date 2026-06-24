package notify

import "testing"

func TestDefaultTemplateAndNormalizeLocale(t *testing.T) {
	tpl := defaultTemplate("ru", TriggerDebtDueSoon)
	if tpl == "" {
		t.Fatal("expected ru template")
	}
	tplEn := defaultTemplate("en", TriggerCreditPayment)
	if tplEn == "" {
		t.Fatal("expected en template")
	}
	if got := normalizeLocale("en"); got != "en" {
		t.Fatalf("locale %q", got)
	}
	if got := localizedOperationType("ru", "expense"); got == "" {
		t.Fatal("expected localized type")
	}
}

func TestLocaleFileCandidates(t *testing.T) {
	paths := localeFileCandidates("ru")
	if len(paths) == 0 {
		t.Fatal("expected locale paths")
	}
	// loadLocaleTemplate falls back to empty when locale files are absent in test cwd
	_ = loadLocaleTemplate("ru", TriggerDebtDueSoon)
}

func TestNormalizeDescription(t *testing.T) {
	if got := normalizeDescription(nil); got != "—" {
		t.Fatalf("nil desc %q", got)
	}
	s := "  note  "
	if got := normalizeDescription(&s); got != "note" {
		t.Fatalf("trim %q", got)
	}
}
