import { describe, expect, it, beforeEach } from 'vitest';
import type { Category } from '$lib/api/client';
import {
	lookupPendingTransaction,
	lookupPendingTransferLegs,
	mergeTransactionLists,
	pendingTransactions
} from '$lib/offline/pending-display';
import {
	enqueueTransactionCreate,
	enqueueTransactionDelete,
	enqueueTransferCreate,
	makeLocalKey,
	resetOutboxForTests
} from '$lib/offline/store';

const accounts = [
	{ id: 'cash', name: 'Наличные' },
	{ id: 'wb', name: 'WB' }
];

const categories: Category[] = [
	{
		id: 'cat-transfer',
		name: 'Перевод',
		type: 'expense',
		icon: 'transfer',
		sort_order: 0,
		is_primary: false,
		is_system: true,
		subcategory_count: 0,
		created_at: '2026-01-01T00:00:00Z'
	}
];

describe('pendingTransactions transfer category', () => {
	beforeEach(() => {
		resetOutboxForTests();
	});

	it('uses system transfer category instead of raw type', () => {
		enqueueTransferCreate(makeLocalKey(), {
			from_account_id: 'cash',
			to_account_id: 'wb',
			amount: '10000.00',
			transaction_date: '2026-07-09 10:26:00'
		});

		const [tx] = pendingTransactions(accounts, categories);
		expect(tx.category_id).toBe('cat-transfer');
		expect(tx.category_icon).toBe('transfer');
		expect(tx.type).toBe('transfer');
	});
});

describe('lookupPendingTransaction', () => {
	beforeEach(() => {
		resetOutboxForTests();
	});

	it('resolves local transaction by entity key', () => {
		const id = makeLocalKey();
		enqueueTransactionCreate(id, {
			account_id: 'cash',
			type: 'expense',
			amount: '100.00',
			transaction_date: '2026-07-09 10:00:00'
		});

		const tx = lookupPendingTransaction(id, accounts, categories);
		expect(tx?.id).toBe(id);
		expect(tx?.amount_display).toBe('100.00');
	});

	it('returns null after pending delete', () => {
		const id = makeLocalKey();
		enqueueTransactionCreate(id, {
			account_id: 'cash',
			type: 'expense',
			amount: '100.00',
			transaction_date: '2026-07-09 10:00:00'
		});
		enqueueTransactionDelete(id);

		expect(lookupPendingTransaction(id, accounts, categories)).toBeNull();
	});

	it('resolves local transfer legs by group id', () => {
		const id = makeLocalKey();
		enqueueTransferCreate(id, {
			from_account_id: 'cash',
			to_account_id: 'wb',
			amount: '50.00',
			transaction_date: '2026-07-09 11:00:00'
		});

		const legs = lookupPendingTransferLegs(id, accounts, categories);
		expect(legs).toHaveLength(2);
		expect(legs[0]?.transfer_is_out).toBe(true);
		expect(legs[1]?.transfer_is_out).toBe(false);
		expect(lookupPendingTransaction(id, accounts, categories)?.transfer_group_id).toBe(id);
	});
});

describe('mergeTransactionLists sort', () => {
	beforeEach(() => {
		resetOutboxForTests();
	});

	it('orders pending + server rows newest-first by transaction_date', () => {
		const older = makeLocalKey();
		const newer = makeLocalKey();
		enqueueTransactionCreate(older, {
			account_id: 'cash',
			type: 'expense',
			amount: '19.99',
			transaction_date: '2026-07-18 11:10:00'
		});
		enqueueTransactionCreate(newer, {
			account_id: 'cash',
			type: 'expense',
			amount: '50.00',
			transaction_date: '2026-07-18 15:55:00'
		});

		const server = [
			{
				id: 'srv-old',
				account_id: 'cash',
				account_name: 'Наличные',
				account_status: 'active' as const,
				type: 'expense' as const,
				kind: 'manual' as const,
				amount: 22000,
				amount_display: '220.00',
				description: null,
				category_id: null,
				category_name: null,
				category_icon: null,
				category_is_system: false,
				subcategory_id: null,
				subcategory_name: null,
				transaction_date: '2026-07-17 23:20:00',
				created_at: '2026-07-17T23:20:00Z',
				updated_at: '2026-07-17T23:20:00Z',
				credit_payment_linked: false,
				deletable: true
			}
		];

		const merged = mergeTransactionLists(server, accounts, categories);
		expect(merged.map((t) => t.transaction_date)).toEqual([
			'2026-07-18 15:55:00',
			'2026-07-18 11:10:00',
			'2026-07-17 23:20:00'
		]);
	});
});
