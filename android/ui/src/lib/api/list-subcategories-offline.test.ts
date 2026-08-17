import { beforeEach, describe, expect, it, vi } from 'vitest';

vi.mock('svelte-i18n', () => ({
	locale: { subscribe: () => () => {} }
}));

vi.mock('$lib/platform/server-url', () => ({
	getApiBase: () => 'http://test.local:8765',
	getServerUrl: () => 'http://test.local:8765'
}));

vi.mock('$lib/platform/auth-token', () => ({
	authHeaders: () => ({})
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
	shouldRedirectApi401: () => false
}));

vi.mock('$lib/platform/debug-log', () => ({
	logApiRequest: () => 0,
	logApiResponse: vi.fn(),
	debugLogInfo: vi.fn(),
	debugLogWarn: vi.fn()
}));

describe('listSubcategories offline', () => {
	beforeEach(async () => {
		vi.resetModules();
		const { resetRefCacheForTests } = await import('$lib/offline/ref-cache');
		resetRefCacheForTests();
	});

	it('returns empty list on offline cache miss instead of throwing', async () => {
		const { listSubcategories } = await import('./client');
		await expect(listSubcategories('missing-cat')).resolves.toEqual([]);
	});

	it('returns cached subcategories when offline', async () => {
		const { writeRefCache } = await import('$lib/offline/ref-cache');
		const subs = [
			{
				id: 's1',
				name: 'Кафе',
				category_id: 'c1',
				icon: 'food',
				sort_order: 0,
				created_at: '2026-01-01T00:00:00Z'
			}
		];
		writeRefCache('/api/v1/categories/c1/subcategories', subs);
		const { listSubcategories } = await import('./client');
		await expect(listSubcategories('c1')).resolves.toEqual(subs);
	});
});
