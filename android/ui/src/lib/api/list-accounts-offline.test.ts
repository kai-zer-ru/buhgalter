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

describe('listAccounts offline', () => {
	beforeEach(async () => {
		vi.resetModules();
		const { resetRefCacheForTests } = await import('$lib/offline/ref-cache');
		resetRefCacheForTests();
	});

	it('returns ui/meta accounts when list cache was cleared', async () => {
		const { writeRefCache } = await import('$lib/offline/ref-cache');
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
		const { listAccounts } = await import('./client');
		const rows = await listAccounts('active');
		expect(rows).toMatchObject([{ id: 'a1', name: 'Наличные', type: 'cash' }]);
	});
});
