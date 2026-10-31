import { resolve } from '$app/paths';
import { isFeatureEnabled } from '$lib/features';

export type AndroidNavItem = {
	href: string;
	labelKey: string;
	/** Hub row only (`SectionNavHub`): one-line description under the label. */
	hintKey?: string;
	/** Hub row only: SVG path `d` for a 24×24 stroke icon. */
	iconPath?: string;
	feature?: string;
	isActive: (pathname: string) => boolean;
};

function filterByFeature(items: AndroidNavItem[]): AndroidNavItem[] {
	return items.filter((item) => !item.feature || isFeatureEnabled(item.feature));
}

export function androidHomeNavItem(): AndroidNavItem {
	return {
		href: resolve('/'),
		labelKey: 'nav.home',
		isActive: (p) => p === '/'
	};
}

export function androidMainNavItems(): AndroidNavItem[] {
	return filterByFeature([
		androidHomeNavItem(),
		{
			href: resolve('/accounts'),
			labelKey: 'nav.accounts',
			isActive: (p) => p.startsWith('/accounts')
		},
		{
			href: resolve('/transactions'),
			labelKey: 'nav.transactions',
			isActive: (p) => p.startsWith('/transactions')
		},
		{
			href: resolve('/settings/bank-notifications/drafts'),
			labelKey: 'nav.bankNotificationDrafts',
			// Drafts + debug history are top-level (not Settings).
			isActive: (p) => isBankNotificationDraftsPath(p)
		},
		{
			href: resolve('/debts'),
			labelKey: 'nav.debts',
			feature: 'debts',
			isActive: (p) => p.startsWith('/debts') || p.startsWith('/debtors')
		},
		{
			href: resolve('/credits'),
			labelKey: 'nav.credits',
			feature: 'credits',
			isActive: (p) => p.startsWith('/credits')
		},
		{
			href: resolve('/subscriptions'),
			labelKey: 'nav.subscriptions',
			feature: 'subscriptions',
			isActive: (p) => p.startsWith('/subscriptions')
		},
		{
			href: resolve('/recurring-operations'),
			labelKey: 'nav.recurring',
			feature: 'recurring',
			isActive: (p) => p.startsWith('/recurring-operations')
		},
		{
			href: resolve('/transaction-templates'),
			labelKey: 'settings.tab.templates',
			feature: 'transaction_templates',
			isActive: (p) => p.startsWith('/transaction-templates')
		},
		{
			href: resolve('/category-rules'),
			labelKey: 'categoryRules.title',
			isActive: (p) => p.startsWith('/category-rules')
		},
		{
			href: resolve('/transactions/inbox'),
			labelKey: 'nav.inbox',
			isActive: (p) => p.startsWith('/transactions/inbox')
		},
		{
			href: resolve('/categories'),
			labelKey: 'nav.categories',
			isActive: (p) => p.startsWith('/categories')
		},
		{
			href: resolve('/merchants'),
			labelKey: 'nav.merchants',
			feature: 'merchants_tags',
			isActive: (p) => p.startsWith('/merchants')
		},
		{
			href: resolve('/tags'),
			labelKey: 'nav.tags',
			feature: 'merchants_tags',
			isActive: (p) => p.startsWith('/tags')
		},
		{
			href: resolve('/budget'),
			labelKey: 'nav.budget',
			feature: 'budget',
			isActive: (p) => p.startsWith('/budget')
		},
		{
			href: resolve('/stats'),
			labelKey: 'nav.stats',
			feature: 'stats',
			isActive: (p) => p.startsWith('/stats')
		}
	]);
}

