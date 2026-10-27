package household

import "testing"

func TestNormalizeKind(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   string
		want string
		err  bool
	}{
		{in: "", want: KindFamily},
		{in: "family", want: KindFamily},
		{in: " Family ", want: KindFamily},
		{in: "team", want: KindTeam},
		{in: "TEAM", want: KindTeam},
		{in: "clan", err: true},
	}
	for _, tc := range cases {
		got, err := NormalizeKind(tc.in)
		if tc.err {
			if err == nil {
				t.Fatalf("%q: want error", tc.in)
			}
			continue
		}
		if err != nil {
			t.Fatalf("%q: %v", tc.in, err)
		}
		if got != tc.want {
			t.Fatalf("%q: got %q want %q", tc.in, got, tc.want)
		}
	}
}

func TestDefaultHouseholdName(t *testing.T) {
	t.Parallel()
	if got := defaultHouseholdName("Макс", "ru", KindFamily); got != "Семья Макс" {
		t.Fatalf("family ru: %q", got)
	}
	if got := defaultHouseholdName("Макс", "ru", KindTeam); got != "Команда Макс" {
		t.Fatalf("team ru: %q", got)
	}
	if got := defaultHouseholdName("Max", "en", KindFamily); got != "Max's family" {
		t.Fatalf("family en: %q", got)
	}
	if got := defaultHouseholdName("Max", "en", KindTeam); got != "Max's team" {
		t.Fatalf("team en: %q", got)
	}
}
