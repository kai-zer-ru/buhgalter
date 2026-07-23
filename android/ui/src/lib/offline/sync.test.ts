import { describe, expect, it, beforeEach } from 'vitest';
import { get } from 'svelte/store';
import { resetOutboxForTests, enqueueTransactionCreate } from '$lib/offline/store';
import { localDataTick, notifyOutboxChanged } from '$lib/offline/sync';
import { makeLocalKey } from '$lib/offline/types';

const txPayload = {
	account_id: 'acc-1',
	type: 'expense' as const,
	amount: '100.00',
	transaction_date: '2026-07-08 10:00:00'
};

beforeEach(() => {
	resetOutboxForTests();
	localDataTick.set(0);
});

describe('notifyOutboxChanged', () => {
	it('bumps localDataTick without a server pull', () => {
		notifyOutboxChanged();
		expect(get(localDataTick)).toBe(1);
	});

	it('runs when outbox store changes', async () => {
		const id = makeLocalKey();
		enqueueTransactionCreate(id, txPayload);
		await new Promise((r) => setTimeout(r, 0));
		expect(get(localDataTick)).toBeGreaterThan(0);
	});
});
