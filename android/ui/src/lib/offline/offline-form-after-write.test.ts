import { beforeEach, describe, expect, it, vi } from 'vitest';

/**
 * Regression: after an online write, mutation-clear used to wipe /accounts.
 * Opening «Расход» offline then had empty счёт/категория and nothing could be saved.
 */

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
	debtors: [],
	merchants: [],
	tags: [],
	active_credits: [],
	closed_credits: []
};

describe('offline expense form catalogs after an online write', () => {
	beforeEach(async () => {
		vi.resetModules();
		const { resetRefCacheForTests } = await import('$lib/offline/ref-cache');
		resetRefCacheForTests();
	});

	it('listAccounts and listCategories still work after mutation-clear (preserve catalogs)', async () => {
		const { writeRefCache, clearRefCache, seedDictionariesFromUIMeta } =
			await import('$lib/offline/ref-cache');
		writeRefCache('/api/v1/ui/meta', morningSync);
		seedDictionariesFromUIMeta(morningSync);
		writeRefCache('/api/v1/accounts?status=active', [
			{
				id: 'a1',
				name: 'Наличные',
				type: 'cash',
				bank_id: null,
				initial_balance: 0,
				balance: 100,
				balance_display: '1.00',
				status: 'active',
				is_primary: true,
				created_at: '',
				updated_at: ''
			}
		]);
		writeRefCache('/api/v1/dashboard', { total_balance: 100, accounts: [] });

		clearRefCache({ preserveAuthMe: true });

		const { listAccounts, listCategories } = await import('$lib/api/client');
		await expect(listAccounts('active')).resolves.toMatchObject([
			{ id: 'a1', name: 'Наличные' }
		]);
		await expect(listCategories('expense')).resolves.toMatchObject([{ id: 'c1', name: 'Еда' }]);
	});

	it('rebuilds /accounts from ui/meta when the list key was never warmed', async () => {
		const { writeRefCache, clearRefCache } = await import('$lib/offline/ref-cache');
		writeRefCache('/api/v1/ui/meta', morningSync);
		writeRefCache('/api/v1/categories?type=expense', morningSync.expense_categories);
		clearRefCache({ preserveAuthMe: true });

		const { listAccounts, listCategories } = await import('$lib/api/client');
		await expect(listAccounts('active')).resolves.toMatchObject([{ id: 'a1' }]);
		await expect(listCategories('expense')).resolves.toMatchObject([{ id: 'c1' }]);
	});
});
