import { describe, expect, it, vi } from 'vitest';
import { categoryDisplayLabel, duplicateCategoryNames } from './category-label';

vi.mock('$lib/i18n', () => ({
	tr: (key: string) => (key === 'transactions.type.income' ? 'Доход' : 'Расход')
}));

describe('categoryDisplayLabel', () => {
	it('adds type suffix for duplicate credit category names', () => {
		const dup = duplicateCategoryNames([
			{ name: 'Кредиты', type: 'income' },
			{ name: 'Кредиты', type: 'expense' }
		]);
		expect(categoryDisplayLabel('Кредиты', 'income', dup)).toBe('Кредиты (Доход)');
		expect(categoryDisplayLabel('Кредиты', 'expense', dup)).toBe('Кредиты (Расход)');
	});
});
