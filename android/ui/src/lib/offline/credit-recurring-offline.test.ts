import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { Credit, RecurringOperation, Subscription } from '$lib/api/client';
import {
	onCreditDeleted,
	onCreditUpdated,
	onRecurringCreated,
	onRecurringDeleted,
	onSubscriptionCreated,
	onSubscriptionDeleted
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
	it('onCreditUpdated writes detail and patches an existing list without dropping siblings', () => {
		const credit = {
			id: 'c1',
			status: 'active',
			name: 'Кредит'
		} as Credit;
		const sibling = { id: 'c2', status: 'active', name: 'Ипотека' } as Credit;
		writeRefCache('/api/v1/credits?status=active', [sibling, credit]);
		onCreditUpdated({ ...credit, name: 'Обновлён' });
		expect(readRefCache('/api/v1/credits/c1')).toMatchObject({ id: 'c1', name: 'Обновлён' });
		expect(readRefCache<Credit[]>('/api/v1/credits?status=active')?.map((c) => c.id)).toEqual([
			'c1',
			'c2'
		]);
	});

	it('onCreditUpdated does not seed a singleton list when lists were cleared', () => {
		const credit = {
			id: 'c1',
			status: 'active',
			name: 'Кредит'
		} as Credit;
		onCreditUpdated(credit);
		expect(readRefCache('/api/v1/credits/c1')).toEqual(credit);
		expect(readRefCache('/api/v1/credits?status=active')).toBeNull();
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

	it('onSubscriptionCreated / Deleted patch list', () => {
		const item = { id: 's1', amount: 100 } as Subscription;
		writeRefCache('/api/v1/subscriptions/summary?upcoming_days=14', { monthly_total: 1 });
		onSubscriptionCreated(item);
		expect(readRefCache<Subscription[]>('/api/v1/subscriptions')?.[0]?.id).toBe('s1');
		expect(readRefCache('/api/v1/subscriptions/summary?upcoming_days=14')).toBeNull();
		onSubscriptionDeleted('s1');
		expect(readRefCache<Subscription[]>('/api/v1/subscriptions')).toEqual([]);
	});
});
