package notify

import (
	"net/url"
	"strings"
)

const previewResetUserID = "00000000-0000-0000-0000-000000000001"

func buildAdminResetURL(externalURL, userID string) string {
	base := strings.TrimRight(strings.TrimSpace(externalURL), "/")
	if base == "" || strings.TrimSpace(userID) == "" {
		return ""
	}
	return base + "/admin/users?reset=" + url.QueryEscape(userID)
}

func resetURLPlaceholderValue(externalURL, localeCode, userID string) string {
	if link := buildAdminResetURL(externalURL, userID); link != "" {
		return link
	}
	if normalizeLocale(localeCode) == "en" {
		return "No external link — configure the external URL in admin settings."
	}
	return "Нет внешней ссылки — настройте внешний URL в админке."
}
