import { describe, expect, it } from 'vitest';
import { groupSubcategoriesByCategory, subcategoryShareOfParent } from './stats-subcategory';
import type { StatsSubcategoryItem } from '$lib/api/client';

function item(
	partial: Partial<StatsSubcategoryItem> &
		Pick<StatsSubcategoryItem, 'category_id' | 'subcategory_id' | 'total'>
): StatsSubcategoryItem {
	return {
		category_name: 'Cat',
		category_icon: 'food',
		subcategory_name: 'Sub',
		percentage: 0,
		count: 1,
		...partial
	};
}

describe('groupSubcategoriesByCategory', () => {
	it('groups and sorts by absolute total', () => {
		const grouped = groupSubcategoriesByCategory([
			item({ category_id: 'c1', subcategory_id: 's1', total: 100, subcategory_name: 'A' }),
			item({ category_id: 'c1', subcategory_id: 's2', total: 500, subcategory_name: 'B' }),
			item({ category_id: 'c2', subcategory_id: 's3', total: 50, subcategory_name: 'C' })
		]);
		expect(grouped.c1.map((x) => x.subcategory_id)).toEqual(['s2', 's1']);
		expect(grouped.c2).toHaveLength(1);
	});
});

describe('subcategoryShareOfParent', () => {
	it('computes percent of parent', () => {
		expect(subcategoryShareOfParent(25, 100)).toBe(25);
		expect(subcategoryShareOfParent(0, 100)).toBe(0);
		expect(subcategoryShareOfParent(10, 0)).toBe(0);
	});
});
