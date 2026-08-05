import { beforeEach, describe, expect, it, vi } from 'vitest';
import { syncOutbox } from '$lib/offline/sync';
import {
	enqueueCategoryCreate,
	enqueueCategoryDelete,
	enqueueDebtCreate,
	enqueueDebtDelete,
	enqueueDebtSettle,
	enqueueAccountCreate,
	enqueueAccountArchive,
	enqueueBudgetCreate,
	enqueueBudgetDelete,
	enqueueCreditMetaUpdate,
	enqueueCreditPay,
	enqueueRecurringCreate,
	enqueueRecurringDelete,
	enqueueSubscriptionCreate,
	enqueueSubscriptionDelete,
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
	createRecurringOperation: vi.fn().mockResolvedValue({ id: 'srv-r1' }),
	createSubscription: vi.fn().mockResolvedValue({ id: 'srv-s1' }),
	deleteCategory: vi.fn().mockResolvedValue(undefined),
	deleteDebt: vi.fn().mockResolvedValue(undefined),
	deleteBudget: vi.fn().mockResolvedValue(undefined),
	deleteRecurringOperation: vi.fn().mockResolvedValue(undefined),
	deleteSubscription: vi.fn().mockResolvedValue(undefined),
	createTransaction: vi.fn(),
	createTransfer: vi.fn(),
	deleteTransaction: vi.fn(),
	deleteTransfer: vi.fn(),
	updateTransaction: vi.fn(),
	updateTransfer: vi.fn(),
	updateCategory: vi.fn(),
	updateAccount: vi.fn().mockResolvedValue({ id: 'srv-a1' }),
	updateBudget: vi.fn().mockResolvedValue({ id: 'srv-b1' }),
	updateRecurringOperation: vi.fn().mockResolvedValue({ id: 'srv-r1' }),
	updateSubscription: vi.fn().mockResolvedValue({ id: 'srv-s1' }),
	updateCredit: vi.fn().mockResolvedValue({ id: 'c1' }),
	addCreditPayment: vi.fn().mockResolvedValue({ id: 'c1' }),
	completeCredit: vi.fn(),
	deleteCredit: vi.fn(),
	deleteCreditPayment: vi.fn(),
	updateCreditSchedule: vi.fn(),
	settleDebt: vi.fn().mockResolvedValue({ id: 'srv-d1', is_settled: true }),
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

describe('syncOutbox replay — credit, settle, recurring', () => {
	it('replays debt settle', async () => {
		enqueueDebtSettle('debt-1', {
			action: 'settle',
			settled_at: '2026-07-10 12:00:00',
			affects_balance: false
		});
		await syncOutbox();
		expect(client.settleDebt).toHaveBeenCalledWith('debt-1', {
			amount: undefined,
			settled_at: '2026-07-10 12:00:00',
			affects_balance: false,
			account_id: undefined
		});
		expect(getOutboxEntries()).toHaveLength(0);
	});

	it('replays credit meta update and pay as separate keys', async () => {
		enqueueCreditMetaUpdate({
			action: 'update',
			credit_id: 'c1',
			name: 'Ипотека'
		});
		enqueueCreditPay({
			action: 'pay',
			credit_id: 'c1',
			amount: '1000.00',
			payment_date: '2026-07-10'
		});
		expect(getOutboxEntries()).toHaveLength(2);

		await syncOutbox();

		expect(client.updateCredit).toHaveBeenCalledWith('c1', { name: 'Ипотека' });
		expect(client.addCreditPayment).toHaveBeenCalledWith('c1', {
			amount: '1000.00',
			payment_date: '2026-07-10',
			account_id: undefined
		});
		expect(getOutboxEntries()).toHaveLength(0);
	});

	it('replays recurring create and delete', async () => {
		const id = makeLocalKey();
		enqueueRecurringCreate(id, {
			type: 'expense',
			amount: '500.00',
			account_id: 'a1',
			category_id: 'cat1',
			period: 'month',
			start_date: '2026-07-01'
		});
		await syncOutbox();
		expect(client.createRecurringOperation).toHaveBeenCalled();
		expect(getOutboxEntries()).toHaveLength(0);

		enqueueRecurringDelete('r1');
		await syncOutbox();
		expect(client.deleteRecurringOperation).toHaveBeenCalledWith('r1');
	});

	it('replays subscription create and delete', async () => {
		const id = makeLocalKey();
		enqueueSubscriptionCreate(id, {
			name: 'Netflix',
			amount: '999.00',
			account_id: 'a1',
			period: 'month',
			start_date: '2026-07-01'
		});
		await syncOutbox();
		expect(client.createSubscription).toHaveBeenCalled();
		expect(getOutboxEntries()).toHaveLength(0);

		enqueueSubscriptionDelete('s1');
		await syncOutbox();
		expect(client.deleteSubscription).toHaveBeenCalledWith('s1');
	});
});