export function androidSettingsNavItems(opts?: { householdLabelKey?: string }): AndroidNavItem[] {
	return filterByFeature([
		{
			href: resolve('/settings/profile'),
			labelKey: 'settings.tab.profile',
			hintKey: 'settings.hub.profile_hint',
			iconPath: 'M16 7a4 4 0 1 1-8 0 4 4 0 0 1 8 0ZM4 21v-1a6 6 0 0 1 6-6h4a6 6 0 0 1 6 6v1',
			isActive: (p) => p === '/settings/profile'
		},
		{
			href: resolve('/settings/password'),
			labelKey: 'settings.tab.password',
			hintKey: 'settings.hub.password_hint',
			iconPath: 'M7 11V7a5 5 0 0 1 10 0v4M5 11h14v10H5z',
			isActive: (p) => p === '/settings/password'
		},
		{
			href: resolve('/settings/security'),
			labelKey: 'settings.tab.security',
			hintKey: 'settings.hub.security_hint',
			iconPath: 'M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10Z',
			isActive: (p) => p === '/settings/security'
		},
		{
			href: resolve('/settings/server'),
			labelKey: 'settings.tab.server',
			hintKey: 'settings.hub.server_hint',
			iconPath: 'M4 4h16v6H4zM4 14h16v6H4zM8 7h.01M8 17h.01',
			isActive: (p) => p === '/settings/server'
		},
		{
			href: resolve('/settings/tokens'),
			labelKey: 'settings.tab.tokens',
			hintKey: 'settings.hub.tokens_hint',
			iconPath: 'M7.5 19a3.5 3.5 0 1 0 0-7 3.5 3.5 0 0 0 0 7ZM10 13l9-9M16 7l2 2M14 9l2 2',
			isActive: (p) => p === '/settings/tokens'
		},
		{
			href: resolve('/settings/household'),
			labelKey: opts?.householdLabelKey ?? 'settings.tab.household',
			hintKey: 'settings.hub.household_hint',
			iconPath:
				'M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2M9 11a4 4 0 1 0 0-8 4 4 0 0 0 0 8ZM22 21v-2a4 4 0 0 0-3-3.87M16 3.13a4 4 0 0 1 0 7.75',
			feature: 'household',
			isActive: (p) => p.startsWith('/settings/household')
		},
		{
			href: resolve('/settings/bank-notifications'),
			labelKey: 'settings.tab.bankNotifications',
			hintKey: 'settings.hub.bankNotifications_hint',
			iconPath: 'M6 8a6 6 0 0 1 12 0c0 7 3 9 3 9H3s3-2 3-9M10.3 21a1.94 1.94 0 0 0 3.4 0',
			isActive: (p) =>
				p.startsWith('/settings/bank-notifications') && !isBankNotificationDraftsPath(p)
		},
		{
			href: resolve('/settings/import'),
			labelKey: 'settings.tab.import',
			hintKey: 'settings.hub.import_hint',
			iconPath: 'M12 3v12M7 10l5 5 5-5M5 21h14',
			feature: 'import_export',
			isActive: (p) => p === '/settings/import'
		}
	]);
}

export function androidAdminNavItems(): AndroidNavItem[] {
	return [
		{
			href: resolve('/admin/system'),
			labelKey: 'admin.tab.system',
			hintKey: 'admin.hub.system_hint',
			iconPath: 'M4 21v-7M4 10V3M12 21v-9M12 8V3M20 21v-5M20 12V3M1 14h6M9 8h6M17 16h6',
			isActive: (p) => p === '/admin/system'
		},
		{
			href: resolve('/admin/backups'),
			labelKey: 'admin.tab.backups',
			hintKey: 'admin.hub.backups_hint',
			iconPath:
				'M3 5c0-1.7 4-3 9-3s9 1.3 9 3-4 3-9 3-9-1.3-9-3ZM3 5v14c0 1.7 4 3 9 3s9-1.3 9-3V5M3 12c0 1.7 4 3 9 3s9-1.3 9-3',
			isActive: (p) => p.startsWith('/admin/backups')
		},
		{
			href: resolve('/admin/diagnostics'),
			labelKey: 'admin.tab.diagnostics',
			hintKey: 'admin.hub.diagnostics_hint',
			iconPath: 'M22 12h-4l-3 9L9 3l-3 9H2',
			isActive: (p) => p.startsWith('/admin/diagnostics')
		}
	];
}

export function androidMainNavItemsAfterHome(): AndroidNavItem[] {
	return androidMainNavItems().slice(1);
}

/** Drafts / history live under /settings/... URL but are main-nav, not Settings. */
export function isBankNotificationDraftsPath(pathname: string): boolean {
	return (
		pathname.startsWith('/settings/bank-notifications/drafts') ||
		pathname.startsWith('/settings/bank-notifications/history')
	);
}

export function isAndroidSettingsGroupActive(pathname: string): boolean {
	if (isBankNotificationDraftsPath(pathname)) return false;
	return pathname === '/settings' || pathname.startsWith('/settings/');
}

export function isAndroidAdminGroupActive(pathname: string): boolean {
	return pathname === '/admin' || pathname.startsWith('/admin/');
}
