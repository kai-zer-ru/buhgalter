import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import {
	clearRefCache,
	fetchWithRefCache,
	readRefCache,
	refCacheReady,
	refCacheUpdate,
	resetRefCacheForTests,
	setRefCacheUserId,
	shouldPersistRefCache,
	writeRefCache
} from './ref-cache';

describe('shouldPersistRefCache', () => {
	it('skips credit detail with full schedule', () => {
		expect(shouldPersistRefCache('/api/v1/credits/abc-123')).toBe(false);
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
});
