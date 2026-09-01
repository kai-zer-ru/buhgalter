import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import {
	fetchWithRefCache,
	readRefCache,
	refCacheReady,
	refCacheTick,
	refCacheUpdate,
	resetRefCacheForTests,
	shouldPersistRefCache,
	writeRefCache
} from './ref-cache';
import * as connectivity from './server-connectivity';

let currentServerUrl = 'http://test.local:8765';

vi.mock('$lib/platform/server-url', () => ({
	getServerUrl: () => currentServerUrl
}));

vi.mock('$lib/platform/server-profile', () => ({
	getServerProfile: () => ({
		lanUrl: 'http://lan.local:8765',
		remoteUrl: 'http://test.local:8765',
		homeSsids: [],
		lanFallbackRemote: false,
		trustedOrigins: []
	})
}));

describe('shouldPersistRefCache', () => {
	it('caches credit detail and list paths; skips health and setup/status', () => {
		expect(shouldPersistRefCache('/api/v1/credits/abc-123')).toBe(true);
		expect(shouldPersistRefCache('/api/v1/credits?status=active')).toBe(true);
		expect(shouldPersistRefCache('/api/v1/credits/abc-123/payments')).toBe(true);
		expect(shouldPersistRefCache('/api/v1/banks')).toBe(true);
		expect(shouldPersistRefCache('/api/v1/health')).toBe(false);
		expect(shouldPersistRefCache('/api/v1/setup/status')).toBe(false);
	});
});

