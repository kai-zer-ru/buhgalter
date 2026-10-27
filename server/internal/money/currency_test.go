package money

import "testing"

func TestCurrenciesAllowlist(t *testing.T) {
	want := []Currency{
		{"RUB", "₽"}, {"USD", "$"}, {"EUR", "€"}, {"CNY", "¥"},
		{"KZT", "₸"}, {"BYN", "Br"}, {"TRY", "₺"}, {"GBP", "£"},
	}
	got := Currencies()
	if len(got) != len(want) {
		t.Fatalf("len %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
	got[0].Code = "XXX"
	if Currencies()[0].Code != "RUB" {
		t.Fatal("Currencies must return a copy")
	}
}

func TestCurrencyValidSymbolNormalize(t *testing.T) {
	for _, code := range []string{"RUB", "USD", "EUR", "CNY", "KZT", "BYN", "TRY", "GBP"} {
		if !Valid(code) {
			t.Fatalf("%s must be valid", code)
		}
	}
	for _, code := range []string{"", "XYZ", "usd", "BTC"} {
		if Valid(code) {
			t.Fatalf("%q must be invalid", code)
		}
	}
	if Symbol("USD") != "$" || Symbol("BYN") != "Br" {
		t.Fatal("unexpected symbol")
	}
	if Symbol("XYZ") != "XYZ" {
		t.Fatal("unknown code must be returned as is")
	}
	if Normalize(" usd ") != "USD" {
		t.Fatal("normalize")
	}
}

func TestOrderWithProfileFirst(t *testing.T) {
	got := OrderWithProfileFirst("RUB", []string{"USD", "eur", "USD", ""})
	want := []string{"RUB", "EUR", "USD"}
	if len(got) != len(want) {
		t.Fatalf("%v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%v want %v", got, want)
		}
	}
	if got := OrderWithProfileFirst("", nil); len(got) != 1 || got[0] != DefaultCurrency {
		t.Fatalf("empty profile: %v", got)
	}
}

func TestOrderPresentWithProfileFirst(t *testing.T) {
	got := OrderPresentWithProfileFirst("RUB", []string{"USD", "EUR"})
	if len(got) != 2 || got[0] != "EUR" || got[1] != "USD" {
		t.Fatalf("profile absent: %v", got)
	}
	got = OrderPresentWithProfileFirst("RUB", []string{"USD", "RUB"})
	if len(got) != 2 || got[0] != "RUB" || got[1] != "USD" {
		t.Fatalf("profile present: %v", got)
	}
}

func TestParseQueryCurrency(t *testing.T) {
	got, err := ParseQueryCurrency("", "rub")
	if err != nil || got != "RUB" {
		t.Fatalf("default %q %v", got, err)
	}
	got, err = ParseQueryCurrency("usd", "RUB")
	if err != nil || got != "USD" {
		t.Fatalf("explicit %q %v", got, err)
	}
	if _, err := ParseQueryCurrency("XYZ", "RUB"); err != ErrInvalidCurrency {
		t.Fatalf("want ErrInvalidCurrency, got %v", err)
	}
}
