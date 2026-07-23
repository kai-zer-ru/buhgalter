import { beforeEach, describe, expect, it, vi } from 'vitest';
import {
	mergeMetaAccounts,
	mergeMetaCategories,
	mergeOutboxTransactions,
	refreshMergeMeta
} from '$lib/offline/merge';
import { enqueueTransactionCreate, makeLocalKey, resetOutboxForTests } from '$lib/offline/store';
import * as client from '$lib/api/client';

vi.mock('$lib/api/client', () => ({
	getUIMeta: vi.fn()
}));

const uiMeta = {
	accounts: [{ id: 'cash', name: 'Наличные', type: 'cash', status: 'active' as const }],
	banks: [],
	expense_categories: [
		{
			id: 'cat-1',
			name: 'Еда',
			type: 'expense' as const,
			icon: 'food',
			sort_order: 0,
			is_primary: false,
			is_system: false,
			subcategory_count: 0,
			created_at: '2026-01-01T00:00:00Z'
		}
	],
	income_categories: [],
	debtors: [],
	active_credits: [],
	closed_credits: []
};

beforeEach(() => {
	resetOutboxForTests();
	vi.mocked(client.getUIMeta).mockResolvedValue(uiMeta);
});

describe('merge meta', () => {
	it('refreshMergeMeta exposes accounts and categories for pending lookup', async () => {
		await refreshMergeMeta();

		expect(mergeMetaAccounts()[0]).toMatchObject({ id: 'cash', name: 'Наличные' });
		expect(mergeMetaCategories()).toHaveLength(1);
		expect(mergeMetaCategories()[0]?.name).toBe('Еда');
	});

	it('mergeOutboxTransactions prepends pending rows', async () => {
		await refreshMergeMeta();
		const id = makeLocalKey();
		enqueueTransactionCreate(id, {
			account_id: 'cash',
			type: 'expense',
			amount: '50.00',
			transaction_date: '2026-07-09 12:00:00'
		});

		const merged = mergeOutboxTransactions([]);
		expect(merged).toHaveLength(1);
		expect(merged[0]?.id).toBe(id);
	});
});