describe('fetchWithRefCache SWR', () => {
	beforeEach(() => {
		resetRefCacheForTests();
		currentServerUrl = 'http://test.local:8765';
		vi.spyOn(connectivity, 'isServerOfflineMode').mockReturnValue(false);
		vi.spyOn(connectivity, 'markServerOnline').mockImplementation(() => {});
		vi.spyOn(connectivity, 'markServerOffline').mockImplementation(() => {});
	});

	afterEach(() => {
		vi.restoreAllMocks();
	});

	it('returns cached data immediately when online and revalidates in background', async () => {
		writeRefCache('/api/v1/dashboard', { total_balance: 100 });
		let resolveFetch!: (value: { total_balance: number }) => void;
		const fetcher = vi.fn(
			() =>
				new Promise<{ total_balance: number }>((resolve) => {
					resolveFetch = resolve;
				})
		);

		const first = await fetchWithRefCache('/api/v1/dashboard', fetcher);
		expect(first).toEqual({ total_balance: 100 });
		await vi.waitFor(() => expect(fetcher).toHaveBeenCalledOnce());

		resolveFetch({ total_balance: 200 });
		await vi.waitFor(() =>
			expect(readRefCache('/api/v1/dashboard')).toEqual({ total_balance: 200 })
		);
	});

	it('does not rewrite cache or notify when revalidate payload is identical', async () => {
		const dash = {
			total_balance: 100,
			accounts: [],
			debts_summary: { i_owe: 0 },
			recent_transactions: []
		};
		writeRefCache('/api/v1/dashboard', dash);
		let notified = false;
		let ignoreInitial = true;
		const unsub = refCacheUpdate.subscribe((v) => {
			if (ignoreInitial) return;
			if (v?.path === '/api/v1/dashboard') notified = true;
		});
		ignoreInitial = false;

		let resolveFetch!: (value: typeof dash) => void;
		const fetcher = vi.fn(
			() =>
				new Promise<typeof dash>((resolve) => {
					resolveFetch = resolve;
				})
		);
		await fetchWithRefCache('/api/v1/dashboard', fetcher);
		await vi.waitFor(() => expect(fetcher).toHaveBeenCalledOnce());
		resolveFetch({ ...dash });
		await vi.waitFor(() => expect(fetcher.mock.results[0]?.type).toBe('return'));
		await Promise.resolve();
		await Promise.resolve();
		expect(notified).toBe(false);
		expect(readRefCache('/api/v1/dashboard')).toEqual(dash);
		unsub();
	});

	it('skips write when only dashboard *_display fields differ', async () => {
		const prev = {
			total_balance: 50,
			total_forecast: 50,
			accounts: [{ id: 'a1', balance: 50, balance_display: '50,00 ₽' }],
			debts_summary: { i_owe: 0 },
			recent_transactions: []
		};
		writeRefCache('/api/v1/dashboard', prev);
		expect(
			writeRefCache('/api/v1/dashboard', {
				...prev,
				accounts: [{ id: 'a1', balance: 50, balance_display: '50.00 RUB' }]
			})
		).toBe(false);
		expect(readRefCache('/api/v1/dashboard')).toEqual(prev);
	});

	it('emits refCacheUpdate with path when background revalidate changes data', async () => {
		writeRefCache('/api/v1/accounts', [{ id: 'a1' }]);
		let last: { path: string; seq: number } | null = null;
		const unsub = refCacheUpdate.subscribe((v) => (last = v));

		await fetchWithRefCache('/api/v1/accounts', async () => [{ id: 'a1' }, { id: 'a2' }]);
		await vi.waitFor(() => expect(last?.path).toBe('/api/v1/accounts'));
		unsub();
	});

	it('emits refCacheUpdate when background revalidate changes data', async () => {
		writeRefCache('/api/v1/accounts', [{ id: 'a1' }]);
		let tick = 0;
		const unsub = refCacheTick.subscribe((n) => (tick = n));

		await fetchWithRefCache('/api/v1/accounts', async () => [{ id: 'a1' }, { id: 'a2' }]);
		await vi.waitFor(() => expect(tick).toBe(0));
		unsub();
	});

	it('blocks on network when cache is empty', async () => {
		const fetcher = vi.fn().mockResolvedValue({ ok: true });
		const value = await fetchWithRefCache('/api/v1/dashboard', fetcher);
		expect(value).toEqual({ ok: true });
		expect(fetcher).toHaveBeenCalledOnce();
		expect(refCacheReady('/api/v1/dashboard')).toBe(true);
	});

	it('serves cache only when offline mode', async () => {
		vi.spyOn(connectivity, 'isServerOfflineMode').mockReturnValue(true);
		writeRefCache('/api/v1/credits?status=active', [{ id: 'c1' }]);
		const fetcher = vi.fn();

		const value = await fetchWithRefCache('/api/v1/credits?status=active', fetcher);
		expect(value).toEqual([{ id: 'c1' }]);
		expect(fetcher).not.toHaveBeenCalled();
	});

	it('serves credit detail from cache when offline', async () => {
		vi.spyOn(connectivity, 'isServerOfflineMode').mockReturnValue(true);
		const detail = { id: 'c1', schedule: [{ id: 'p1', amount: 100 }] };
		writeRefCache('/api/v1/credits/c1', detail);
		const fetcher = vi.fn();

		const value = await fetchWithRefCache('/api/v1/credits/c1', fetcher);
		expect(value).toEqual(detail);
		expect(fetcher).not.toHaveBeenCalled();
	});

	it('reuses cached data from another configured origin after active url switch', async () => {
		currentServerUrl = 'http://lan.local:8765';
		const cached = { total_balance: 321 };
		writeRefCache('/api/v1/dashboard', cached);

		currentServerUrl = 'http://test.local:8765';
		expect(readRefCache('/api/v1/dashboard')).toEqual(cached);
		expect(refCacheReady('/api/v1/dashboard')).toBe(true);
	});
});

