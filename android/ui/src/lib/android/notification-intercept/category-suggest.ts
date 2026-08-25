import { listTransactions, type Transaction } from '$lib/api/client';

export type CategorySuggestion = {
	categoryId: string;
	subcategoryId?: string;
};

/** API allows up to 200; take a wide window so rare but consistent merchants still match. */
const HISTORY_LIMIT = 100;

function majorityId(items: { id: string; index: number }[]): { id: string; index: number } | null {
	const counts = new Map<string, { n: number; firstIndex: number }>();
	for (const item of items) {
		const prev = counts.get(item.id);
		if (prev) prev.n += 1;
		else counts.set(item.id, { n: 1, firstIndex: item.index });
	}
	let best: { id: string; n: number; firstIndex: number } | null = null;
	for (const [id, row] of counts) {
		if (!best || row.n > best.n || (row.n === best.n && row.firstIndex < best.firstIndex)) {
			best = { id, n: row.n, firstIndex: row.firstIndex };
		}
	}
	return best ? { id: best.id, index: best.firstIndex } : null;
}

/**
 * Suggest category (+ subcategory when history has one) from recent txs.
 * 1) majority category_id among non-system expenses/incomes;
 * 2) among that category, majority of *non-empty* subcategory_id (empty does not block).
 * Ties → more recent (earlier in date_desc list).
 */
export function pickCategorySuggestion(
	txs: Transaction[],
	type: 'expense' | 'income' = 'expense'
): CategorySuggestion | null {
	const usable: { index: number; categoryId: string; subcategoryId?: string }[] = [];
	txs.forEach((tx, index) => {
		if (tx.type !== type) return;
		const categoryId = tx.category_id;
		if (!categoryId || tx.category_is_system) return;
		usable.push({
			index,
			categoryId,
			subcategoryId: tx.subcategory_id || undefined
		});
	});
	if (!usable.length) return null;

	const cat = majorityId(usable.map((u) => ({ id: u.categoryId, index: u.index })));
	if (!cat) return null;

	const inCat = usable.filter((u) => u.categoryId === cat.id);
	const withSub = inCat
		.filter((u) => u.subcategoryId)
		.map((u) => ({ id: u.subcategoryId!, index: u.index }));
	const sub = withSub.length ? majorityId(withSub) : null;

	return {
		categoryId: cat.id,
		subcategoryId: sub?.id
	};
}

/**
 * Suggest category/subcategory from prior txs of this merchant (same type).
 * Offline / API errors → null (form keeps primary default).
 */
export async function suggestCategoryFromMerchant(
	merchantId: string,
	type: 'expense' | 'income' = 'expense'
): Promise<CategorySuggestion | null> {
	const id = merchantId.trim();
	if (!id) return null;
	try {
		const res = await listTransactions({
			type,
			merchant_id: id,
			sort: 'date_desc',
			limit: String(HISTORY_LIMIT),
			page: '1'
		});
		return pickCategorySuggestion(res.data ?? [], type);
	} catch {
		return null;
	}
}
