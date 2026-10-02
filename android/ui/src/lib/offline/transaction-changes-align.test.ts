import { beforeEach, describe, expect, it, vi } from 'vitest';

const listTransactionChanges = vi.fn();

vi.mock('$lib/api/client', () => ({
	listTransactionChanges: (...args: unknown[]) => listTransactionChanges(...args)
}));

import {
	alignTransactionChangesCursor,
	resetTransactionChangesCursorForTests
} from './transaction-changes';
import { readRefCache, resetRefCacheForTests } from './ref-cache';

beforeEach(() => {
	resetRefCacheForTests();
	resetTransactionChangesCursorForTests();
	listTransactionChanges.mockReset();
});

describe('alignTransactionChangesCursor', () => {
	it('uses watermark when server returns empty changes', async () => {
		listTransactionChanges.mockResolvedValue({
			server_time: '2026-01-01',
			since_id: 42_000,
			has_more: false,
			changes: []
		});
		await alignTransactionChangesCursor();
		expect(listTransactionChanges).toHaveBeenCalledWith({ watermark: true });
		expect(readRefCache('__internal_tx_changes_cursor__')?.since_id).toBe(42_000);
	});

	it('seeks without applying when watermark response includes changes (legacy server)', async () => {
		listTransactionChanges
			.mockResolvedValueOnce({
				server_time: '2026-01-01',
				since_id: 500,
				has_more: true,
				changes: [{ id: 1, action: 'upsert', entity_id: 'tx-1', occurred_at: '' }]
			})
			.mockResolvedValueOnce({
				server_time: '2026-01-01',
				since_id: 1000,
				has_more: false,
				changes: []
			});
		await alignTransactionChangesCursor();
		expect(listTransactionChanges).toHaveBeenNthCalledWith(1, { watermark: true });
		expect(listTransactionChanges).toHaveBeenNthCalledWith(2, { since_id: 0, limit: 500 });
		expect(readRefCache('__internal_tx_changes_cursor__')?.since_id).toBe(1000);
	});
});
