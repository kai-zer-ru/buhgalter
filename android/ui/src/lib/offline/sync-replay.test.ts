import { beforeEach, describe, expect, it, vi } from 'vitest';
import { syncOutbox } from '$lib/offline/sync';
import {
	enqueueCategoryCreate,
	enqueueCategoryDelete,
	enqueueDebtCreate,
	enqueueDebtDelete,
	enqueueAccountCreate,
	enqueueAccountArchive,
	enqueueBudgetCreate,
	enqueueBudgetDelete,
	getOutboxEntries,
	resetOutboxForTests
} from '$lib/offline/store';
import { makeLocalKey } from '$lib/offline/types';
import * as client from '$lib/api/client';
import * as connectivity from '$lib/offline/server-connectivity';

vi.mock('$lib/api/cache', () => ({
	invalidateApiCache: vi.fn()
}));

vi.mock('$lib/api/client', () => ({
	createCategory: vi.fn().mockResolvedValue({ id: 'srv-c1' }),
	createDebt: vi.fn().mockResolvedValue({ id: 'srv-d1' }),
	createAccount: vi.fn().mockResolvedValue({ id: 'srv-a1' }),
	createBudget: vi.fn().mockResolvedValue({ id: 'srv-b1' }),
	deleteCategory: vi.fn().mockResolvedValue(undefined),
	deleteDebt: vi.fn().mockResolvedValue(undefined),
	deleteBudget: vi.fn().mockResolvedValue(undefined),
	createTransaction: vi.fn(),
	createTransfer: vi.fn(),
	deleteTransaction: vi.fn(),
	deleteTransfer: vi.fn(),
	updateTransaction: vi.fn(),
	updateTransfer: vi.fn(),
	updateCategory: vi.fn(),
	updateAccount: vi.fn().mockResolvedValue({ id: 'srv-a1' }),
	updateBudget: vi.fn().mockResolvedValue({ id: 'srv-b1' }),
	archiveAccount: vi.fn().mockResolvedValue({ id: 'srv-a1', status: 'archived' }),
	unarchiveAccount: vi.fn().mockResolvedValue({ id: 'srv-a1', status: 'active' }),
	ApiError: class ApiError extends Error {
		constructor(
			public code: string,
			message: string,
			public status: number
		) {
			super(message);
		}
	},
	isTransientHttpError: () => false
}));

vi.mock('$lib/offline/server-connectivity', async (importOriginal) => {
	const actual = await importOriginal<typeof connectivity>();
	return {
		...actual,
		shouldTryServer: vi.fn().mockResolvedValue(true),
		markServerOnline: vi.fn(),
		markServerOffline: vi.fn()
	};
});

const categoryPayload = {
	name: 'Еда',
	type: 'expense' as const,
	icon: 'food'
};

const debtPayload = {
	debtor_name: 'Иван',
	direction: 'lent' as const,
	amount: '1000.00',
	debt_date: '2026-07-08 10:00:00',
	due_date: '2026-07-15 23:59:59',
	affects_balance: false
};

beforeEach(() => {
	resetOutboxForTests();
	vi.clearAllMocks();
});

describe('syncOutbox replay — category and debt', () => {
	it('replays category create', async () => {
		const id = makeLocalKey();
		enqueueCategoryCreate(id, categoryPayload);

		await syncOutbox();

		expect(client.createCategory).toHaveBeenCalledWith(categoryPayload);
		expect(getOutboxEntries()).toHaveLength(0);
	});

	it('replays category delete', async () => {
		enqueueCategoryDelete('cat-server-1');

		await syncOutbox();

		expect(client.deleteCategory).toHaveBeenCalledWith('cat-server-1');
		expect(getOutboxEntries()).toHaveLength(0);
	});

	it('replays debt create', async () => {
		const id = makeLocalKey();
		enqueueDebtCreate(id, debtPayload);

		await syncOutbox();

		expect(client.createDebt).toHaveBeenCalledWith(debtPayload);
		expect(getOutboxEntries()).toHaveLength(0);
	});

	it('replays debt delete', async () => {
		enqueueDebtDelete('debt-server-1');

		await syncOutbox();

		expect(client.deleteDebt).toHaveBeenCalledWith('debt-server-1');
		expect(getOutboxEntries()).toHaveLength(0);
	});

	it('replays account create and archive', async () => {
		const id = makeLocalKey();
		enqueueAccountCreate(id, {
			name: 'Cash',
			type: 'cash',
			initial_balance: '10.00'
		});
		await syncOutbox();
		expect(client.createAccount).toHaveBeenCalled();
		expect(getOutboxEntries()).toHaveLength(0);

		enqueueAccountArchive('acc-1', 'acc-2');
		await syncOutbox();
		expect(client.archiveAccount).toHaveBeenCalledWith('acc-1', 'acc-2');
		expect(getOutboxEntries()).toHaveLength(0);
	});

	it('replays budget create and delete', async () => {
		const id = makeLocalKey();
		enqueueBudgetCreate(id, {
			name: 'Food',
			scope: 'all_expense',
			amount: '1000.00',
			month: '2026-07'
		});
		await syncOutbox();
		expect(client.createBudget).toHaveBeenCalledWith(
			{
				name: 'Food',
				scope: 'all_expense',
				amount: '1000.00'
			},
			'2026-07'
		);
		expect(getOutboxEntries()).toHaveLength(0);

		enqueueBudgetDelete('b1');
		await syncOutbox();
		expect(client.deleteBudget).toHaveBeenCalledWith('b1');
	});
});
