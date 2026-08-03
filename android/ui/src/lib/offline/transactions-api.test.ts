import { beforeEach, describe, expect, it, vi } from 'vitest';
import { get } from 'svelte/store';
import { deleteTransaction, deleteTransfer } from '$lib/offline/transactions-api';
import { getOutboxEntries, resetOutboxForTests } from '$lib/offline/store';
import * as client from '$lib/api/client';
import * as connectivity from '$lib/offline/server-connectivity';
import * as network from '$lib/offline/network';
import * as sync from '$lib/offline/sync';
import { dataRefreshTick } from '$lib/offline/sync';

vi.mock('$lib/api/client', () => ({
	createTransaction: vi.fn(),
	updateTransaction: vi.fn(),
	deleteTransaction: vi.fn(),
	createTransfer: vi.fn(),
	updateTransfer: vi.fn(),
	deleteTransfer: vi.fn(),
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

beforeEach(() => {
	resetOutboxForTests();
	dataRefreshTick.set(0);
	vi.clearAllMocks();
	vi.mocked(network.shouldUseOfflineQueue).mockReturnValue(true);
	vi.mocked(connectivity.shouldTryServer).mockResolvedValue(false);
	vi.mocked(connectivity.isConnectionError).mockReturnValue(true);
});

describe('transactions-api delete', () => {
	it('online delete bumps dataRefreshTick so lists reload without waiting for sync', async () => {
		vi.mocked(connectivity.shouldTryServer).mockResolvedValue(true);
		vi.mocked(client.deleteTransaction).mockResolvedValue(undefined);

		await deleteTransaction('tx-1');

		expect(client.deleteTransaction).toHaveBeenCalledWith('tx-1');
		expect(getOutboxEntries()).toHaveLength(0);
		expect(get(dataRefreshTick)).toBe(1);
		expect(sync.scheduleSyncOutbox).toHaveBeenCalled();
	});

	it('online transfer delete bumps dataRefreshTick', async () => {
		vi.mocked(connectivity.shouldTryServer).mockResolvedValue(true);
		vi.mocked(client.deleteTransfer).mockResolvedValue(undefined);

		await deleteTransfer('grp-1');

		expect(client.deleteTransfer).toHaveBeenCalledWith('grp-1');
		expect(get(dataRefreshTick)).toBe(1);
		expect(sync.scheduleSyncOutbox).toHaveBeenCalled();
	});

	it('queues delete when server is unreachable (local overlay via outbox)', async () => {
		vi.mocked(connectivity.shouldTryServer).mockResolvedValue(true);
		vi.mocked(client.deleteTransaction).mockRejectedValue(new TypeError('Failed to fetch'));

		await deleteTransaction('tx-2');

		expect(getOutboxEntries()).toHaveLength(1);
		expect(getOutboxEntries()[0]).toMatchObject({ op: 'delete', entityKey: 'tx-2' });
		expect(get(dataRefreshTick)).toBe(0);
	});
});
