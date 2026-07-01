package notify

import (
	"strings"
	"testing"
)

func TestBuildAdminResetURL(t *testing.T) {
	userID := "abc-123"
	tests := []struct {
		name        string
		externalURL string
		want        string
	}{
		{
			name:        "plain url",
			externalURL: "https://buhgalter.example.com",
			want:        "https://buhgalter.example.com/admin/users?reset=abc-123",
		},
		{
			name:        "trailing slash",
			externalURL: "https://buhgalter.example.com/",
			want:        "https://buhgalter.example.com/admin/users?reset=abc-123",
		},
		{
			name:        "empty external url",
			externalURL: "",
			want:        "",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := buildAdminResetURL(tc.externalURL, userID); got != tc.want {
				t.Fatalf("buildAdminResetURL() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestResetURLPlaceholderValue(t *testing.T) {
	userID := "user-uuid"
	if got := resetURLPlaceholderValue("", "ru", userID); got != "Нет внешней ссылки — настройте внешний URL в админке." {
		t.Fatalf("ru hint = %q", got)
	}
	if got := resetURLPlaceholderValue("", "en", userID); got != "No external link — configure the external URL in admin settings." {
		t.Fatalf("en hint = %q", got)
	}
	want := "https://host.example/admin/users?reset=user-uuid"
	if got := resetURLPlaceholderValue("https://host.example", "ru", userID); got != want {
		t.Fatalf("url = %q, want %q", got, want)
	}
}

func TestPasswordResetTemplateIncludesResetURL(t *testing.T) {
	targetUserID := "target-user-id"
	externalURL := "https://buhgalter.example.com"
	template := defaultTemplate("ru", TriggerPasswordReset)
	text, err := Format(TriggerPasswordReset, "ru", &template, FormatData{
		"login":        "alice",
		"display_name": "Alice",
		"requested_at": "01.07.2026 12:00",
		"reset_url":    resetURLPlaceholderValue(externalURL, "ru", targetUserID),
	})
	if err != nil {
		t.Fatal(err)
	}
	wantLink := buildAdminResetURL(externalURL, targetUserID)
	if !strings.Contains(text, wantLink) {
		t.Fatalf("text %q should contain %q", text, wantLink)
	}
}
