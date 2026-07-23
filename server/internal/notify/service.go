package notify

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kai-zer-ru/buhgalter/internal/db"
	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
	"github.com/kai-zer-ru/buhgalter/internal/settingscache"
	"github.com/kai-zer-ru/buhgalter/internal/timeutil"
)

type TemplateView struct {
	TriggerType  string   `json:"trigger_type"`
	Template     string   `json:"template"`
	Placeholders []string `json:"placeholders"`
	IsCustom     bool     `json:"is_custom"`
}

type SettingsView struct {
	SecretKeyConfigured           bool           `json:"secret_key_configured"`
	TelegramEnabled               bool           `json:"telegram_enabled"`
	TelegramConfigured            bool           `json:"telegram_configured"`
	TelegramChatID                *string        `json:"telegram_chat_id,omitempty"`
	MaxEnabled                    bool           `json:"max_enabled"`
	MaxConfigured                 bool           `json:"max_configured"`
	MaxProvider                   *string        `json:"max_provider,omitempty"`
	MaxUserID                     *int64         `json:"max_user_id,omitempty"`
	MaxRecipientID                *int64         `json:"max_recipient_id,omitempty"`
	TriggerDebt                   bool           `json:"trigger_debt"`
	TriggerCredit                 bool           `json:"trigger_credit"`
	TriggerPlanned                bool           `json:"trigger_planned"`
	TriggerNegativeBalance        bool           `json:"trigger_negative_balance"`
	TriggerBudget                 bool           `json:"trigger_budget"`
	TriggerAutoTopupDisabled      bool           `json:"trigger_auto_topup_disabled"`
	TriggerUserRegistration       bool           `json:"trigger_user_registration"`
	TriggerPasswordReset          bool           `json:"trigger_password_reset"`
	DebtDaysBefore                int64          `json:"debt_days_before"`
	MyDebtOverdueDaysLimit        int64          `json:"my_debt_overdue_days_limit"`
	OwedDebtOverdueStartAfterDays int64          `json:"owed_debt_overdue_start_after_days"`
	OwedDebtOverdueDaysLimit      int64          `json:"owed_debt_overdue_days_limit"`
	CreditDaysBefore              int64          `json:"credit_days_before"`
	NotificationTimeLocal         string         `json:"notification_time_local"`
	Templates                     []TemplateView `json:"templates"`
}

type TemplateUpdate struct {
	TriggerType string `json:"trigger_type"`
	Template    string `json:"template"`
}

type UpdateSettingsInput struct {
	TelegramEnabled               *bool            `json:"telegram_enabled,omitempty"`
	TelegramBotToken              *string          `json:"telegram_bot_token,omitempty"`
	TelegramChatID                *string          `json:"telegram_chat_id,omitempty"`
	MaxEnabled                    *bool            `json:"max_enabled,omitempty"`
	MaxProvider                   *string          `json:"max_provider,omitempty"`
	MaxToken                      *string          `json:"max_token,omitempty"`
	MaxUserID                     *int64           `json:"max_user_id,omitempty"`
	MaxRecipientID                *int64           `json:"max_recipient_id,omitempty"`
	TriggerDebt                   *bool            `json:"trigger_debt,omitempty"`
	TriggerCredit                 *bool            `json:"trigger_credit,omitempty"`
	TriggerPlanned                *bool            `json:"trigger_planned,omitempty"`
	TriggerNegativeBalance        *bool            `json:"trigger_negative_balance,omitempty"`
	TriggerBudget                 *bool            `json:"trigger_budget,omitempty"`
	TriggerAutoTopupDisabled      *bool            `json:"trigger_auto_topup_disabled,omitempty"`
	TriggerUserRegistration       *bool            `json:"trigger_user_registration,omitempty"`
	TriggerPasswordReset          *bool            `json:"trigger_password_reset,omitempty"`
	DebtDaysBefore                *int64           `json:"debt_days_before,omitempty"`
	MyDebtOverdueDaysLimit        *int64           `json:"my_debt_overdue_days_limit,omitempty"`
	OwedDebtOverdueStartAfterDays *int64           `json:"owed_debt_overdue_start_after_days,omitempty"`
	OwedDebtOverdueDaysLimit      *int64           `json:"owed_debt_overdue_days_limit,omitempty"`
	CreditDaysBefore              *int64           `json:"credit_days_before,omitempty"`
	NotificationTimeLocal         *string          `json:"notification_time_local,omitempty"`
	Templates                     []TemplateUpdate `json:"templates,omitempty"`
}

