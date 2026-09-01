import { describe, expect, it } from 'vitest';
import {
	androidAdminNavItems,
	androidMainNavItems,
	androidSettingsNavItems,
	isAndroidSettingsGroupActive,
	isBankNotificationDraftsPath
} from './nav-items';

describe('android nav items', () => {
	it('includes full main navigation like web', () => {
		const keys = androidMainNavItems().map((item) => item.labelKey);
		expect(keys).toEqual([
			'nav.home',
			'nav.accounts',
			'nav.transactions',
			'nav.bankNotificationDrafts',
			'nav.debts',
			'nav.credits',
			'nav.subscriptions',
			'nav.recurring',
			'settings.tab.templates',
			'nav.categories',
			'nav.merchants',
			'nav.tags',
			'nav.budget',
			'nav.stats'
		]);
	});

	it('includes transaction templates in main navigation', () => {
		const paths = androidMainNavItems().map((item) => item.href);
		expect(paths.some((href) => href.endsWith('/transaction-templates'))).toBe(true);
	});

	it('includes settings tabs with server URL and web settings', () => {
		const paths = androidSettingsNavItems().map((item) => item.href);
		expect(paths.some((href) => href.endsWith('/settings/profile'))).toBe(true);
		expect(paths.some((href) => href.endsWith('/settings/server'))).toBe(true);
		expect(paths.some((href) => href.endsWith('/settings/import'))).toBe(true);
		expect(paths.some((href) => href.endsWith('/settings/categories'))).toBe(false);
		expect(paths.some((href) => href.endsWith('/settings/recurring-operations'))).toBe(false);
		expect(paths.some((href) => href.endsWith('/settings/transaction-templates'))).toBe(false);
		expect(paths.some((href) => href.endsWith('/categories'))).toBe(false);
		expect(paths.some((href) => href.endsWith('/recurring-operations'))).toBe(false);
		expect(paths.some((href) => href.endsWith('/transaction-templates'))).toBe(false);
	});

	it('does not mark Settings active on bank drafts/history', () => {
		expect(isBankNotificationDraftsPath('/settings/bank-notifications/drafts')).toBe(true);
		expect(isBankNotificationDraftsPath('/settings/bank-notifications/history')).toBe(true);
		expect(isBankNotificationDraftsPath('/settings/bank-notifications')).toBe(false);
		expect(isAndroidSettingsGroupActive('/settings/bank-notifications/drafts')).toBe(false);
		expect(isAndroidSettingsGroupActive('/settings/bank-notifications/history')).toBe(false);
		expect(isAndroidSettingsGroupActive('/settings/bank-notifications')).toBe(true);

		const drafts = androidMainNavItems().find((i) => i.labelKey === 'nav.bankNotificationDrafts')!;
		expect(drafts.isActive('/settings/bank-notifications/drafts')).toBe(true);
		expect(drafts.isActive('/settings/bank-notifications/history')).toBe(true);
		expect(drafts.isActive('/settings/bank-notifications')).toBe(false);

		const bankSettings = androidSettingsNavItems().find((i) =>
			i.href.endsWith('/settings/bank-notifications')
		)!;
		expect(bankSettings.isActive('/settings/bank-notifications')).toBe(true);
		expect(bankSettings.isActive('/settings/bank-notifications/drafts')).toBe(false);
	});

	it('includes admin navigation with system route', () => {
		const items = androidAdminNavItems();
		const keys = items.map((item) => item.labelKey);
		const paths = items.map((item) => item.href);
		expect(keys).toContain('admin.tab.system');
		expect(keys).not.toContain('admin.tab.users');
		expect(paths.some((href) => href.endsWith('/admin/system'))).toBe(true);
	});
});
