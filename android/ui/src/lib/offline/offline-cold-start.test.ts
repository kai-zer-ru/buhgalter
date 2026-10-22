import { beforeEach, describe, expect, it, vi } from 'vitest';

vi.mock('svelte-i18n', () => ({
	locale: { subscribe: () => () => {} }
}));

let currentServerUrl = 'http://new.lan:8765';

vi.mock('$lib/platform/server-url', () => ({
	getApiBase: () => currentServerUrl,
	getServerUrl: () => currentServerUrl
}));

vi.mock('$lib/platform/server-profile', () => ({
	getServerProfile: () => ({
		lanUrl: 'http://new.lan:8765',
		remoteUrl: '',
		homeSsids: [],
		lanFallbackRemote: false,
		trustedOrigins: []
	})
}));

describe('offline cold start after days without opening the app', () => {
	beforeEach(async () => {
		vi.resetModules();
		currentServerUrl = 'http://new.lan:8765';
		const { resetRefCacheForTests } = await import('$lib/offline/ref-cache');
		resetRefCacheForTests();
	});

	it('finds catalogs cached under a stale server origin not in profile', async () => {
		const staleOrigin = 'http://192.168.1.99:8765';
		currentServerUrl = staleOrigin;
		const { writeRefCache, readRefCache } = await import('$lib/offline/ref-cache');
		writeRefCache('/api/v1/categories?type=expense', [{ id: 'c1', name: 'Еда' }]);
		currentServerUrl = 'http://new.lan:8765';
		expect(readRefCache('/api/v1/categories?type=expense')).toMatchObject([{ id: 'c1' }]);
	});

	it('stable offline catalog survives ref-cache key drift', async () => {
		const {
			persistStableOfflineCatalog,
			readAccountsFromOfflineCache,
			readCategoriesFromOfflineCache
		} = await import('$lib/offline/ref-cache');
		persistStableOfflineCatalog({
			accounts: [{ id: 'a1', name: 'Наличные', type: 'cash', status: 'active' }],
			expense_categories: [{ id: 'c1', name: 'Еда' }],
			income_categories: [],
			banks: []
		});
		expect(readAccountsFromOfflineCache('active')).toMatchObject([{ id: 'a1' }]);
		expect(readCategoriesFromOfflineCache('expense')).toMatchObject([{ id: 'c1' }]);
	});

	it('reconcileOfflineCatalogsOnUnlock writes catalogs for the active origin', async () => {
		const { persistStableOfflineCatalog, reconcileOfflineCatalogsOnUnlock, readRefCache } =
			await import('$lib/offline/ref-cache');
		persistStableOfflineCatalog({
			accounts: [{ id: 'a1', name: 'Карта', type: 'bank', status: 'active' }],
			expense_categories: [{ id: 'c1', name: 'Еда' }],
			income_categories: [],
			banks: []
		});
		reconcileOfflineCatalogsOnUnlock();
		expect(readRefCache('/api/v1/accounts?status=active')).toMatchObject([{ id: 'a1' }]);
		expect(readRefCache('/api/v1/categories?type=expense')).toMatchObject([{ id: 'c1' }]);
	});
});