func GetSettings(ctx context.Context, sqlDB *sql.DB, userID string) (SettingsView, error) {
	var view SettingsView
	err := db.WithBusyRetry(ctx, 6, func() error {
		var err error
		view, err = getSettingsOnce(ctx, sqlDB, userID)
		return err
	})
	return view, err
}

func getSettingsOnce(ctx context.Context, sqlDB *sql.DB, userID string) (SettingsView, error) {
	q := sqlcdb.New(sqlDB)
	if err := q.EnsureNotificationSettings(ctx, userID); err != nil {
		return SettingsView{}, err
	}
	settings, err := q.GetNotificationSettings(ctx, userID)
	if err != nil {
		return SettingsView{}, err
	}
	templates, err := q.ListNotificationTemplates(ctx, userID)
	if err != nil {
		return SettingsView{}, err
	}
	language, err := userLanguage(ctx, sqlDB, userID)
	if err != nil {
		return SettingsView{}, err
	}
	isAdmin, err := userIsAdmin(ctx, sqlDB, userID)
	if err != nil {
		return SettingsView{}, err
	}
	regEnabled, err := registrationEnabled(ctx, sqlDB)
	if err != nil {
		return SettingsView{}, err
	}
	custom := make(map[string]string, len(templates))
	for _, tpl := range templates {
		custom[tpl.TriggerType] = tpl.Template
	}
	view := SettingsView{
		SecretKeyConfigured:           SecretKeyConfigured(ctx, sqlDB),
		TelegramEnabled:               settings.TelegramEnabled == 1,
		TelegramConfigured:            strings.TrimSpace(derefStr(settings.TelegramBotToken)) != "" && strings.TrimSpace(derefStr(settings.TelegramChatID)) != "",
		TelegramChatID:                settings.TelegramChatID,
		MaxEnabled:                    settings.MaxEnabled == 1,
		MaxConfigured:                 strings.TrimSpace(derefStr(settings.MaxToken)) != "" && (settings.MaxUserID != nil || settings.MaxRecipientID != nil),
		MaxProvider:                   settings.MaxProvider,
		MaxUserID:                     settings.MaxUserID,
		MaxRecipientID:                settings.MaxRecipientID,
		TriggerDebt:                   settings.TriggerDebt == 1,
		TriggerCredit:                 settings.TriggerCredit == 1,
		TriggerPlanned:                settings.TriggerPlanned == 1,
		TriggerNegativeBalance:        settings.TriggerNegativeBalance == 1,
		TriggerBudget:                 settings.TriggerBudget == 1,
		TriggerAutoTopupDisabled:      settings.TriggerAutoTopupDisabled == 1,
		TriggerUserRegistration:       isAdmin && regEnabled && settings.TriggerUserRegistration == 1,
		TriggerPasswordReset:          isAdmin && settings.TriggerPasswordReset == 1,
		DebtDaysBefore:                settings.DebtDaysBefore,
		MyDebtOverdueDaysLimit:        settings.MyDebtOverdueDaysLimit,
		OwedDebtOverdueStartAfterDays: settings.OwedDebtOverdueStartAfterDays,
		OwedDebtOverdueDaysLimit:      settings.OwedDebtOverdueDaysLimit,
		CreditDaysBefore:              settings.CreditDaysBefore,
		NotificationTimeLocal:         normalizeNotificationTimeLocal(settings.NotificationTimeLocal),
		Templates:                     make([]TemplateView, 0, len(triggerOrder)),
	}
	for _, trigger := range triggerOrder {
		if !isAdmin && IsAdminOnlyTrigger(trigger) {
			continue
		}
		if RequiresRegistrationEnabled(trigger) && !regEnabled {
			continue
		}
		customTemplate, ok := custom[trigger]
		view.Templates = append(view.Templates, TemplateView{
			TriggerType:  trigger,
			Template:     choose(ok, customTemplate, defaultTemplate(language, trigger)),
			Placeholders: AvailablePlaceholders(trigger),
			IsCustom:     ok,
		})
	}
	return view, nil
}

