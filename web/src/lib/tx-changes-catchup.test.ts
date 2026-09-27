import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import {
	catchUpAfterReconnect,
	resetTxChangesCursorForTests,
	seedTransactionChangesCursor,
	TX_CHANGES_CURSOR_KEY
} from './tx-changes-catchup';

const listMock = vi.fn();

vi.mock('$lib/api/client', () => ({
	listTransactionChanges: (...args: unknown[]) => listMock(...args)
}));

vi.mock('$lib/ref-cache', () => ({
	clearRefCache: vi.fn(),
	notifyRealtimeInvalidate: vi.fn()
}));

describe('tx-changes-catchup', () => {
	beforeEach(() => {
		resetTxChangesCursorForTests();
		listMock.mockReset();
		localStorage.clear();
	});

	afterEach(() => {
		vi.clearAllMocks();
	});

	it('seeds cursor to tip without requiring prior cursor', async () => {
		listMock.mockResolvedValueOnce({
			server_time: '',
			since_id: 42,
			has_more: false,
			changes: [{ id: 1, action: 'upsert', entity_id: 't1', occurred_at: '' }]
		});
		await seedTransactionChangesCursor();
		expect(localStorage.getItem(TX_CHANGES_CURSOR_KEY)).toBe('42');
		expect(listMock).toHaveBeenCalledOnce();
	});

	it('catch-up advances cursor and invalidates', async () => {
		const { clearRefCache, notifyRealtimeInvalidate } = await import('$lib/ref-cache');
		localStorage.setItem(TX_CHANGES_CURSOR_KEY, '10');
		listMock.mockResolvedValueOnce({
			server_time: '',
			since_id: 15,
			has_more: false,
			changes: [{ id: 11, action: 'upsert', entity_id: 't2', occurred_at: '' }]
		});
		await catchUpAfterReconnect();
		expect(localStorage.getItem(TX_CHANGES_CURSOR_KEY)).toBe('15');
		expect(clearRefCache).toHaveBeenCalled();
		expect(notifyRealtimeInvalidate).toHaveBeenCalled();
	});
});