describe('clearRefCache preserveAuthMe', () => {
	beforeEach(() => {
		resetRefCacheForTests();
	});

	it('keeps /auth/me and account lists when preserveAuthMe is set', async () => {
		const { clearRefCache, readRefCache, writeRefCache } = await import('./ref-cache');
		writeRefCache('/api/v1/auth/me', { id: 'u1' });
		writeRefCache('/api/v1/accounts', [{ id: 'a1' }]);
		writeRefCache('/api/v1/accounts?status=active', [{ id: 'a1' }]);
		writeRefCache('/api/v1/dashboard', { total: 1 });
		clearRefCache({ preserveAuthMe: true });
		expect(readRefCache('/api/v1/auth/me')).toEqual({ id: 'u1' });
		expect(readRefCache('/api/v1/accounts')).toEqual([{ id: 'a1' }]);
		expect(readRefCache('/api/v1/accounts?status=active')).toEqual([{ id: 'a1' }]);
		expect(readRefCache('/api/v1/dashboard')).toBeNull();
	});

	it('keeps category dictionaries so offline forms survive a write', async () => {
		const { clearRefCache, readRefCache, writeRefCache } = await import('./ref-cache');
		const cats = [{ id: 'c1', name: 'Еда' }];
		const subs = [{ id: 's1', name: 'Кафе' }];
		writeRefCache('/api/v1/categories?type=expense', cats);
		writeRefCache('/api/v1/categories/c1/subcategories', subs);
		writeRefCache('/api/v1/ui/meta', { expense_categories: cats });
		writeRefCache('/api/v1/merchants', [{ id: 'm1' }]);
		writeRefCache('/api/v1/dashboard', { total: 1 });
		clearRefCache({ preserveAuthMe: true });
		expect(readRefCache('/api/v1/categories?type=expense')).toEqual(cats);
		expect(readRefCache('/api/v1/categories/c1/subcategories')).toEqual(subs);
		expect(readRefCache('/api/v1/ui/meta')).toEqual({ expense_categories: cats });
		expect(readRefCache('/api/v1/merchants')).toEqual([{ id: 'm1' }]);
		expect(readRefCache('/api/v1/dashboard')).toBeNull();
	});

	it('seedDictionariesFromUIMeta overwrites dictionaries without clearing other keys', async () => {
		const { readRefCache, seedDictionariesFromUIMeta, writeRefCache } = await import('./ref-cache');
		writeRefCache('/api/v1/dashboard', { total: 1 });
		writeRefCache('/api/v1/merchants', [{ id: 'old' }]);
		seedDictionariesFromUIMeta({
			expense_categories: [{ id: 'c1' }],
			income_categories: [],
			merchants: [{ id: 'm1' }],
			tags: [{ id: 't1' }],
			banks: [{ id: 'b1' }],
			transaction_templates: [{ id: 'tpl1' }],
			debtors: [{ id: 'd1' }]
		});
		expect(readRefCache('/api/v1/merchants')).toEqual([{ id: 'm1' }]);
		expect(readRefCache('/api/v1/tags')).toEqual([{ id: 't1' }]);
		expect(readRefCache('/api/v1/banks')).toEqual([{ id: 'b1' }]);
		expect(readRefCache('/api/v1/transaction-templates')).toEqual([{ id: 'tpl1' }]);
		expect(readRefCache('/api/v1/debtors')).toEqual([{ id: 'd1' }]);
		expect(readRefCache('/api/v1/categories?type=expense')).toEqual([{ id: 'c1' }]);
		expect(readRefCache('/api/v1/dashboard')).toEqual({ total: 1 });
	});

	it('seeds empty account lists from ui/meta after a mutation clear', async () => {
		const { clearRefCache, readRefCache, writeRefCache } = await import('./ref-cache');
		writeRefCache('/api/v1/ui/meta', {
			accounts: [{ id: 'a1', name: 'Наличные', type: 'cash', status: 'active' }],
			banks: [],
			expense_categories: [{ id: 'c1', name: 'Еда' }],
			income_categories: [],
			debtors: [],
			merchants: [],
			tags: [],
			active_credits: [],
			closed_credits: []
		});
		writeRefCache('/api/v1/dashboard', { total: 1, accounts: [] });
		clearRefCache({ preserveAuthMe: true });
		expect(readRefCache('/api/v1/dashboard')).toBeNull();
		expect(readRefCache('/api/v1/accounts?status=active')).toMatchObject([
			{ id: 'a1', name: 'Наличные', type: 'cash', status: 'active' }
		]);
	});

	it('full clear still removes dictionaries', async () => {
		const { clearRefCache, readRefCache, writeRefCache } = await import('./ref-cache');
		writeRefCache('/api/v1/categories?type=expense', [{ id: 'c1' }]);
		writeRefCache('/api/v1/auth/me', { id: 'u1' });
		clearRefCache();
		expect(readRefCache('/api/v1/categories?type=expense')).toBeNull();
		expect(readRefCache('/api/v1/auth/me')).toBeNull();
	});
});