func UpdateSettings(ctx context.Context, db *sql.DB, userID string, in UpdateSettingsInput, box *SecretBox) (SettingsView, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return SettingsView{}, err
	}
	defer func() { _ = tx.Rollback() }()

	q := sqlcdb.New(tx)
	if err := q.EnsureNotificationSettings(ctx, userID); err != nil {
		return SettingsView{}, err
	}
	settings, err := q.GetNotificationSettings(ctx, userID)
	if err != nil {
		return SettingsView{}, err
	}
	isAdmin, err := userIsAdmin(ctx, db, userID)
	if err != nil {
		return SettingsView{}, err
	}

	triggerDebt := settings.TriggerDebt == 1
	if in.TriggerDebt != nil {
		triggerDebt = *in.TriggerDebt
	}
	triggerCredit := settings.TriggerCredit == 1
	if in.TriggerCredit != nil {
		triggerCredit = *in.TriggerCredit
	}
	triggerPlanned := settings.TriggerPlanned == 1
	if in.TriggerPlanned != nil {
		triggerPlanned = *in.TriggerPlanned
	}
	if in.DebtDaysBefore != nil && !PolicySettingEnabled(triggerDebt, triggerCredit, triggerPlanned, "debt_days_before") {
		return SettingsView{}, fmt.Errorf("debt_days_before requires trigger_debt to be enabled")
	}
	if in.MyDebtOverdueDaysLimit != nil && !PolicySettingEnabled(triggerDebt, triggerCredit, triggerPlanned, "my_debt_overdue_days_limit") {
		return SettingsView{}, fmt.Errorf("my_debt_overdue_days_limit requires trigger_debt to be enabled")
	}
	if in.OwedDebtOverdueStartAfterDays != nil && !PolicySettingEnabled(triggerDebt, triggerCredit, triggerPlanned, "owed_debt_overdue_start_after_days") {
		return SettingsView{}, fmt.Errorf("owed_debt_overdue_start_after_days requires trigger_debt to be enabled")
	}
	if in.OwedDebtOverdueDaysLimit != nil && !PolicySettingEnabled(triggerDebt, triggerCredit, triggerPlanned, "owed_debt_overdue_days_limit") {
		return SettingsView{}, fmt.Errorf("owed_debt_overdue_days_limit requires trigger_debt to be enabled")
	}
	if in.CreditDaysBefore != nil && !PolicySettingEnabled(triggerDebt, triggerCredit, triggerPlanned, "credit_days_before") {
		return SettingsView{}, fmt.Errorf("credit_days_before requires trigger_credit to be enabled")
	}
	if in.NotificationTimeLocal != nil && !PolicySettingEnabled(triggerDebt, triggerCredit, triggerPlanned, "notification_time_local") {
		return SettingsView{}, fmt.Errorf("notification_time_local requires at least one scheduled trigger (debt, credit, or planned) to be enabled")
	}

	if in.MaxProvider != nil {
		provider := strings.TrimSpace(*in.MaxProvider)
		if provider != "" && provider != MaxProviderA161 && provider != MaxProviderOfficial {
			return SettingsView{}, fmt.Errorf("max_provider must be one of: a161, official")
		}
		settings.MaxProvider = strPtrOrNil(provider)
	}
	if in.DebtDaysBefore != nil {
		if *in.DebtDaysBefore < 0 || *in.DebtDaysBefore > 30 {
			return SettingsView{}, fmt.Errorf("debt_days_before must be in range 0..30")
		}
		settings.DebtDaysBefore = *in.DebtDaysBefore
	}
	if in.CreditDaysBefore != nil {
		if *in.CreditDaysBefore < 0 || *in.CreditDaysBefore > 30 {
			return SettingsView{}, fmt.Errorf("credit_days_before must be in range 0..30")
		}
		settings.CreditDaysBefore = *in.CreditDaysBefore
	}
	if in.MyDebtOverdueDaysLimit != nil {
		if *in.MyDebtOverdueDaysLimit < 0 || *in.MyDebtOverdueDaysLimit > 365 {
			return SettingsView{}, fmt.Errorf("my_debt_overdue_days_limit must be in range 0..365")
		}
		settings.MyDebtOverdueDaysLimit = *in.MyDebtOverdueDaysLimit
	}
	if in.OwedDebtOverdueStartAfterDays != nil {
		if *in.OwedDebtOverdueStartAfterDays < 0 || *in.OwedDebtOverdueStartAfterDays > 365 {
			return SettingsView{}, fmt.Errorf("owed_debt_overdue_start_after_days must be in range 0..365")
		}
		settings.OwedDebtOverdueStartAfterDays = *in.OwedDebtOverdueStartAfterDays
	}
	if in.OwedDebtOverdueDaysLimit != nil {
		if *in.OwedDebtOverdueDaysLimit < 0 || *in.OwedDebtOverdueDaysLimit > 365 {
			return SettingsView{}, fmt.Errorf("owed_debt_overdue_days_limit must be in range 0..365")
		}
		settings.OwedDebtOverdueDaysLimit = *in.OwedDebtOverdueDaysLimit
	}
	if in.NotificationTimeLocal != nil {
		normalized, err := validateNotificationTimeLocal(*in.NotificationTimeLocal)
		if err != nil {
			return SettingsView{}, err
		}
		settings.NotificationTimeLocal = normalized
	}
	if in.TelegramEnabled != nil {
		settings.TelegramEnabled = boolToInt(*in.TelegramEnabled)
	}
	if in.MaxEnabled != nil {
		settings.MaxEnabled = boolToInt(*in.MaxEnabled)
	}
	if in.TriggerDebt != nil {
		settings.TriggerDebt = boolToInt(*in.TriggerDebt)
	}
	if in.TriggerCredit != nil {
		settings.TriggerCredit = boolToInt(*in.TriggerCredit)
	}
	if in.TriggerPlanned != nil {
		settings.TriggerPlanned = boolToInt(*in.TriggerPlanned)
	}
	if in.TriggerNegativeBalance != nil {
		settings.TriggerNegativeBalance = boolToInt(*in.TriggerNegativeBalance)
	}
	if in.TriggerBudget != nil {
		settings.TriggerBudget = boolToInt(*in.TriggerBudget)
	}
	if in.TriggerAutoTopupDisabled != nil {
		settings.TriggerAutoTopupDisabled = boolToInt(*in.TriggerAutoTopupDisabled)
	}
	if in.TriggerUserRegistration != nil {
		if !isAdmin {
			return SettingsView{}, fmt.Errorf("trigger_user_registration is admin-only")
		}
		regEnabled, err := registrationEnabled(ctx, db)
		if err != nil {
			return SettingsView{}, err
		}
		if !regEnabled {
			return SettingsView{}, fmt.Errorf("trigger_user_registration requires registration to be enabled")
		}
		settings.TriggerUserRegistration = boolToInt(*in.TriggerUserRegistration)
	}
	if in.TriggerPasswordReset != nil {
		if !isAdmin {
			return SettingsView{}, fmt.Errorf("trigger_password_reset is admin-only")
		}
		settings.TriggerPasswordReset = boolToInt(*in.TriggerPasswordReset)
	}
	if in.TelegramChatID != nil {
		settings.TelegramChatID = strPtrOrNil(strings.TrimSpace(*in.TelegramChatID))
	}
	if in.MaxUserID != nil {
		settings.MaxUserID = in.MaxUserID
	}
	if in.MaxRecipientID != nil {
		settings.MaxRecipientID = in.MaxRecipientID
	}
	if in.TelegramBotToken != nil {
		encrypted, err := upsertSecret(*in.TelegramBotToken, box)
		if err != nil {
			return SettingsView{}, err
		}
		settings.TelegramBotToken = encrypted
	}
	if in.MaxToken != nil {
		encrypted, err := upsertSecret(*in.MaxToken, box)
		if err != nil {
			return SettingsView{}, err
		}
		settings.MaxToken = encrypted
	}
	for _, tpl := range in.Templates {
		trigger := strings.TrimSpace(tpl.TriggerType)
		if _, ok := triggerPlaceholders[trigger]; !ok {
			return SettingsView{}, fmt.Errorf("unknown trigger_type: %s", trigger)
		}
		if !isAdmin && IsAdminOnlyTrigger(trigger) {
			return SettingsView{}, fmt.Errorf("unknown trigger_type: %s", trigger)
		}
		if RequiresRegistrationEnabled(trigger) {
			regEnabled, err := registrationEnabled(ctx, db)
			if err != nil {
				return SettingsView{}, err
			}
			if !regEnabled {
				return SettingsView{}, fmt.Errorf("unknown trigger_type: %s", trigger)
			}
		}
		if !TemplateSettingEnabled(settings, trigger) {
			return SettingsView{}, fmt.Errorf("template %s requires its notification setting to be enabled", trigger)
		}
		template := strings.TrimSpace(tpl.Template)
		if len(template) < 1 || len(template) > 500 {
			return SettingsView{}, fmt.Errorf("template length must be in range 1..500")
		}
		if err := ValidateTemplate(trigger, template); err != nil {
			return SettingsView{}, err
		}
		if err := q.UpsertNotificationTemplate(ctx, sqlcdb.UpsertNotificationTemplateParams{
			UserID:      userID,
			TriggerType: trigger,
			Template:    template,
			UpdatedAt:   time.Now().UTC().Format(time.RFC3339),
		}); err != nil {
			return SettingsView{}, err
		}
	}
	if err := q.UpsertNotificationSettings(ctx, sqlcdb.UpsertNotificationSettingsParams{
		UserID:                        userID,
		TelegramEnabled:               settings.TelegramEnabled,
		TelegramBotToken:              settings.TelegramBotToken,
		TelegramChatID:                settings.TelegramChatID,
		MaxEnabled:                    settings.MaxEnabled,
		MaxProvider:                   settings.MaxProvider,
		MaxToken:                      settings.MaxToken,
		MaxUserID:                     settings.MaxUserID,
		MaxRecipientID:                settings.MaxRecipientID,
		TriggerDebt:                   settings.TriggerDebt,
		TriggerCredit:                 settings.TriggerCredit,
		TriggerPlanned:                settings.TriggerPlanned,
		TriggerNegativeBalance:        settings.TriggerNegativeBalance,
		TriggerBudget:                 settings.TriggerBudget,
		TriggerAutoTopupDisabled:      settings.TriggerAutoTopupDisabled,
		TriggerUserRegistration:       settings.TriggerUserRegistration,
		TriggerPasswordReset:          settings.TriggerPasswordReset,
		DebtDaysBefore:                settings.DebtDaysBefore,
		MyDebtOverdueDaysLimit:        settings.MyDebtOverdueDaysLimit,
		OwedDebtOverdueStartAfterDays: settings.OwedDebtOverdueStartAfterDays,
		OwedDebtOverdueDaysLimit:      settings.OwedDebtOverdueDaysLimit,
		CreditDaysBefore:              settings.CreditDaysBefore,
		NotificationTimeLocal:         settings.NotificationTimeLocal,
		UpdatedAt:                     time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		return SettingsView{}, err
	}
	if err := tx.Commit(); err != nil {
		return SettingsView{}, err
	}
	return GetSettings(ctx, db, userID)
}

