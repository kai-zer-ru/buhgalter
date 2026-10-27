import { describe, expect, it, beforeEach } from 'vitest';
import type { Transaction } from '$lib/api/client';
import {
	computeOutboxAccountDeltas,
	effectFromTransactionPayload,
	effectFromTransferPayload,
	resetOutboxDeltasCacheForTests
} from '$lib/offline/balance-overlay';
import {
	enqueueTransactionCreate,
	enqueueTransactionDelete,
	enqueueTransactionUpdate,
	enqueueTransferCreate,
	resetOutboxForTests
} from '$lib/offline/store';
import {
	indexTransactions,
	flushTransactionIndexForTests,
	resetTransactionIndexForTests
} from '$lib/offline/transaction-index';
import { makeLocalKey } from '$lib/offline/types';

const tz = 'Europe/Moscow';
const pastDate = '2026-07-01 10:00:00';

beforeEach(() => {
	resetOutboxForTests();
	resetTransactionIndexForTests();
	resetOutboxDeltasCacheForTests();
});

describe('effectFromTransactionPayload', () => {
	it('expense in the past reduces balance', () => {
		const effect = effectFromTransactionPayload(
			{
				account_id: 'acc-1',
				type: 'expense',
				amount: '100.00',
				transaction_date: pastDate
			},
			tz
		);
		expect(effect.balance['acc-1']).toBe(-10_000);
		expect(effect.forecast['acc-1']).toBe(-10_000);
	});
});

describe('computeOutboxAccountDeltas', () => {
	it('create offline expense adjusts balance', () => {
		enqueueTransactionCreate(makeLocalKey(), {
			account_id: 'acc-1',
			type: 'expense',
			amount: '50.00',
			transaction_date: pastDate
		});
		const deltas = computeOutboxAccountDeltas(tz);
		expect(deltas.balance['acc-1']).toBe(-5_000);
	});

	it('update server transaction replaces effect', () => {
		const serverTx: Transaction = {
			id: 'tx-1',
			account_id: 'acc-1',
			type: 'expense',
			kind: 'manual',
			amount: 10_000,
			amount_display: '100.00',
			description: null,
			category_id: null,
			subcategory_id: null,
			transaction_date: pastDate,
			created_at: pastDate,
			updated_at: pastDate
		};
		indexTransactions([serverTx]);
		flushTransactionIndexForTests();
		enqueueTransactionUpdate('tx-1', {
			account_id: 'acc-1',
			type: 'expense',
			amount: '30.00',
			transaction_date: pastDate
		});
		const deltas = computeOutboxAccountDeltas(tz);
		expect(deltas.balance['acc-1']).toBe(7_000);
	});

	it('delete server transaction reverts effect', () => {
		indexTransactions([
			{
				id: 'tx-2',
				account_id: 'acc-2',
				type: 'income',
				kind: 'manual',
				amount: 20_000,
				amount_display: '200.00',
				description: null,
				category_id: null,
				subcategory_id: null,
				transaction_date: pastDate,
				created_at: pastDate,
				updated_at: pastDate
			}
		]);
		flushTransactionIndexForTests();
		enqueueTransactionDelete('tx-2');
		const deltas = computeOutboxAccountDeltas(tz);
		expect(deltas.balance['acc-2']).toBe(-20_000);
	});
});

describe('effectFromTransferPayload FX', () => {
	it('credits the destination with to_amount, not the source amount', () => {
		const effect = effectFromTransferPayload(
			{
				from_account_id: 'rub',
				to_account_id: 'gbp',
				amount: '9000.00',
				to_amount: '100.00',
				transaction_date: pastDate
			},
			tz
		);
		expect(effect.balance['rub']).toBe(-900_000);
		expect(effect.balance['gbp']).toBe(10_000);
	});

	it('queues FX overlay via outbox', () => {
		enqueueTransferCreate(makeLocalKey(), {
			from_account_id: 'rub',
			to_account_id: 'gbp',
			amount: '9000.00',
			to_amount: '100.00',
			transaction_date: pastDate
		});
		const deltas = computeOutboxAccountDeltas(tz);
		expect(deltas.balance['rub']).toBe(-900_000);
		expect(deltas.balance['gbp']).toBe(10_000);
	});
});
