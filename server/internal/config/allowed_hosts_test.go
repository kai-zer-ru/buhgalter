package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseHostList(t *testing.T) {
	got := ParseHostList(" 129.1.2.3 , buhgalter.example.com , ")
	want := []string{"129.1.2.3", "buhgalter.example.com"}
	if len(got) != len(want) {
		t.Fatalf("ParseHostList = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ParseHostList[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestParseHostListJSON(t *testing.T) {
	got := ParseHostList(`["129.1.2.3","buhgalter.example.com"]`)
	want := []string{"129.1.2.3", "buhgalter.example.com"}
	if len(got) != len(want) {
		t.Fatalf("ParseHostList JSON = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ParseHostList JSON[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestLoadDotEnvDoesNotOverrideExistingEnv(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")
	if err := os.WriteFile(envPath, []byte("BUHGALTER_ADDR=:9999\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("BUHGALTER_ADDR", ":8765")

	if err := LoadDotEnv(envPath); err != nil {
		t.Fatal(err)
	}
	if got := os.Getenv("BUHGALTER_ADDR"); got != ":8765" {
		t.Fatalf("BUHGALTER_ADDR = %q, want :8765", got)
	}
}

func TestLoadDotEnvReadsValues(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")
	content := "BUHGALTER_ADDR=:9999\nBUHGALTER_ALLOWED_HOSTS=[\"203.0.113.10\"]\n"
	if err := os.WriteFile(envPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("BUHGALTER_ADDR", "")
	t.Setenv("BUHGALTER_ALLOWED_HOSTS", "")

	if err := LoadDotEnv(envPath); err != nil {
		t.Fatal(err)
	}
	if got := os.Getenv("BUHGALTER_ADDR"); got != ":9999" {
		t.Fatalf("BUHGALTER_ADDR = %q, want :9999", got)
	}
	if got := ParseHostList(os.Getenv("BUHGALTER_ALLOWED_HOSTS")); len(got) != 1 || got[0] != "203.0.113.10" {
		t.Fatalf("BUHGALTER_ALLOWED_HOSTS = %v", got)
	}
}