describe('readAccountsFromOfflineCache', () => {
	beforeEach(() => {
		resetRefCacheForTests();
		currentServerUrl = 'http://test.local:8765';
	});

	it('returns cached active accounts list', async () => {
		const { readAccountsFromOfflineCache, writeRefCache } = await import('./ref-cache');
		const rows = [
			{
				id: 'a1',
				name: 'Карта',
				type: 'bank' as const,
				bank_id: null,
				initial_balance: 0,
				balance: 100,
				balance_display: '1.00',
				status: 'active' as const,
				is_primary: true,
				created_at: '',
				updated_at: ''
			}
		];
		writeRefCache('/api/v1/accounts?status=active', rows);
		expect(readAccountsFromOfflineCache('active')).toEqual(rows);
	});

	it('falls back to ui/meta when accounts list was cleared on write', async () => {
		const { readAccountsFromOfflineCache, writeRefCache } = await import('./ref-cache');
		writeRefCache('/api/v1/ui/meta', {
			accounts: [{ id: 'a1', name: 'Наличные', type: 'cash', status: 'active' }],
			banks: [],
			expense_categories: [],
			income_categories: [],
			debtors: [],
			merchants: [],
			tags: [],
			active_credits: [],
			closed_credits: []
		});
		expect(readAccountsFromOfflineCache('active')).toMatchObject([
			{ id: 'a1', name: 'Наличные', type: 'cash', status: 'active' }
		]);
	});

	it('ignores empty cached accounts list and falls back to ui/meta', async () => {
		const { readAccountsFromOfflineCache, writeRefCache } = await import('./ref-cache');
		writeRefCache('/api/v1/ui/meta', {
			accounts: [{ id: 'a1', name: 'Наличные', type: 'cash', status: 'active' }],
			banks: [],
			expense_categories: [],
			income_categories: [],
			debtors: [],
			merchants: [],
			tags: [],
			active_credits: [],
			closed_credits: []
		});
		writeRefCache('/api/v1/accounts?status=active', []);
		expect(readAccountsFromOfflineCache('active')).toMatchObject([{ id: 'a1' }]);
	});

	it('readCategoriesFromOfflineCache ignores empty list cache', async () => {
		const { readCategoriesFromOfflineCache, writeRefCache } = await import('./ref-cache');
		writeRefCache('/api/v1/ui/meta', {
			expense_categories: [{ id: 'c1', name: 'Еда' }],
			income_categories: []
		});
		writeRefCache('/api/v1/categories?type=expense', []);
		expect(readCategoriesFromOfflineCache('expense')).toMatchObject([{ id: 'c1', name: 'Еда' }]);
	});

	it('merges dashboard balances into ui/meta accounts', async () => {
		const { readAccountsFromOfflineCache, writeRefCache } = await import('./ref-cache');
		writeRefCache('/api/v1/ui/meta', {
			accounts: [{ id: 'a1', name: 'Карта', type: 'bank', status: 'active' }],
			banks: [],
			expense_categories: [],
			income_categories: [],
			debtors: [],
			merchants: [],
			tags: [],
			active_credits: [],
			closed_credits: []
		});
		writeRefCache('/api/v1/dashboard', {
			total_balance: 500,
			total_forecast: 500,
			accounts: [
				{
					id: 'a1',
					name: 'Карта',
					type: 'bank',
					balance: 500,
					balance_display: '5.00',
					forecast_balance: 500,
					forecast_display: '5.00',
					has_future_this_month: false,
					is_primary: true
				}
			],
			recent_transactions: [],
			debts_summary: { i_owe: 0 }
		});
		expect(readAccountsFromOfflineCache('active')?.[0]).toMatchObject({
			id: 'a1',
			balance: 500,
			is_primary: true
		});
	});
});