func PreviewTemplate(ctx context.Context, db *sql.DB, userID, triggerType, template string) (string, error) {
	triggerType = strings.TrimSpace(triggerType)
	template = strings.TrimSpace(template)
	if _, ok := triggerPlaceholders[triggerType]; !ok {
		return "", fmt.Errorf("unknown trigger_type: %s", triggerType)
	}
	if err := rejectRegistrationDependentTrigger(ctx, db, userID, triggerType); err != nil {
		return "", err
	}
	if err := rejectTemplateWhenSettingDisabled(ctx, db, userID, triggerType); err != nil {
		return "", err
	}
	if err := ValidateTemplate(triggerType, template); err != nil {
		return "", err
	}
	localeCode, timezone, currencyCode, err := userFormatting(ctx, db, userID)
	if err != nil {
		return "", err
	}
	externalURL, err := settingscache.ExternalURL(ctx, db)
	if err != nil {
		return "", err
	}
	urlValue := ""
	if externalURL.Valid {
		urlValue = externalURL.String
	}
	data := previewData(triggerType, localeCode, timezone, currencyCode, "", urlValue)
	text, err := Format(triggerType, localeCode, &template, data)
	if err != nil {
		return "", err
	}
	return text, nil
}

func ResetTemplates(ctx context.Context, db *sql.DB, userID string, triggerType *string) error {
	q := sqlcdb.New(db)
	if triggerType == nil || strings.TrimSpace(*triggerType) == "" {
		_, err := q.DeleteNotificationTemplatesByUser(ctx, userID)
		return err
	}
	trigger := strings.TrimSpace(*triggerType)
	if _, ok := triggerPlaceholders[trigger]; !ok {
		return fmt.Errorf("unknown trigger_type: %s", trigger)
	}
	if err := rejectRegistrationDependentTrigger(ctx, db, userID, trigger); err != nil {
		return err
	}
	_, err := q.DeleteNotificationTemplate(ctx, sqlcdb.DeleteNotificationTemplateParams{
		UserID:      userID,
		TriggerType: trigger,
	})
	return err
}

