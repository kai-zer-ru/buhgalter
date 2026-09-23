import { beforeEach, describe, expect, it } from 'vitest';
import { applyTransactionChanges } from '$lib/offline/transaction-changes';
import { resetRefCacheForTests, writeRefCache, readRefCache } from '$lib/offline/ref-cache';
import {
	lookupServerTransaction,
	resetTransactionIndexForTests
} from '$lib/offline/transaction-index';
import { HOME_PAST_TRANSACTIONS_PATH } from '$lib/api/transactions-path';
import type { Transaction, TransactionList } from '$lib/api/client';

function tx(partial: Partial<Transaction> & { id: string }): Transaction {
	return {
		account_id: 'acc-1',
		type: 'expense',
		kind: 'manual',
		amount: 1000,
		amount_display: '10.00',
		description: null,
		category_id: 'cat-1',
		subcategory_id: null,
		transaction_date: '2026-01-15 12:00:00',
		created_at: '2026-01-15 12:00:00',
		updated_at: '2026-01-15 12:00:00',
		...partial
	};
}

function listOf(items: Transaction[]): TransactionList {
	return { data: items, meta: { page: 1, limit: 10, total: items.length } };
}

beforeEach(() => {
	resetRefCacheForTests();
	resetTransactionIndexForTests();
});

