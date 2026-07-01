package notify

import (
	"net/url"
	"strings"
)

const previewResetUserID = "00000000-0000-0000-0000-000000000001"

func buildAdminResetURL(externalURL, userID string) string {
	base := trimExternalURL(externalURL)
	userID = strings.TrimSpace(userID)
	if base == "" || userID == "" {
		return ""
	}
	return base + "/admin/users?reset=" + url.QueryEscape(userID)
}

func resetURLPlaceholderValue(externalURL, localeCode, userID string) string {
	if link := buildAdminResetURL(externalURL, userID); link != "" {
		return link
	}
	return externalURLMissingHint(localeCode)
}