func SendTest(ctx context.Context, db *sql.DB, userID string, channel string, box *SecretBox) error {
	q := sqlcdb.New(db)
	if err := q.EnsureNotificationSettings(ctx, userID); err != nil {
		return err
	}
	settings, err := q.GetNotificationSettings(ctx, userID)
	if err != nil {
		return err
	}
	localeCode, timezone, currencyCode, err := userFormatting(ctx, db, userID)
	if err != nil {
		return err
	}
	templates, err := q.ListNotificationTemplates(ctx, userID)
	if err != nil {
		return err
	}
	customMap := make(map[string]string, len(templates))
	for _, tpl := range templates {
		customMap[tpl.TriggerType] = tpl.Template
	}
	var custom *string
	if value, ok := customMap[TriggerTest]; ok {
		custom = &value
	}
	externalURL, err := settingscache.ExternalURL(ctx, db)
	if err != nil {
		return err
	}
	urlValue := ""
	if externalURL.Valid {
		urlValue = externalURL.String
	}
	channelValue := strings.TrimSpace(channel)
	if channelValue == "" {
		channelValue = ChannelTelegram
	}
	text, err := Format(TriggerTest, localeCode, custom, previewData(TriggerTest, localeCode, timezone, currencyCode, channelValue, urlValue))
	if err != nil {
		return err
	}
	notifier, recipient, err := buildNotifier(settings, channel, box)
	if err != nil {
		_ = appendLog(ctx, q, userID, TriggerTest, channel, nil, nil, "error", text)
		return err
	}
	if err := notifier.Send(ctx, recipient, text); err != nil {
		_ = appendLog(ctx, q, userID, TriggerTest, channel, nil, nil, "error", text)
		return err
	}
	return appendLog(ctx, q, userID, TriggerTest, channel, nil, nil, "sent", text)
}

