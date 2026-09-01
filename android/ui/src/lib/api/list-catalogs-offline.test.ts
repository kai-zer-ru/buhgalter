import { beforeEach, describe, expect, it, vi } from 'vitest';

vi.mock('svelte-i18n', () => ({
	locale: { subscribe: () => () => {} }
}));

vi.mock('$lib/platform/server-url', () => ({
	getApiBase: () => 'http://test.local:8765',
	getServerUrl: () => 'http://test.local:8765'
}));

vi.mock('$lib/platform/auth-token', () => ({
	authHeaders: () => ({}),
	getAuthToken: () => 'token',
	getAuthServerOrigin: () => 'http://test.local:8765'
}));

vi.mock('$lib/platform/native', () => ({
	isNativeApp: () => false
}));

vi.mock('$lib/offline/network', () => ({
	shouldUseOfflineQueue: () => true
}));

vi.mock('$lib/offline/server-connectivity', () => ({
	isServerOfflineMode: () => true,
	isConnectionError: () => false,
	markServerOffline: vi.fn(),
	markServerOnline: vi.fn()
}));

vi.mock('$lib/auth/session-expired', () => ({
	notifySessionExpired: vi.fn(),
	shouldLogoutOnApi401: () => false
}));

vi.mock('$lib/platform/debug-log', () => ({
	logApiRequest: () => 0,
	logApiResponse: vi.fn(),
	debugLogInfo: vi.fn(),
	debugLogWarn: vi.fn()
}));

const morningSync = {
	accounts: [{ id: 'a1', name: 'Наличные', type: 'cash' as const, status: 'active' as const }],
	banks: [] as { id: string; name: string; icon_path: string }[],
	expense_categories: [
		{
			id: 'c1',
			name: 'Еда',
			type: 'expense' as const,
			icon: 'food',
			sort_order: 0,
			is_primary: true,
			is_system: false,
			subcategory_count: 0,
			created_at: '2026-01-01T00:00:00Z'
		}
	],
	income_categories: [],
	debtors: [{ id: 'd1', name: 'Иван' }],
	merchants: [{ id: 'm1', name: 'Магнит', icon: 'default' }],
	tags: [{ id: 't1', name: 'работа' }],
	active_credits: [],
	closed_credits: []
};

describe('list catalogs offline empty-cache fallback', () => {
	beforeEach(async () => {
		vi.resetModules();
		const { resetRefCacheForTests } = await import('$lib/offline/ref-cache');
		resetRefCacheForTests();
	});

	it('listAccounts ignores empty cached list and uses ui/meta', async () => {
		const { writeRefCache } = await import('$lib/offline/ref-cache');
		writeRefCache('/api/v1/ui/meta', morningSync);
		writeRefCache('/api/v1/accounts?status=active', []);
		const { listAccounts } = await import('./client');
		await expect(listAccounts('active')).resolves.toMatchObject([{ id: 'a1', name: 'Наличные' }]);
	});

	it('listCategories ignores empty cached list and uses ui/meta', async () => {
		const { writeRefCache } = await import('$lib/offline/ref-cache');
		writeRefCache('/api/v1/ui/meta', morningSync);
		writeRefCache('/api/v1/categories?type=expense', []);
		const { listCategories } = await import('./client');
		await expect(listCategories('expense')).resolves.toMatchObject([{ id: 'c1', name: 'Еда' }]);
	});

	it('listMerchants and listDebtors fall back to ui/meta', async () => {
		const { writeRefCache } = await import('$lib/offline/ref-cache');
		writeRefCache('/api/v1/ui/meta', morningSync);
		const { listMerchants, listDebtors, listTags } = await import('./client');
		await expect(listMerchants()).resolves.toMatchObject([{ id: 'm1', name: 'Магнит' }]);
		await expect(listDebtors()).resolves.toMatchObject([{ id: 'd1', name: 'Иван' }]);
		await expect(listTags()).resolves.toMatchObject([{ id: 't1', name: 'работа' }]);
	});
});
