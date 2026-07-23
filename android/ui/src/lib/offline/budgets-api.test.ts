import { beforeEach, describe, expect, it, vi } from 'vitest';
import { createBudget, deleteBudget, updateBudget } from '$lib/offline/budgets-api';
import { getOutboxEntries, resetOutboxForTests } from '$lib/offline/store';
import { isLocalEntityKey } from '$lib/offline/types';
import * as connectivity from '$lib/offline/server-connectivity';
import * as network from '$lib/offline/network';
import * as sync from '$lib/offline/sync';

vi.mock('$lib/api/client', () => ({
	createBudget: vi.fn(),
	updateBudget: vi.fn(),
	deleteBudget: vi.fn(),
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

const body = {
	name: 'Еда',
	scope: 'all_expense' as const,
	amount: '10000.00'
};

beforeEach(() => {
	resetOutboxForTests();
	vi.clearAllMocks();
	vi.mocked(network.shouldUseOfflineQueue).mockReturnValue(true);
	vi.mocked(connectivity.shouldTryServer).mockResolvedValue(false);
	vi.mocked(connectivity.isConnectionError).mockReturnValue(true);
});

describe('budgets-api offline', () => {
	it('createBudget enqueues with month', async () => {
		const item = await createBudget(body, '2026-07');
		expect(isLocalEntityKey(item.id)).toBe(true);
		expect(getOutboxEntries()[0].payload).toMatchObject({
			...body,
			month: '2026-07'
		});
	});

	it('updateBudget enqueues for server id', async () => {
		await updateBudget('b1', body, '2026-07');
		expect(getOutboxEntries()[0].op).toBe('update');
	});

	it('deleteBudget enqueues delete', async () => {
		await deleteBudget('b1', '2026-07');
		expect(getOutboxEntries()[0]).toMatchObject({ kind: 'budget', op: 'delete' });
	});
});
