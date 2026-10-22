import type { StatsSubcategoryItem } from '$lib/api/client';

/** Group subcategory rows by parent category_id (sorted by |total| desc). */
export function groupSubcategoriesByCategory(
	items: StatsSubcategoryItem[]
): Record<string, StatsSubcategoryItem[]> {
	const map: Record<string, StatsSubcategoryItem[]> = {};
	for (const item of items) {
		const list = map[item.category_id] ?? [];
		list.push(item);
		map[item.category_id] = list;
	}
	for (const id of Object.keys(map)) {
		map[id].sort((a, b) => Math.abs(b.total) - Math.abs(a.total));
	}
	return map;
}

/** Share of subcategory total within its parent category (0–100). */
export function subcategoryShareOfParent(subTotal: number, parentTotal: number): number {
	const parent = Math.abs(parentTotal);
	if (parent === 0) return 0;
	return (Math.abs(subTotal) / parent) * 100;
}