func NotifyAdminsOnPasswordReset(ctx context.Context, db *sql.DB, targetUserID, login, displayName, requestedAt, entityID string) error {
	q := sqlcdb.New(db)
	secret, err := ResolveSecretKey(ctx, db)
	if err != nil {
		return nil
	}
	box, err := NewSecretBox(secret)
	if err != nil {
		return nil
	}
	externalURL, err := settingscache.ExternalURL(ctx, db)
	if err != nil {
		return err
	}
	externalURLValue := ""
	if externalURL.Valid {
		externalURLValue = externalURL.String
	}
	adminIDs, err := q.ListAdminUserIDs(ctx)
	if err != nil {
		return err
	}
	for _, adminID := range adminIDs {
		if err := q.EnsureNotificationSettings(ctx, adminID); err != nil {
			continue
		}
		settings, err := q.GetNotificationSettings(ctx, adminID)
		if err != nil {
			continue
		}
		if settings.TriggerPasswordReset != 1 {
			continue
		}
		localeCode, timezone, currencyCode, err := userFormatting(ctx, db, adminID)
		if err != nil {
			continue
		}
		templates, err := q.ListNotificationTemplates(ctx, adminID)
		if err != nil {
			continue
		}
		customMap := toTemplateMap(templates)
		display := strings.TrimSpace(displayName)
		if display == "" {
			display = login
		}
		text, err := Format(TriggerPasswordReset, localeCode, customMap[TriggerPasswordReset], FormatData{
			"login":        login,
			"display_name": display,
			"requested_at": timeutil.FormatDisplayDateTimeShortInTimezone(requestedAt, timezone),
			"amount":       FormatAmountDisplay(0, currencyCode),
			"reset_url":    resetURLPlaceholderValue(externalURLValue, localeCode, targetUserID),
		})
		if err != nil {
			continue
		}
		now := time.Now()
		loc, err := time.LoadLocation(defaultTZ(timezone))
		if err != nil {
			loc = time.UTC
		}
		dateKey := now.In(loc).Format("2006-01-02")
		for _, channel := range activeChannels(settings) {
			exists, err := DedupExists(ctx, q, adminID, TriggerPasswordReset, channel, entityID, dateKey)
			if err != nil || exists {
				continue
			}
			notifier, recipient, err := buildNotifier(settings, channel, box)
			if err != nil {
				_ = appendLog(ctx, q, adminID, TriggerPasswordReset, channel, &entityID, &dateKey, "error", text)
				continue
			}
			if err := notifier.Send(ctx, recipient, text); err != nil {
				_ = appendLog(ctx, q, adminID, TriggerPasswordReset, channel, &entityID, &dateKey, "error", text)
				continue
			}
			_ = appendLog(ctx, q, adminID, TriggerPasswordReset, channel, &entityID, &dateKey, "sent", text)
		}
	}
	return nil
}

func appendLog(ctx context.Context, q *sqlcdb.Queries, userID, triggerType, channel string, entityID, dedupDate *string, status, message string) error {
	msg := message
	return q.InsertNotificationLog(ctx, sqlcdb.InsertNotificationLogParams{
		ID:          uuid.NewString(),
		UserID:      userID,
		TriggerType: triggerType,
		Channel:     channel,
		EntityID:    entityID,
		DedupDate:   dedupDate,
		Status:      status,
		Message:     &msg,
		CreatedAt:   time.Now().UTC().Format(time.RFC3339),
	})
}

