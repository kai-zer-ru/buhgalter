import { describe, expect, it } from 'vitest';
import { pickCategorySuggestion } from './category-suggest';
import type { Transaction } from '$lib/api/client';

function tx(
	partial: Partial<Transaction> & {
		category_id: string | null;
		subcategory_id?: string | null;
	}
): Transaction {
	return {
		id: partial.id ?? 't1',
		type: partial.type ?? 'expense',
		amount: 100,
		amount_display: '1.00',
		account_id: 'a1',
		transaction_date: '2024-01-01 12:00:00',
		kind: 'manual',
		category_id: partial.category_id,
		category_is_system: partial.category_is_system ?? false,
		subcategory_id: partial.subcategory_id ?? null,
		merchant_id: 'm1',
		description: null,
		...partial
	} as Transaction;
}

describe('pickCategorySuggestion', () => {
	it('returns null for empty / no usable categories', () => {
		expect(pickCategorySuggestion([])).toBeNull();
		expect(
			pickCategorySuggestion([
				tx({ category_id: null }),
				tx({ id: 't2', category_id: 'sys', category_is_system: true })
			])
		).toBeNull();
	});

	it('picks majority pair', () => {
		const result = pickCategorySuggestion([
			tx({ id: '1', category_id: 'food', subcategory_id: 'cafe' }),
			tx({ id: '2', category_id: 'food', subcategory_id: 'cafe' }),
			tx({ id: '3', category_id: 'transport', subcategory_id: null })
		]);
		expect(result).toEqual({ categoryId: 'food', subcategoryId: 'cafe' });
	});

	it('on tie prefers more recent (earlier in date_desc list)', () => {
		const result = pickCategorySuggestion([
			tx({ id: '1', category_id: 'food', subcategory_id: 'a' }),
			tx({ id: '2', category_id: 'transport', subcategory_id: 'b' })
		]);
		expect(result).toEqual({ categoryId: 'food', subcategoryId: 'a' });
	});

	it('treats missing subcategory as its own pair', () => {
		const result = pickCategorySuggestion([
			tx({ id: '1', category_id: 'food', subcategory_id: null }),
			tx({ id: '2', category_id: 'food', subcategory_id: null }),
			tx({ id: '3', category_id: 'food', subcategory_id: 'cafe' })
		]);
		expect(result).toEqual({ categoryId: 'food', subcategoryId: undefined });
	});

	it('filters by transaction type', () => {
		const txs = [
			tx({ id: '1', type: 'income', category_id: 'salary' }),
			tx({ id: '2', category_id: 'food', subcategory_id: 'cafe' })
		];
		expect(pickCategorySuggestion(txs, 'expense')).toEqual({
			categoryId: 'food',
			subcategoryId: 'cafe'
		});
		expect(pickCategorySuggestion(txs, 'income')).toEqual({ categoryId: 'salary' });
	});
});