describe('applyTransactionChanges', () => {
	it('removes a deleted old operation from every cached list', () => {
		const old = tx({ id: 'old-1', transaction_date: '2025-04-01 10:00:00' });
		writeRefCache(HOME_PAST_TRANSACTIONS_PATH, listOf([old]));
		writeRefCache('/api/v1/transactions?account_id=acc-1&page=1', listOf([old]));
		applyTransactionChanges([
			{
				id: 0,
				action: 'upsert',
				entity_id: old.id,
				occurred_at: '2025-04-01 10:00:00',
				transaction: old
			}
		]);

		applyTransactionChanges([
			{ id: 1, action: 'deleted', entity_id: 'old-1', occurred_at: '2026-09-07 10:00:00' }
		]);

		expect(readRefCache<TransactionList>(HOME_PAST_TRANSACTIONS_PATH)?.data).toEqual([]);
		expect(
			readRefCache<TransactionList>('/api/v1/transactions?account_id=acc-1&page=1')?.data
		).toEqual([]);
		expect(lookupServerTransaction('old-1')).toBeNull();
	});

	it('inserts a missing operation into matching page-1 lists', () => {
		const existing = tx({ id: 'a', transaction_date: '2026-09-01 10:00:00' });
		writeRefCache(HOME_PAST_TRANSACTIONS_PATH, listOf([existing]));
		const added = tx({
			id: 'split-2',
			transaction_date: '2026-09-06 10:00:00',
			amount: 500,
			amount_display: '5.00'
		});

		applyTransactionChanges([
			{
				id: 2,
				action: 'upsert',
				entity_id: added.id,
				occurred_at: '2026-09-07 10:00:00',
				transaction: added
			}
		]);

		const home = readRefCache<TransactionList>(HOME_PAST_TRANSACTIONS_PATH);
		expect(home?.data.map((row) => row.id)).toEqual(['split-2', 'a']);
		expect(lookupServerTransaction('split-2')?.amount).toBe(500);
	});

	it('does not insert into page 2 lists', () => {
		const path = '/api/v1/transactions?limit=10&page=2&sort=date_desc';
		writeRefCache(path, listOf([tx({ id: 'page2' })]));
		applyTransactionChanges([
			{
				id: 3,
				action: 'upsert',
				entity_id: 'new-1',
				occurred_at: '2026-09-07 10:00:00',
				transaction: tx({ id: 'new-1' })
			}
		]);
		expect(readRefCache<TransactionList>(path)?.data.map((row) => row.id)).toEqual(['page2']);
	});

	it('replaces an edited operation in place', () => {
		const original = tx({ id: 'edit-1', amount: 1000, amount_display: '10.00' });
		writeRefCache(HOME_PAST_TRANSACTIONS_PATH, listOf([original]));
		applyTransactionChanges([
			{
				id: 4,
				action: 'upsert',
				entity_id: 'edit-1',
				occurred_at: '2026-09-07 10:00:00',
				transaction: tx({ id: 'edit-1', amount: 2500, amount_display: '25.00' })
			}
		]);
		expect(readRefCache<TransactionList>(HOME_PAST_TRANSACTIONS_PATH)?.data[0]?.amount).toBe(2500);
	});

	it('does not recount an older operation already covered by meta.total', () => {
		const path = '/api/v1/transactions?kind=manual&limit=1&page=1&sort=date_desc';
		const recent = tx({ id: 'new', transaction_date: '2026-09-01 10:00:00' });
		writeRefCache(path, { data: [recent], meta: { page: 1, limit: 1, total: 10 } });
		const older = tx({ id: 'old', transaction_date: '2020-01-01 10:00:00' });
		applyTransactionChanges([
			{
				id: 8,
				action: 'upsert',
				entity_id: older.id,
				occurred_at: '2020-01-01 10:00:00',
				transaction: older
			}
		]);
		const cached = readRefCache<TransactionList>(path);
		expect(cached?.meta.total).toBe(10);
		expect(cached?.data.map((row) => row.id)).toEqual(['new']);
	});

	it('counts a transfer once and skips the incoming leg on the journal list', () => {
		writeRefCache(HOME_PAST_TRANSACTIONS_PATH, {
			data: [],
			meta: { page: 1, limit: 10, total: 5 }
		});
		applyTransactionChanges([
			{
				id: 9,
				action: 'upsert',
				entity_id: 'out',
				occurred_at: '2026-09-07 10:00:00',
				transaction: tx({
					id: 'out',
					type: 'transfer',
					transfer_group_id: 'g',
					transfer_is_out: true,
					transaction_date: '2026-09-07 10:00:00'
				})
			},
			{
				id: 10,
				action: 'upsert',
				entity_id: 'in',
				occurred_at: '2026-09-07 10:00:01',
				transaction: tx({
					id: 'in',
					type: 'transfer',
					transfer_group_id: 'g',
					transfer_is_out: false,
					transaction_date: '2026-09-07 10:00:01'
				})
			}
		]);
		const cached = readRefCache<TransactionList>(HOME_PAST_TRANSACTIONS_PATH);
		expect(cached?.data.map((row) => row.id)).toEqual(['out']);
		expect(cached?.meta.total).toBe(6);
	});

	it('keeps the incoming transfer leg on that account list', () => {
		const path = '/api/v1/transactions?account_id=acc-2&limit=10&page=1&sort=date_desc';
		writeRefCache(path, { data: [], meta: { page: 1, limit: 10, total: 0 } });
		applyTransactionChanges([
			{
				id: 11,
				action: 'upsert',
				entity_id: 'in',
				occurred_at: '2026-09-07 10:00:00',
				transaction: tx({
					id: 'in',
					account_id: 'acc-2',
					type: 'transfer',
					transfer_group_id: 'g',
					transfer_is_out: false
				})
			}
		]);
		const cached = readRefCache<TransactionList>(path);
		expect(cached?.data.map((row) => row.id)).toEqual(['in']);
		expect(cached?.meta.total).toBe(1);
	});

	it('does not add transfer commission expense to visible lists', () => {
		writeRefCache(HOME_PAST_TRANSACTIONS_PATH, listOf([]));
		applyTransactionChanges([
			{
				id: 5,
				action: 'upsert',
				entity_id: 'fee-1',
				occurred_at: '2026-09-07 10:00:00',
				transaction: tx({
					id: 'fee-1',
					type: 'expense',
					transfer_group_id: 'g1',
					category_is_system: true
				})
			}
		]);
		expect(readRefCache<TransactionList>(HOME_PAST_TRANSACTIONS_PATH)?.data).toEqual([]);
	});
});
