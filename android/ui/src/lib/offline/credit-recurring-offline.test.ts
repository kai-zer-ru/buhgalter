import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { Credit, RecurringOperation } from '$lib/api/client';
import {
	onCreditDeleted,
	onCreditUpdated,
	onRecurringCreated,
	onRecurringDeleted
} from '$lib/offline/ref-cache-mutations';
import { readRefCache, resetRefCacheForTests, writeRefCache } from '$lib/offline/ref-cache';
import {
	enqueueCreditMetaUpdate,
	enqueueCreditPay,
	getOutboxEntries,
	resetOutboxForTests
} from '$lib/offline/store';

vi.mock('$lib/offline/sync', () => ({
	notifyOutboxChanged: vi.fn(),
	scheduleSyncOutbox: vi.fn()
}));

beforeEach(() => {
	resetOutboxForTests();
	resetRefCacheForTests();
});

describe('credit outbox keys', () => {
	it('keeps pay and meta update as separate entries', () => {
		enqueueCreditMetaUpdate({
			action: 'update',
			credit_id: 'c1',
			bank_id: 'b1'
		});
		enqueueCreditPay({
			action: 'pay',
			credit_id: 'c1',
			amount: '10.00',
			payment_date: '2026-07-01'
		});
		enqueueCreditMetaUpdate({
			action: 'update',
			credit_id: 'c1',
			name: 'Новое'
		});

		const entries = getOutboxEntries();
		expect(entries).toHaveLength(2);
		expect(entries.find((e) => e.entityKey === 'credit:c1')?.payload).toMatchObject({
			action: 'update',
			name: 'Новое'
		});
		expect(entries.some((e) => e.entityKey.includes(':pay:'))).toBe(true);
	});
});

describe('credit / recurring ref-cache mutations', () => {
	it('onCreditUpdated writes detail and list caches', () => {
		const credit = {
			id: 'c1',
			status: 'active',
			name: 'Кредит'
		} as Credit;
		onCreditUpdated(credit);
		expect(readRefCache('/api/v1/credits/c1')).toEqual(credit);
		expect(readRefCache<Credit[]>('/api/v1/credits?status=active')?.[0]?.id).toBe('c1');
	});

	it('onCreditDeleted removes detail and lists', () => {
		const credit = { id: 'c1', status: 'active' } as Credit;
		writeRefCache('/api/v1/credits/c1', credit);
		writeRefCache('/api/v1/credits?status=active', [credit]);
		onCreditDeleted('c1');
		expect(readRefCache('/api/v1/credits/c1')).toBeNull();
		expect(readRefCache<Credit[]>('/api/v1/credits?status=active')).toEqual([]);
	});

	it('onRecurringCreated / Deleted patch list', () => {
		const item = { id: 'r1', amount: 100 } as RecurringOperation;
		onRecurringCreated(item);
		expect(readRefCache<RecurringOperation[]>('/api/v1/recurring-operations')?.[0]?.id).toBe('r1');
		onRecurringDeleted('r1');
		expect(readRefCache<RecurringOperation[]>('/api/v1/recurring-operations')).toEqual([]);
	});
});