func DedupExists(ctx context.Context, q *sqlcdb.Queries, userID, triggerType, channel string, entityID, dedupDate string) (bool, error) {
	count, err := q.ExistsNotificationDedup(ctx, sqlcdb.ExistsNotificationDedupParams{
		UserID:      userID,
		TriggerType: triggerType,
		Channel:     channel,
		EntityID:    strPtrOrNil(entityID),
		DedupDate:   strPtrOrNil(dedupDate),
	})
	return count > 0, err
}

func buildNotifier(settings sqlcdb.NotificationSetting, channel string, box *SecretBox) (Notifier, string, error) {
	switch strings.TrimSpace(channel) {
	case ChannelTelegram:
		token, err := decryptSecret(settings.TelegramBotToken, box)
		if err != nil {
			return nil, "", err
		}
		notifier := &TelegramNotifier{
			Token:  token,
			ChatID: derefStr(settings.TelegramChatID),
		}
		return notifier, derefStr(settings.TelegramChatID), notifier.ValidateConfig()
	case ChannelMax:
		token, err := decryptSecret(settings.MaxToken, box)
		if err != nil {
			return nil, "", err
		}
		provider := derefStr(settings.MaxProvider)
		if provider == MaxProviderOfficial {
			recipient := intToString(settings.MaxRecipientID)
			notifier := &MaxOfficialNotifier{
				Token:       token,
				RecipientID: recipient,
			}
			return notifier, recipient, notifier.ValidateConfig()
		}
		recipient := intToString(settings.MaxUserID)
		notifier := &MaxA161Notifier{
			Token:  token,
			UserID: recipient,
		}
		return notifier, recipient, notifier.ValidateConfig()
	default:
		return nil, "", errors.New("unsupported channel")
	}
}

func upsertSecret(raw string, box *SecretBox) (*string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	if box == nil {
		return nil, errors.New("encryption is not configured")
	}
	encrypted, err := box.Encrypt(raw)
	if err != nil {
		return nil, err
	}
	return &encrypted, nil
}

func decryptSecret(raw *string, box *SecretBox) (string, error) {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return "", nil
	}
	if box == nil {
		return "", errors.New("decryption is not configured")
	}
	return box.Decrypt(*raw)
}

func userLanguage(ctx context.Context, db *sql.DB, userID string) (string, error) {
	language, err := sqlcdb.New(db).GetUserLanguage(ctx, userID)
	if err != nil {
		return "", err
	}
	return normalizeLocale(language), nil
}

func userIsAdmin(ctx context.Context, db *sql.DB, userID string) (bool, error) {
	isAdmin, err := sqlcdb.New(db).GetUserIsAdmin(ctx, userID)
	if err != nil {
		return false, err
	}
	return isAdmin == 1, nil
}

func registrationEnabled(ctx context.Context, db *sql.DB) (bool, error) {
	enabled, err := sqlcdb.New(db).GetRegistrationEnabled(ctx)
	if err != nil {
		return false, err
	}
	return enabled == 1, nil
}

func rejectTemplateWhenSettingDisabled(ctx context.Context, db *sql.DB, userID, triggerType string) error {
	q := sqlcdb.New(db)
	if err := q.EnsureNotificationSettings(ctx, userID); err != nil {
		return err
	}
	settings, err := q.GetNotificationSettings(ctx, userID)
	if err != nil {
		return err
	}
	if TemplateSettingEnabled(settings, triggerType) {
		return nil
	}
	return fmt.Errorf("template %s requires its notification setting to be enabled", triggerType)
}

func rejectRegistrationDependentTrigger(ctx context.Context, db *sql.DB, userID, triggerType string) error {
	if !RequiresRegistrationEnabled(triggerType) {
		return nil
	}
	isAdmin, err := userIsAdmin(ctx, db, userID)
	if err != nil {
		return err
	}
	if !isAdmin {
		return fmt.Errorf("unknown trigger_type: %s", triggerType)
	}
	regEnabled, err := registrationEnabled(ctx, db)
	if err != nil {
		return err
	}
	if !regEnabled {
		return fmt.Errorf("unknown trigger_type: %s", triggerType)
	}
	return nil
}

func userFormatting(ctx context.Context, db *sql.DB, userID string) (localeCode, timezone, currencyCode string, err error) {
	row, err := sqlcdb.New(db).GetUserFormatting(ctx, userID)
	if err != nil {
		return "", "", "", err
	}
	localeCode = row.Language
	timezone = row.Timezone
	currencyCode = row.Currency
	if strings.TrimSpace(timezone) == "" {
		timezone = "Europe/Moscow"
	}
	if strings.TrimSpace(currencyCode) == "" {
		currencyCode = "RUB"
	}
	return normalizeLocale(localeCode), timezone, currencyCode, nil
}

