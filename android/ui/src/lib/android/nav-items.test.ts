import { describe, expect, it } from 'vitest';
import { androidAdminNavItems, androidMainNavItems, androidSettingsNavItems } from './nav-items';

describe('android nav items', () => {
	it('includes full main navigation like web', () => {
		const keys = androidMainNavItems().map((item) => item.labelKey);
		expect(keys).toEqual([
			'nav.home',
			'nav.accounts',
			'nav.transactions',
			'nav.debts',
			'nav.credits',
			'nav.budget',
			'nav.stats'
		]);
	});

	it('includes settings tabs with server URL and web settings', () => {
		const paths = androidSettingsNavItems().map((item) => item.href);
		expect(paths.some((href) => href.endsWith('/settings/profile'))).toBe(true);
		expect(paths.some((href) => href.endsWith('/settings/server'))).toBe(true);
		expect(paths.some((href) => href.endsWith('/settings/import'))).toBe(true);
		expect(paths.some((href) => href.endsWith('/settings/recurring-operations'))).toBe(true);
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
