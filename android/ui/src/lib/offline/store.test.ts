import { describe, expect, it, beforeEach } from 'vitest';
import {
	enqueueTransactionCreate,
	enqueueTransactionDelete,
	enqueueTransactionUpdate,
	enqueueTransferCreate,
	enqueueTransferDelete,
	enqueueTransferUpdate,
	enqueueCategoryCreate,
	enqueueCategoryDelete,
	enqueueCategoryUpdate,
	enqueueDebtCreate,
	enqueueDebtDelete,
	failedOutboxCount,
	getOutboxEntries,
	markOutboxFailed,
	pendingOutboxCount,
	resetOutboxForTests
} from '$lib/offline/store';
import { makeLocalKey } from '$lib/offline/types';

const txPayload = {
	account_id: 'acc-1',
	type: 'expense' as const,
	amount: '100.00',
	transaction_date: '2026-07-08 10:00:00'
};

const txPayload2 = { ...txPayload, amount: '200.00' };

const transferPayload = {
	from_account_id: 'a1',
	to_account_id: 'a2',
	amount: '500.00',
	transaction_date: '2026-07-08 11:00:00'
};

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
});

describe('outbox coalescing — local transaction', () => {
	it('create then edit keeps single POST payload', () => {
		const id = makeLocalKey();
		enqueueTransactionCreate(id, txPayload);
		enqueueTransactionUpdate(id, txPayload2);
		const entries = getOutboxEntries();
		expect(entries).toHaveLength(1);
		expect(entries[0].op).toBe('create');
		expect(entries[0].payload).toEqual(txPayload2);
	});

	it('create then delete removes entry', () => {
		const id = makeLocalKey();
		enqueueTransactionCreate(id, txPayload);
		enqueueTransactionDelete(id);
		expect(getOutboxEntries()).toHaveLength(0);
	});
});

describe('outbox coalescing — server transaction', () => {
	it('multiple updates collapse to one PUT', () => {
		enqueueTransactionUpdate('srv-1', txPayload);
		enqueueTransactionUpdate('srv-1', txPayload2);
		const entries = getOutboxEntries();
		expect(entries).toHaveLength(1);
		expect(entries[0].op).toBe('update');
		expect(entries[0].payload).toEqual(txPayload2);
	});

	it('update then delete becomes only DELETE', () => {
		enqueueTransactionUpdate('srv-1', txPayload);
		enqueueTransactionDelete('srv-1');
		const entries = getOutboxEntries();
		expect(entries).toHaveLength(1);
		expect(entries[0].op).toBe('delete');
	});

	it('delete supersedes pending update', () => {
		enqueueTransactionUpdate('srv-1', txPayload);
		enqueueTransactionDelete('srv-1');
		expect(getOutboxEntries()[0].op).toBe('delete');
	});
});

describe('outbox coalescing — transfer', () => {
	it('local create edit coalesces', () => {
		const id = makeLocalKey();
		enqueueTransferCreate(id, transferPayload);
		enqueueTransferUpdate(id, { ...transferPayload, amount: '600.00' });
		expect(getOutboxEntries()).toHaveLength(1);
		expect((getOutboxEntries()[0].payload as { amount: string }).amount).toBe('600.00');
	});

	it('local create delete removes', () => {
		const id = makeLocalKey();
		enqueueTransferCreate(id, transferPayload);
		enqueueTransferDelete(id);
		expect(getOutboxEntries()).toHaveLength(0);
	});
});

describe('outbox ordering', () => {
	it('preserves FIFO by seq', () => {
		const a = makeLocalKey();
		const b = makeLocalKey();
		enqueueTransactionCreate(a, txPayload);
		enqueueTransactionCreate(b, { ...txPayload, amount: '50.00' });
		const keys = getOutboxEntries().map((e) => e.entityKey);
		expect(keys).toEqual([a, b]);
	});
});

describe('outbox counts', () => {
	it('counts pending vs failed entries', () => {
		const id = makeLocalKey();
		enqueueTransactionCreate(id, txPayload);
		expect(pendingOutboxCount()).toBe(1);
		expect(failedOutboxCount()).toBe(0);
		markOutboxFailed(id, 'conflict');
		expect(pendingOutboxCount()).toBe(0);
		expect(failedOutboxCount()).toBe(1);
	});
});

describe('outbox coalescing — category', () => {
	it('local create then edit keeps single POST payload with type', () => {
		const id = makeLocalKey();
		enqueueCategoryCreate(id, categoryPayload);
		enqueueCategoryUpdate(id, { name: 'Транспорт', icon: 'car' });
		const entries = getOutboxEntries();
		expect(entries).toHaveLength(1);
		expect(entries[0].kind).toBe('category');
		expect(entries[0].op).toBe('create');
		expect(entries[0].payload).toEqual({
			name: 'Транспорт',
			icon: 'car',
			sort_order: undefined,
			type: 'expense'
		});
	});

	it('local create then delete removes entry', () => {
		const id = makeLocalKey();
		enqueueCategoryCreate(id, categoryPayload);
		enqueueCategoryDelete(id);
		expect(getOutboxEntries()).toHaveLength(0);
	});

	it('server update queues PUT', () => {
		enqueueCategoryUpdate('cat-1', { name: 'New', icon: 'star' });
		const entry = getOutboxEntries()[0];
		expect(entry.op).toBe('update');
		expect(entry.kind).toBe('category');
		expect(entry.isLocalOnly).toBe(false);
	});
});

describe('outbox coalescing — debt', () => {
	it('local create stays in outbox', () => {
		const id = makeLocalKey();
		enqueueDebtCreate(id, debtPayload);
		const entries = getOutboxEntries();
		expect(entries).toHaveLength(1);
		expect(entries[0].kind).toBe('debt');
		expect(entries[0].op).toBe('create');
	});

	it('local create delete removes entry', () => {
		const id = makeLocalKey();
		enqueueDebtCreate(id, debtPayload);
		enqueueDebtDelete(id);
		expect(getOutboxEntries()).toHaveLength(0);
	});

	it('server delete queues DELETE', () => {
		enqueueDebtDelete('debt-1');
		const entry = getOutboxEntries()[0];
		expect(entry.op).toBe('delete');
		expect(entry.kind).toBe('debt');
	});
});
