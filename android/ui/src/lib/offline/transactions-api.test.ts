import { beforeEach, describe, expect, it, vi } from 'vitest';
import { get } from 'svelte/store';
import {
	createTransaction,
	createTransfer,
	deleteTransaction,
	deleteTransfer
} from '$lib/offline/transactions-api';
import { getOutboxEntries, resetOutboxForTests } from '$lib/offline/store';
import * as client from '$lib/api/client';
import * as connectivity from '$lib/offline/server-connectivity';
import * as network from '$lib/offline/network';
import * as sync from '$lib/offline/sync';
import { dataRefreshTick } from '$lib/offline/sync';
import {
	enrichAccountsWithCachedBalances,
	readRefCache,
	resetRefCacheForTests,
	writeRefCache
} from '$lib/offline/ref-cache';
import { touchLedgerAfterTransactionMutation } from '$lib/offline/ref-cache-mutations';

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
	resetRefCacheForTests();
	dataRefreshTick.set(0);
	vi.clearAllMocks();
	vi.mocked(network.shouldUseOfflineQueue).mockReturnValue(true);
	vi.mocked(connectivity.shouldTryServer).mockResolvedValue(false);
	vi.mocked(connectivity.isConnectionError).mockReturnValue(true);
});

describe('transactions-api delete', () => {
	it('queues create when server is unreachable so the operation stays on device', async () => {
		const tx = await createTransaction({
			account_id: 'a1',
			type: 'expense',
			amount: '1254.00',
			category_id: 'c1',
			transaction_date: '2026-09-01 12:41:00'
		});

		expect(client.createTransaction).not.toHaveBeenCalled();
		expect(getOutboxEntries()).toHaveLength(1);
		expect(getOutboxEntries()[0]).toMatchObject({
			op: 'create',
			kind: 'transaction',
			payload: { account_id: 'a1', amount: '1254.00', category_id: 'c1' }
		});
		expect(tx.account_id).toBe('a1');
		expect(tx.amount_display).toBe('1254.00');
	});

	it('online delete bumps dataRefreshTick so lists reload without waiting for sync', async () => {
		vi.mocked(connectivity.shouldTryServer).mockResolvedValue(true);
		vi.mocked(client.deleteTransaction).mockResolvedValue(undefined);
		writeRefCache('/api/v1/dashboard', { total_balance: 500, accounts: [] });
		writeRefCache('/api/v1/transactions?limit=20', [{ id: 'tx-1' }]);

		await deleteTransaction('tx-1');

		expect(client.deleteTransaction).toHaveBeenCalledWith('tx-1');
		expect(getOutboxEntries()).toHaveLength(0);
		expect(readRefCache('/api/v1/dashboard')).toBeNull();
		expect(readRefCache('/api/v1/transactions?limit=20')).toBeNull();
		expect(get(dataRefreshTick)).toBe(1);
		expect(sync.scheduleSyncOutbox).toHaveBeenCalled();
	});

	it('online transfer create/delete invalidate ledger and bump dataRefreshTick', async () => {
		vi.mocked(connectivity.shouldTryServer).mockResolvedValue(true);
		vi.mocked(client.createTransfer).mockResolvedValue({
			group_id: 'grp-1',
			from_account_id: 'a1',
			to_account_id: 'a2',
			amount: 100,
			amount_display: '1.00',
			commission: 0,
			commission_display: '0.00',
			description: null,
			transaction_date: '2026-10-13 12:00:00',
			kind: 'manual',
			legs: []
		} as never);
		vi.mocked(client.deleteTransfer).mockResolvedValue(undefined);
		writeRefCache('/api/v1/dashboard', { total_balance: 500, accounts: [] });
		writeRefCache('/api/v1/accounts?status=active', [{ id: 'a1', balance: 500 }]);

		await createTransfer({
			from_account_id: 'a1',
			to_account_id: 'a2',
			amount: '1.00',
			transaction_date: '2026-10-13 12:00:00'
		});
		expect(readRefCache('/api/v1/dashboard')).toBeNull();
		expect(readRefCache('/api/v1/accounts?status=active')).toBeNull();
		expect(get(dataRefreshTick)).toBe(1);

		writeRefCache('/api/v1/dashboard', { total_balance: 400, accounts: [] });
		await deleteTransfer('grp-1');
		expect(client.deleteTransfer).toHaveBeenCalledWith('grp-1');
		expect(readRefCache('/api/v1/dashboard')).toBeNull();
		expect(get(dataRefreshTick)).toBe(2);
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

	it('online create without offline queue invalidates ledger and bumps dataRefreshTick', async () => {
		vi.mocked(network.shouldUseOfflineQueue).mockReturnValue(false);
		writeRefCache('/api/v1/dashboard', {
			total_balance: 500,
			total_forecast: 500,
			accounts: [
				{
					id: 'a1',
					name: 'Карта',
					type: 'bank',
					balance: 500,
					balance_display: '5.00',
					forecast_balance: 500,
					forecast_display: '5.00',
					has_future_this_month: false,
					is_primary: true
				}
			],
			recent_transactions: [],
			debts_summary: {
				i_owe: 0,
				owed_to_me: 0,
				overdue_i_owe: 0,
				overdue_owed_to_me: 0,
				active_count: 0
			}
		});
		writeRefCache('/api/v1/transactions?limit=20', [{ id: 'old' }]);
		vi.mocked(client.createTransaction).mockResolvedValue({
			id: 'tx-new',
			account_id: 'a1',
			type: 'expense',
			kind: 'manual',
			amount: 100,
			amount_display: '1.00',
			description: null,
			merchant_id: null,
			merchant_name: null,
			category_id: 'c1',
			subcategory_id: null,
			transaction_date: '2026-10-13 12:00:00',
			created_at: '',
			updated_at: ''
		} as never);

		await createTransaction({
			account_id: 'a1',
			type: 'expense',
			amount: '1.00',
			category_id: 'c1',
			transaction_date: '2026-10-13 12:00:00'
		});

		expect(readRefCache('/api/v1/dashboard')).toBeNull();
		expect(readRefCache('/api/v1/transactions?limit=20')).toBeNull();
		expect(get(dataRefreshTick)).toBe(1);
	});

	it('online delete without offline queue invalidates ledger snapshots', async () => {
		vi.mocked(network.shouldUseOfflineQueue).mockReturnValue(false);
		writeRefCache('/api/v1/dashboard', { total_balance: 1, accounts: [] });
		writeRefCache('/api/v1/accounts?status=active', [
			{
				id: 'a1',
				name: 'Карта',
				type: 'bank',
				bank_id: null,
				initial_balance: 0,
				balance: 500,
				balance_display: '5.00',
				status: 'active',
				is_primary: false,
				created_at: '',
				updated_at: ''
			}
		]);
		vi.mocked(client.deleteTransaction).mockResolvedValue(undefined);

		await deleteTransaction('tx-1');

		expect(readRefCache('/api/v1/dashboard')).toBeNull();
		expect(readRefCache('/api/v1/accounts?status=active')).toBeNull();
		expect(get(dataRefreshTick)).toBe(1);
	});
});

describe('touchLedgerAfterTransactionMutation', () => {
	beforeEach(() => {
		resetRefCacheForTests();
	});

	it('drops dashboard so listAccounts enrich cannot overlay pre-mutation balances', () => {
		writeRefCache('/api/v1/dashboard', {
			total_balance: 500,
			total_forecast: 500,
			accounts: [
				{
					id: 'a1',
					name: 'Карта',
					type: 'bank',
					balance: 500,
					balance_display: '5.00',
					forecast_balance: 500,
					forecast_display: '5.00',
					has_future_this_month: false,
					is_primary: true
				}
			],
			recent_transactions: [],
			debts_summary: {
				i_owe: 0,
				owed_to_me: 0,
				overdue_i_owe: 0,
				overdue_owed_to_me: 0,
				active_count: 0
			}
		});
		writeRefCache('/api/v1/accounts/a1/balance', { id: 'a1', balance: 500 });
		const rows = [
			{
				id: 'a1',
				name: 'Карта',
				type: 'bank' as const,
				bank_id: null,
				initial_balance: 0,
				balance: 100,
				balance_display: '1.00',
				status: 'active' as const,
				is_primary: false,
				created_at: '',
				updated_at: ''
			}
		];
		expect(enrichAccountsWithCachedBalances(rows)[0]?.balance).toBe(500);

		touchLedgerAfterTransactionMutation();

		expect(readRefCache('/api/v1/dashboard')).toBeNull();
		expect(readRefCache('/api/v1/accounts/a1/balance')).toBeNull();
		expect(enrichAccountsWithCachedBalances(rows)[0]?.balance).toBe(100);
	});
});
