import { resolve } from '$app/paths';

export type AndroidNavItem = {
	href: string;
	labelKey: string;
	isActive: (pathname: string) => boolean;
};

export function androidHomeNavItem(): AndroidNavItem {
	return {
		href: resolve('/'),
		labelKey: 'nav.home',
		isActive: (p) => p === '/'
	};
}

export function androidMainNavItems(): AndroidNavItem[] {
	return [
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
			isActive: (p) => p.startsWith('/debts') || p.startsWith('/debtors')
		},
		{
			href: resolve('/credits'),
			labelKey: 'nav.credits',
			isActive: (p) => p.startsWith('/credits')
		},
		{
			href: resolve('/subscriptions'),
			labelKey: 'nav.subscriptions',
			isActive: (p) => p.startsWith('/subscriptions')
		},
		{
			href: resolve('/recurring-operations'),
			labelKey: 'nav.recurring',
			isActive: (p) => p.startsWith('/recurring-operations')
		},
		{
			href: resolve('/categories'),
			labelKey: 'nav.categories',
			isActive: (p) => p.startsWith('/categories')
		},
		{
			href: resolve('/merchants'),
			labelKey: 'nav.merchants',
			isActive: (p) => p.startsWith('/merchants')
		},
		{
			href: resolve('/tags'),
			labelKey: 'nav.tags',
			isActive: (p) => p.startsWith('/tags')
		},
		{
			href: resolve('/budget'),
			labelKey: 'nav.budget',
			isActive: (p) => p.startsWith('/budget')
		},
		{
			href: resolve('/stats'),
			labelKey: 'nav.stats',
			isActive: (p) => p.startsWith('/stats')
		}
	];
}

export function androidSettingsNavItems(): AndroidNavItem[] {
	return [
		{
			href: resolve('/settings/profile'),
			labelKey: 'settings.tab.profile',
			isActive: (p) => p === '/settings/profile'
		},
		{
			href: resolve('/settings/password'),
			labelKey: 'settings.tab.password',
			isActive: (p) => p === '/settings/password'
		},
		{
			href: resolve('/settings/security'),
			labelKey: 'settings.tab.security',
			isActive: (p) => p === '/settings/security'
		},
		{
			href: resolve('/settings/server'),
			labelKey: 'settings.tab.server',
			isActive: (p) => p === '/settings/server'
		},
		{
			href: resolve('/settings/tokens'),
			labelKey: 'settings.tab.tokens',
			isActive: (p) => p === '/settings/tokens'
		},
		{
			href: resolve('/settings/notifications'),
			labelKey: 'settings.tab.notifications',
			isActive: (p) => p === '/settings/notifications'
		},
		{
			href: resolve('/settings/bank-notifications'),
			labelKey: 'settings.tab.bankNotifications',
			isActive: (p) =>
				p.startsWith('/settings/bank-notifications') && !isBankNotificationDraftsPath(p)
		},
		{
			href: resolve('/settings/import'),
			labelKey: 'settings.tab.import',
			isActive: (p) => p === '/settings/import'
		}
	];
}

export function androidAdminNavItems(): AndroidNavItem[] {
	return [
		{
			href: resolve('/admin/system'),
			labelKey: 'admin.tab.system',
			isActive: (p) => p === '/admin/system'
		},
		{
			href: resolve('/admin/backups'),
			labelKey: 'admin.tab.backups',
			isActive: (p) => p.startsWith('/admin/backups')
		},
		{
			href: resolve('/admin/diagnostics'),
			labelKey: 'admin.tab.diagnostics',
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