func previewData(triggerType, localeCode, timezone, currencyCode, channel, externalURL string) FormatData {
	now := time.Now().UTC()
	futureDate := now.Add(48 * time.Hour).Format("2006-01-02 15:04:05")
	channelValue := strings.TrimSpace(channel)
	if channelValue == "" {
		channelValue = ChannelTelegram
	}
	data := FormatData{
		"debtor":        choose(normalizeLocale(localeCode) == "ru", "Денис", "Denis"),
		"amount":        FormatAmountDisplay(1000000, currencyCode),
		"due_date":      timeutil.FormatDisplayDateInTimezone(now.Add(-24*time.Hour).Format(timeutil.Layout), timezone),
		"days":          "2",
		"action":        DebtActionPhrase(localeCode, "borrowed"),
		"credit":        choose(normalizeLocale(localeCode) == "ru", "Ипотека", "Mortgage"),
		"payment_date":  timeutil.FormatDisplayDateInTimezone(futureDate, timezone),
		"when":          RelativeWhen(localeCode, futureDate, now, timezone),
		"type":          localizedOperationType(localeCode, "expense"),
		"description":   choose(normalizeLocale(localeCode) == "ru", "Подписка", "Subscription"),
		"date":          timeutil.FormatDisplayDateTimeShortInTimezone(now.Format(timeutil.Layout), timezone),
		"login":         choose(normalizeLocale(localeCode) == "ru", "user1", "user1"),
		"display_name":  choose(normalizeLocale(localeCode) == "ru", "Пользователь", "User"),
		"requested_at":  timeutil.FormatDisplayDateTimeShortInTimezone(now.Format(timeutil.Layout), timezone),
		"registered_at": timeutil.FormatDisplayDateTimeShortInTimezone(now.Format(timeutil.Layout), timezone),
		"channel":       channelValue,
	}
	applyPreviewURLs(triggerType, data, externalURL, localeCode, currencyCode)
	return data
}

func applyPreviewURLs(triggerType string, data FormatData, externalURL, localeCode, currencyCode string) {
	switch triggerType {
	case TriggerDebtOverdue, TriggerDebtDueSoon:
		data["debt_url"] = debtURLPlaceholderValue(externalURL, localeCode, previewDebtID)
	case TriggerCreditPayment:
		data["credit_url"] = creditURLPlaceholderValue(externalURL, localeCode, previewCreditID)
	case TriggerPlannedOp:
		data["transaction_url"] = transactionURLPlaceholderValue(externalURL, localeCode, previewTransactionID)
	case TriggerBudgetThreshold:
		data["name"] = choose(normalizeLocale(localeCode) == "ru", "Продукты", "Groceries")
		data["spent"] = FormatAmountDisplay(240000, currencyCode)
		data["planned"] = FormatAmountDisplay(300000, currencyCode)
		data["percent"] = "80"
		data["budget_url"] = budgetURLPlaceholderValue(externalURL, localeCode)
	case TriggerAutoTopupDisabled:
		data["account"] = choose(normalizeLocale(localeCode) == "ru", "Яндекс", "Yandex")
		data["source_account"] = choose(normalizeLocale(localeCode) == "ru", "Сбер", "Sber")
		data["amount"] = FormatAmountDisplay(250000, currencyCode)
		data["source_balance"] = FormatAmountDisplay(100000, currencyCode)
		data["account_url"] = accountURLPlaceholderValue(externalURL, localeCode, previewAccountID)
	case TriggerTest:
		data["settings_url"] = settingsURLPlaceholderValue(externalURL, localeCode)
	case TriggerPasswordReset:
		data["reset_url"] = resetURLPlaceholderValue(externalURL, localeCode, previewResetUserID)
	case TriggerUserRegistration:
		data["moderation_url"] = moderationURLPlaceholderValue(externalURL, localeCode, previewResetUserID)
	}
}

func boolToInt(value bool) int64 {
	if value {
		return 1
	}
	return 0
}

func derefStr(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func intToString(value *int64) string {
	if value == nil {
		return ""
	}
	return fmt.Sprintf("%d", *value)
}

func strPtrOrNil(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func choose[T any](ok bool, a T, b T) T {
	if ok {
		return a
	}
	return b
}

func normalizeNotificationTimeLocal(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "00:00"
	}
	return value
}

func validateNotificationTimeLocal(value string) (string, error) {
	candidate := normalizeNotificationTimeLocal(value)
	if _, err := time.Parse("15:04", candidate); err != nil {
		return "", fmt.Errorf("notification_time_local must be in HH:MM format")
	}
	return candidate, nil
}
