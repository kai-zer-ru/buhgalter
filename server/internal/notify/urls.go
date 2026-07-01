package notify

import (
	"net/url"
	"strings"
)

const (
	previewDebtID        = "00000000-0000-0000-0000-000000000002"
	previewCreditID      = "00000000-0000-0000-0000-000000000003"
	previewTransactionID = "00000000-0000-0000-0000-000000000004"
)

func trimExternalURL(externalURL string) string {
	return strings.TrimRight(strings.TrimSpace(externalURL), "/")
}

func buildExternalPath(externalURL, path string) string {
	base := trimExternalURL(externalURL)
	path = strings.TrimSpace(path)
	if base == "" || path == "" {
		return ""
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return base + path
}

func externalURLMissingHint(localeCode string) string {
	if normalizeLocale(localeCode) == "en" {
		return "No external link — configure the external URL in admin settings."
	}
	return "Нет внешней ссылки — настройте внешний URL в админке."
}

func externalURLPlaceholderValue(externalURL, localeCode, path string) string {
	if link := buildExternalPath(externalURL, path); link != "" {
		return link
	}
	return externalURLMissingHint(localeCode)
}

func debtPath(debtID string) string {
	_ = debtID
	return "/debts"
}

func creditPath(creditID string) string {
	creditID = strings.TrimSpace(creditID)
	if creditID == "" {
		return ""
	}
	return "/credits/" + url.PathEscape(creditID)
}

func transactionPath(transactionID string) string {
	_ = transactionID
	return "/transactions"
}

func settingsNotificationsPath() string {
	return "/settings?tab=notifications"
}

func debtURLPlaceholderValue(externalURL, localeCode, debtID string) string {
	return externalURLPlaceholderValue(externalURL, localeCode, debtPath(debtID))
}

func creditURLPlaceholderValue(externalURL, localeCode, creditID string) string {
	return externalURLPlaceholderValue(externalURL, localeCode, creditPath(creditID))
}

func transactionURLPlaceholderValue(externalURL, localeCode, transactionID string) string {
	return externalURLPlaceholderValue(externalURL, localeCode, transactionPath(transactionID))
}

func settingsURLPlaceholderValue(externalURL, localeCode string) string {
	return externalURLPlaceholderValue(externalURL, localeCode, settingsNotificationsPath())
}

func moderationURLPlaceholderValue(externalURL, localeCode, userID string) string {
	if link := buildAdminModerationURL(externalURL, userID); link != "" {
		return link
	}
	if normalizeLocale(localeCode) == "en" {
		return "configure external URL in admin settings"
	}
	return "настройте внешний URL в админке"
}

func buildAdminModerationURL(externalURL, userID string) string {
	base := trimExternalURL(externalURL)
	userID = strings.TrimSpace(userID)
	if base == "" || userID == "" {
		return ""
	}
	return base + "/admin/users?moderate=" + url.QueryEscape(userID)
}
