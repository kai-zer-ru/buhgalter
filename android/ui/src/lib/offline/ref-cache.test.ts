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

vi.mock('$lib/platform/server-url', () => ({
	getServerUrl: () => 'http://test.local:8765'
}));

describe('shouldPersistRefCache', () => {
	it('skips credit detail (full schedule) but keeps list/action paths', () => {
		expect(shouldPersistRefCache('/api/v1/credits/abc-123')).toBe(false);
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
		expect(fetcher).toHaveBeenCalledOnce();

		resolveFetch({ total_balance: 200 });
		await vi.waitFor(() =>
			expect(readRefCache('/api/v1/dashboard')).toEqual({ total_balance: 200 })
		);
	});

	it('emits refCacheUpdate with path when background revalidate changes data', async () => {
		writeRefCache('/api/v1/accounts', [{ id: 'a1' }]);
		let last: { path: string; seq: number } | null = null;
		const unsub = refCacheUpdate.subscribe((v) => (last = v));

		await fetchWithRefCache('/api/v1/accounts', async () => [{ id: 'a1' }, { id: 'a2' }]);
		await vi.waitFor(() => expect(last?.path).toBe('/api/v1/accounts'));
		unsub();
	});

	it('bumps refCacheTick when background revalidate changes data', async () => {
		writeRefCache('/api/v1/accounts', [{ id: 'a1' }]);
		let tick = 0;
		const unsub = refCacheTick.subscribe((n) => (tick = n));

		await fetchWithRefCache('/api/v1/accounts', async () => [{ id: 'a1' }, { id: 'a2' }]);
		await vi.waitFor(() => expect(tick).toBeGreaterThan(0));
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
});

describe('clearRefCache preserveAuthMe', () => {
	beforeEach(() => {
		resetRefCacheForTests();
	});

	it('keeps /auth/me when preserveAuthMe is set', async () => {
		const { clearRefCache, readRefCache, writeRefCache } = await import('./ref-cache');
		writeRefCache('/api/v1/auth/me', { id: 'u1' });
		writeRefCache('/api/v1/accounts', [{ id: 'a1' }]);
		clearRefCache({ preserveAuthMe: true });
		expect(readRefCache('/api/v1/auth/me')).toEqual({ id: 'u1' });
		expect(readRefCache('/api/v1/accounts')).toBeNull();
	});
});
