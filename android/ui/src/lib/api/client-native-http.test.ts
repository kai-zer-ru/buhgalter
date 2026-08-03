import { beforeEach, describe, expect, it, vi } from 'vitest';

const nativeHttpRequest = vi.fn();

vi.mock('svelte-i18n', () => ({
	locale: { subscribe: () => () => {} }
}));

vi.mock('$lib/platform/native', () => ({
	isNativeApp: () => true
}));

vi.mock('$lib/platform/server-url', () => ({
	getApiBase: () => 'https://buh.example.com'
}));

vi.mock('$lib/platform/auth-token', () => ({
	authHeaders: () => ({ Authorization: 'Bearer test' })
}));

vi.mock('$lib/platform/ssl-trust', () => ({
	isHttpsOrigin: () => true,
	isOriginTrusted: () => true,
	nativeHttpRequest: (...args: unknown[]) => nativeHttpRequest(...args),
	nativeResultToApiError: () => null
}));

vi.mock('$lib/offline/network', () => ({
	shouldUseOfflineQueue: () => false
}));

vi.mock('$lib/offline/server-connectivity', () => ({
	isServerOfflineMode: () => false
}));

vi.mock('$lib/api/cache', () => ({
	cachedGet: vi.fn(),
	invalidateApiCache: vi.fn(),
	seedStaticRef: vi.fn()
}));

vi.mock('$lib/offline/ref-cache', () => ({
	clearRefCache: vi.fn(),
	fetchWithRefCache: vi.fn(),
	isOfflineFetchError: () => false,
	OfflineCacheMissError: class OfflineCacheMissError extends Error {},
	readCategoriesFromUIMetaCache: vi.fn(),
	seedCategoriesFromUIMeta: vi.fn(),
	shouldPersistRefCache: () => false
}));

vi.mock('$lib/offline/transaction-index', () => ({
	indexTransactions: vi.fn()
}));

vi.mock('$lib/auth/session-expired', () => ({
	notifySessionExpired: vi.fn(),
	shouldRedirectApi401: () => false
}));

vi.mock('$lib/platform/debug-log', () => ({
	logApiRequest: () => 0,
	logApiResponse: vi.fn()
}));

vi.mock('$lib/platform/abort-timeout', () => ({
	abortTimeout: () => undefined
}));

describe('native HTTPS API client', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('treats DELETE 204 with empty body as success', async () => {
		nativeHttpRequest.mockResolvedValue({
			status: 204,
			ok: true,
			body: ''
		});

		const { deleteTransfer } = await import('./client');
		await expect(deleteTransfer('group-1')).resolves.toBeUndefined();
		expect(nativeHttpRequest).toHaveBeenCalledWith(
			'https://buh.example.com/api/v1/transfers/group-1',
			expect.objectContaining({ method: 'DELETE' })
		);
	});

	it('treats DELETE transaction 204 with empty body as success', async () => {
		nativeHttpRequest.mockResolvedValue({
			status: 204,
			ok: true,
			body: undefined
		});

		const { deleteTransaction } = await import('./client');
		await expect(deleteTransaction('tx-1')).resolves.toBeUndefined();
	});
});