describe('enrichAccountsWithCachedBalances', () => {
	it('overlays dashboard balances onto account list rows', async () => {
		const { enrichAccountsWithCachedBalances } = await import('./ref-cache');
		const rows = enrichAccountsWithCachedBalances(
			[
				{
					id: 'a1',
					name: 'Карта',
					type: 'bank',
					bank_id: null,
					initial_balance: 0,
					balance: 0,
					balance_display: '0.00',
					status: 'active',
					is_primary: false,
					created_at: '',
					updated_at: ''
				}
			],
			{
				total_balance: 500,
				total_forecast: 500,
				accounts: [
					{
						id: 'a1',
						name: 'Карта',
						type: 'bank',
						balance: 500,
						balance_display: '5.00',
						forecast_balance: 500,
						forecast_display: '5.00',
						has_future_this_month: false,
						is_primary: true
					}
				],
				recent_transactions: [],
				debts_summary: {
					i_owe: 0,
					owed_to_me: 0,
					overdue_i_owe: 0,
					overdue_owed_to_me: 0,
					active_count: 0
				}
			}
		);
		expect(rows[0]).toMatchObject({ balance: 500, balance_display: '5.00', is_primary: true });
	});

	it('patches preserved account list caches when dashboard is written', async () => {
		const { writeRefCache, readRefCache, resetRefCacheForTests } = await import('./ref-cache');
		resetRefCacheForTests();
		writeRefCache('/api/v1/accounts?status=active', [
			{
				id: 'a1',
				name: 'Карта',
				type: 'bank',
				bank_id: null,
				initial_balance: 0,
				balance: 0,
				balance_display: '0.00',
				status: 'active',
				is_primary: false,
				created_at: '',
				updated_at: ''
			}
		]);
		writeRefCache('/api/v1/dashboard', {
			total_balance: 500,
			total_forecast: 500,
			accounts: [
				{
					id: 'a1',
					name: 'Карта',
					type: 'bank',
					balance: 500,
					balance_display: '5.00',
					forecast_balance: 500,
					forecast_display: '5.00',
					has_future_this_month: false,
					is_primary: true
				}
			],
			recent_transactions: [],
			debts_summary: {
				i_owe: 0,
				owed_to_me: 0,
				overdue_i_owe: 0,
				overdue_owed_to_me: 0,
				active_count: 0
			}
		});
		expect(readRefCache('/api/v1/accounts?status=active')?.[0]).toMatchObject({
			balance: 500,
			balance_display: '5.00'
		});
	});
});

describe('isPreservedOfflineRefPath', () => {
	it('matches dictionary paths used by offline forms', async () => {
		const { isPreservedOfflineRefPath } = await import('./ref-cache');
		expect(isPreservedOfflineRefPath('/api/v1/auth/me')).toBe(true);
		expect(isPreservedOfflineRefPath('/api/v1/categories?type=income')).toBe(true);
		expect(isPreservedOfflineRefPath('/api/v1/categories/abc/subcategories')).toBe(true);
		expect(isPreservedOfflineRefPath('/api/v1/transaction-templates')).toBe(true);
		expect(isPreservedOfflineRefPath('/api/v1/dashboard')).toBe(false);
		expect(isPreservedOfflineRefPath('/api/v1/accounts?status=active')).toBe(true);
		expect(isPreservedOfflineRefPath('/api/v1/accounts')).toBe(true);
		expect(isPreservedOfflineRefPath('/api/v1/transactions')).toBe(false);
	});
});
