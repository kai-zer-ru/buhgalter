import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { Debt } from '$lib/api/client';
import { createDebt, deleteDebt } from '$lib/offline/debts-api';
import {
	readRefCache,
	refCacheReady,
	resetRefCacheForTests,
	writeRefCache
} from '$lib/offline/ref-cache';
import { getOutboxEntries, resetOutboxForTests } from '$lib/offline/store';
import { isLocalEntityKey } from '$lib/offline/types';
import * as client from '$lib/api/client';
import * as connectivity from '$lib/offline/server-connectivity';
import * as network from '$lib/offline/network';
import * as sync from '$lib/offline/sync';

vi.mock('$lib/api/client', () => ({
	createDebt: vi.fn(),
	deleteDebt: vi.fn(),
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

const debtPayload = {
	debtor_name: 'Иван',
	direction: 'lent' as const,
	amount: '1000.00',
	debt_date: '2026-07-08 10:00:00',
	due_date: '2026-07-15 23:59:59',
	affects_balance: false
};

const serverDebt: Debt = {
	id: 'srv-d1',
	debtor_id: 'debtor-1',
	debtor_name: 'Иван',
	direction: 'lent',
	amount: 100_000,
	amount_display: '1000.00',
	affects_balance: false,
	debt_date: '2026-07-08 10:00:00',
	due_date: '2026-07-15 23:59:59',
	description: null,
	transaction_id: null,
	is_settled: false,
	settled_at: null,
	is_overdue: false,
	created_at: '2026-07-08T10:00:00Z'
};

beforeEach(() => {
	resetOutboxForTests();
	resetRefCacheForTests();
	vi.clearAllMocks();
	vi.mocked(network.shouldUseOfflineQueue).mockReturnValue(true);
	vi.mocked(connectivity.shouldTryServer).mockResolvedValue(false);
	vi.mocked(connectivity.isConnectionError).mockReturnValue(true);
	writeRefCache('/api/v1/debts?settled=false', []);
	writeRefCache('/api/v1/debts/summary', { total_lent: 0 });
	writeRefCache('/api/v1/debtors', []);
});

describe('debts-api offline', () => {
	it('createDebt enqueues local entry and patches active list', async () => {
		const debt = await createDebt(debtPayload);

		expect(isLocalEntityKey(debt.id)).toBe(true);
		expect(debt.amount_display).toBe('1000.00');
		expect(getOutboxEntries()).toHaveLength(1);
		expect(getOutboxEntries()[0].kind).toBe('debt');
		expect(readRefCache<Debt[]>('/api/v1/debts?settled=false')?.[0]?.id).toBe(debt.id);
		expect(refCacheReady('/api/v1/debts/summary')).toBe(false);
	});

	it('createDebt with new debtor name patches debtors list', async () => {
		await createDebt(debtPayload);

		const debtors = readRefCache<{ id: string; name: string }[]>('/api/v1/debtors');
		expect(debtors).toHaveLength(1);
		expect(debtors?.[0]?.name).toBe('Иван');
		expect(isLocalEntityKey(debtors?.[0]?.id ?? '')).toBe(true);
	});

	it('deleteDebt removes local entry and list row', async () => {
		const debt = await createDebt(debtPayload);
		await deleteDebt(debt.id);

		expect(getOutboxEntries()).toHaveLength(0);
		expect(readRefCache<Debt[]>('/api/v1/debts?settled=false')).toEqual([]);
	});
});

describe('debts-api online', () => {
	it('createDebt calls API and patches cache when server responds', async () => {
		vi.mocked(connectivity.shouldTryServer).mockResolvedValue(true);
		vi.mocked(client.createDebt).mockResolvedValue(serverDebt);
		writeRefCache('/api/v1/debtors', []);

		const debt = await createDebt(debtPayload);

		expect(debt.id).toBe('srv-d1');
		expect(client.createDebt).toHaveBeenCalledWith(debtPayload);
		expect(readRefCache<Debt[]>('/api/v1/debts?settled=false')?.[0]?.id).toBe('srv-d1');
		expect(readRefCache<{ id: string; name: string }[]>('/api/v1/debtors')?.[0]?.id).toBe(
			'debtor-1'
		);
		expect(sync.scheduleSyncOutbox).toHaveBeenCalled();
	});

	it('queues delete when server is unreachable', async () => {
		vi.mocked(connectivity.shouldTryServer).mockResolvedValue(true);
		vi.mocked(client.deleteDebt).mockRejectedValue(new TypeError('Failed to fetch'));

		await deleteDebt('srv-d1');

		expect(getOutboxEntries()).toHaveLength(1);
		expect(getOutboxEntries()[0].op).toBe('delete');
		expect(getOutboxEntries()[0].kind).toBe('debt');
	});
});
