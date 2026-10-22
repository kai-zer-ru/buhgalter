import { beforeEach, describe, expect, it, vi } from 'vitest';
import { get } from 'svelte/store';
import {
	refCacheReady,
	refCacheUpdate,
	resetRefCacheForTests,
	setRefCacheUserId,
	writeRefCache
} from '$lib/ref-cache';

describe('web client write soft-reload', () => {
	beforeEach(() => {
		resetRefCacheForTests();
		setRefCacheUserId('user-1');
		vi.restoreAllMocks();
	});

	it('successful transaction DELETE clears SWR and emits refCacheUpdate *', async () => {
		writeRefCache('/api/v1/dashboard', { total_balance: 100 });
		expect(refCacheReady('/api/v1/dashboard')).toBe(true);

		vi.stubGlobal(
			'fetch',
			vi.fn().mockResolvedValue({
				ok: true,
				status: 204,
				json: async () => ({})
			})
		);

		const { deleteTransaction } = await import('$lib/api/client');
		await deleteTransaction('tx-1');

		expect(refCacheReady('/api/v1/dashboard')).toBe(false);
		expect(get(refCacheUpdate)?.path).toBe('*');
	});
});
