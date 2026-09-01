import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import {
	clearRefCache,
	fetchWithRefCache,
	invalidateRefCacheAfterWrite,
	readAccountsFromOfflineCache,
	readCategoriesFromOfflineCache,
	readRefCache,
	refCacheReady,
	refCacheUpdate,
	resetRefCacheForTests,
	setRefCacheUserId,
	shouldPersistRefCache,
	writeRefCache
} from './ref-cache';

describe('shouldPersistRefCache', () => {
	it('caches credit detail and list paths', () => {
		expect(shouldPersistRefCache('/api/v1/credits/abc-123')).toBe(true);
		expect(shouldPersistRefCache('/api/v1/credits?status=active')).toBe(true);
		expect(shouldPersistRefCache('/api/v1/banks')).toBe(true);
	});

	it('skips setup status (registration flag for public pages)', () => {
		expect(shouldPersistRefCache('/api/v1/setup/status')).toBe(false);
	});
});

describe('web fetchWithRefCache SWR', () => {
	beforeEach(() => {
		resetRefCacheForTests();
		setRefCacheUserId('user-1');
	});

	afterEach(() => {
		vi.restoreAllMocks();
	});

	it('returns cached data immediately and revalidates in background', async () => {
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
		expect(fetcher).toHaveBeenCalledOnce();

		resolveFetch({ total_balance: 200 });
		await vi.waitFor(() =>
			expect(readRefCache('/api/v1/dashboard')).toEqual({ total_balance: 200 })
		);
	});

	it('emits refCacheUpdate with path when background data changes', async () => {
		writeRefCache('/api/v1/accounts', [{ id: 'a1' }]);
		let last: { path: string; seq: number } | null = null;
		const unsub = refCacheUpdate.subscribe((v) => (last = v));

		await fetchWithRefCache('/api/v1/accounts', async () => [{ id: 'a1' }, { id: 'a2' }]);
		await vi.waitFor(() => expect(last?.path).toBe('/api/v1/accounts'));
		unsub();
	});

	it('isolates cache per user id', () => {
		writeRefCache('/api/v1/dashboard', { total_balance: 1 });
		setRefCacheUserId('user-2');
		expect(refCacheReady('/api/v1/dashboard')).toBe(false);
	});

	it('clearRefCache drops cache and ignores in-flight revalidate writes', async () => {
		writeRefCache('/api/v1/debts?settled=false', [{ id: 'old' }]);
		let resolveFetch!: (value: unknown[]) => void;
		const fetcher = vi.fn(
			() =>
				new Promise<unknown[]>((resolve) => {
					resolveFetch = resolve;
				})
		);

		await fetchWithRefCache('/api/v1/debts?settled=false', fetcher);
		clearRefCache();
		expect(refCacheReady('/api/v1/debts?settled=false')).toBe(false);

		resolveFetch([{ id: 'stale-from-before-mutation' }]);
		await Promise.resolve();
		await Promise.resolve();
		expect(refCacheReady('/api/v1/debts?settled=false')).toBe(false);

		const fresh = await fetchWithRefCache('/api/v1/debts?settled=false', async () => [
			{ id: 'new' }
		]);
		expect(fresh).toEqual([{ id: 'new' }]);
		expect(readRefCache('/api/v1/debts?settled=false')).toEqual([{ id: 'new' }]);
	});

	it('preserveAuthMe keeps dictionaries and seeds accounts from ui/meta', () => {
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
		writeRefCache('/api/v1/dashboard', { total_balance: 1 });
		clearRefCache({ preserveAuthMe: true });
		expect(readRefCache('/api/v1/dashboard')).toBeNull();
		expect(readRefCache('/api/v1/categories?type=expense')).toMatchObject([{ id: 'c1' }]);
		expect(readRefCache('/api/v1/accounts?status=active')).toMatchObject([{ id: 'a1' }]);
	});

	it('invalidateRefCacheAfterWrite clears preserved dictionary cache for the mutated resource', () => {
		writeRefCache('/api/v1/categories?type=expense', [{ id: 'c1', name: 'Еда' }]);
		writeRefCache('/api/v1/accounts?status=active', [{ id: 'a1', name: 'Наличные' }]);
		writeRefCache('/api/v1/dashboard', { total_balance: 1 });
		clearRefCache({ preserveAuthMe: true });
		expect(readRefCache('/api/v1/categories?type=expense')).toMatchObject([{ id: 'c1' }]);
		expect(readRefCache('/api/v1/accounts?status=active')).toMatchObject([{ id: 'a1' }]);

		invalidateRefCacheAfterWrite('/api/v1/categories');
		expect(readRefCache('/api/v1/categories?type=expense')).toBeNull();
		expect(readRefCache('/api/v1/accounts?status=active')).toMatchObject([{ id: 'a1' }]);

		invalidateRefCacheAfterWrite('/api/v1/accounts');
		expect(readRefCache('/api/v1/accounts?status=active')).toBeNull();
		expect(readRefCache('/api/v1/dashboard')).toBeNull();
	});

	it('readAccountsFromOfflineCache ignores empty list cache', () => {
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

	it('readCategoriesFromOfflineCache ignores empty list cache', () => {
		writeRefCache('/api/v1/ui/meta', {
			expense_categories: [{ id: 'c1', name: 'Еда' }],
			income_categories: []
		});
		writeRefCache('/api/v1/categories?type=expense', []);
		expect(readCategoriesFromOfflineCache('expense')).toMatchObject([{ id: 'c1', name: 'Еда' }]);
	});
});
