import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { Account } from '$lib/api/client';
import { createAccount, updateAccount, archiveAccount } from '$lib/offline/accounts-api';
import { readRefCache, resetRefCacheForTests, writeRefCache } from '$lib/offline/ref-cache';
import { getOutboxEntries, resetOutboxForTests } from '$lib/offline/store';
import { isLocalEntityKey } from '$lib/offline/types';
import * as client from '$lib/api/client';
import * as connectivity from '$lib/offline/server-connectivity';
import * as network from '$lib/offline/network';
import * as sync from '$lib/offline/sync';

vi.mock('$lib/api/client', () => ({
	createAccount: vi.fn(),
	updateAccount: vi.fn(),
	archiveAccount: vi.fn(),
	unarchiveAccount: vi.fn(),
	ApiError: class ApiError extends Error {
		constructor(
			public code: string,
			message: string,
			public status: number
		) {
			super(message);
		}
	},
	isTransientHttpError: (status: number) => status === 503
}));

vi.mock('$lib/offline/server-connectivity', async (importOriginal) => {
	const actual = await importOriginal<typeof connectivity>();
	return {
		...actual,
		shouldTryServer: vi.fn(),
		markServerOffline: vi.fn(),
		isConnectionError: vi.fn()
	};
});

vi.mock('$lib/offline/network', () => ({
	shouldUseOfflineQueue: vi.fn()
}));

vi.mock('$lib/offline/sync', async (importOriginal) => {
	const actual = await importOriginal<typeof sync>();
	return {
		...actual,
		scheduleSyncOutbox: vi.fn()
	};
});

vi.mock('$lib/offline/merge', () => ({
	refreshMergeMeta: vi.fn().mockResolvedValue(undefined)
}));

const payload = {
	name: 'Наличные',
	type: 'cash' as const,
	initial_balance: '100.00'
};

beforeEach(() => {
	resetOutboxForTests();
	resetRefCacheForTests();
	vi.clearAllMocks();
	vi.mocked(network.shouldUseOfflineQueue).mockReturnValue(true);
	vi.mocked(connectivity.shouldTryServer).mockResolvedValue(false);
	vi.mocked(connectivity.isConnectionError).mockReturnValue(true);
	writeRefCache('/api/v1/accounts?status=active', []);
});

describe('accounts-api offline', () => {
	it('createAccount enqueues local entry and patches cache', async () => {
		const account = await createAccount(payload);
		expect(isLocalEntityKey(account.id)).toBe(true);
		expect(getOutboxEntries()[0].kind).toBe('account');
		expect(readRefCache<Account[]>('/api/v1/accounts?status=active')?.[0]?.id).toBe(account.id);
	});

	it('updateAccount enqueues for server id', async () => {
		await updateAccount('acc-1', { name: 'Карта', initial_balance: '0' });
		expect(getOutboxEntries()).toHaveLength(1);
		expect(getOutboxEntries()[0].op).toBe('update');
		expect(client.updateAccount).not.toHaveBeenCalled();
	});

	it('archiveAccount enqueues status payload', async () => {
		await archiveAccount('acc-1', 'acc-2');
		const entry = getOutboxEntries()[0];
		expect(entry.kind).toBe('account');
		expect(entry.payload).toEqual({
			action: 'archive',
			transfer_to_account_id: 'acc-2'
		});
	});
});
