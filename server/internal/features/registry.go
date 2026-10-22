package features

// Stable instance-level module keys. Do not rename after release.
const (
	Registration         = "registration"
	Debts                = "debts"
	Credits              = "credits"
	Budget               = "budget"
	Subscriptions        = "subscriptions"
	Recurring            = "recurring"
	BalanceMaintenance   = "balance_maintenance"
	ImportExport         = "import_export"
	Stats                = "stats"
	Notifications        = "notifications"
	MerchantsTags        = "merchants_tags"
	TransactionTemplates = "transaction_templates"
)

// Flag describes a known feature flag in the code registry.
type Flag struct {
	Key            string
	TitleKey       string
	DescriptionKey string
	DefaultEnabled bool
}

// Registry is the ordered catalog of known flags.
var Registry = []Flag{
	{
		Key:            Registration,
		TitleKey:       "features.registration.title",
		DescriptionKey: "features.registration.description",
		DefaultEnabled: false,
	},
	{
		Key:            Debts,
		TitleKey:       "features.debts.title",
		DescriptionKey: "features.debts.description",
		DefaultEnabled: true,
	},
	{
		Key:            Credits,
		TitleKey:       "features.credits.title",
		DescriptionKey: "features.credits.description",
		DefaultEnabled: true,
	},
	{
		Key:            Budget,
		TitleKey:       "features.budget.title",
		DescriptionKey: "features.budget.description",
		DefaultEnabled: true,
	},
	{
		Key:            Subscriptions,
		TitleKey:       "features.subscriptions.title",
		DescriptionKey: "features.subscriptions.description",
		DefaultEnabled: true,
	},
	{
		Key:            Recurring,
		TitleKey:       "features.recurring.title",
		DescriptionKey: "features.recurring.description",
		DefaultEnabled: true,
	},
	{
		Key:            BalanceMaintenance,
		TitleKey:       "features.balance_maintenance.title",
		DescriptionKey: "features.balance_maintenance.description",
		DefaultEnabled: true,
	},
	{
		Key:            ImportExport,
		TitleKey:       "features.import_export.title",
		DescriptionKey: "features.import_export.description",
		DefaultEnabled: true,
	},
	{
		Key:            Stats,
		TitleKey:       "features.stats.title",
		DescriptionKey: "features.stats.description",
		DefaultEnabled: true,
	},
	{
		Key:            Notifications,
		TitleKey:       "features.notifications.title",
		DescriptionKey: "features.notifications.description",
		DefaultEnabled: true,
	},
	{
		Key:            MerchantsTags,
		TitleKey:       "features.merchants_tags.title",
		DescriptionKey: "features.merchants_tags.description",
		DefaultEnabled: true,
	},
	{
		Key:            TransactionTemplates,
		TitleKey:       "features.transaction_templates.title",
		DescriptionKey: "features.transaction_templates.description",
		DefaultEnabled: true,
	},
}

func knownKeys() map[string]Flag {
	out := make(map[string]Flag, len(Registry))
	for _, f := range Registry {
		out[f.Key] = f
	}
	return out
}

// IsKnown reports whether key is in the registry.
func IsKnown(key string) bool {
	_, ok := knownKeys()[key]
	return ok
}
