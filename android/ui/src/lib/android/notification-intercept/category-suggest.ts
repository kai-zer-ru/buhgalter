import { listTransactions, type Transaction } from '$lib/api/client';

export type CategorySuggestion = {
	categoryId: string;
	subcategoryId?: string;
};

const HISTORY_LIMIT = 10;

/** Pair key: category + optional subcategory (empty subcategory is distinct from missing). */
function pairKey(categoryId: string, subcategoryId: string | null | undefined): string {
	return `${categoryId}\0${subcategoryId ?? ''}`;
}

/**
 * Majority vote over recent expenses for a merchant.
 * Ignores empty / system categories. Ties → more recent pair (first in date_desc list).
 */
export function pickCategorySuggestion(txs: Transaction[]): CategorySuggestion | null {
	const counts = new Map<
		string,
		{ n: number; firstIndex: number; categoryId: string; subcategoryId?: string }
	>();

	txs.forEach((tx, index) => {
		if (tx.type !== 'expense') return;
		const categoryId = tx.category_id;
		if (!categoryId || tx.category_is_system) return;
		const subcategoryId = tx.subcategory_id || undefined;
		const key = pairKey(categoryId, subcategoryId);
		const prev = counts.get(key);
		if (prev) {
			prev.n += 1;
		} else {
			counts.set(key, { n: 1, firstIndex: index, categoryId, subcategoryId });
		}
	});

	let best: { n: number; firstIndex: number; categoryId: string; subcategoryId?: string } | null =
		null;
	for (const row of counts.values()) {
		if (!best || row.n > best.n || (row.n === best.n && row.firstIndex < best.firstIndex)) {
			best = row;
		}
	}
	if (!best) return null;
	return {
		categoryId: best.categoryId,
		subcategoryId: best.subcategoryId
	};
}

/**
 * Suggest category/subcategory from prior expenses of this merchant catalog entry.
 * Offline / API errors → null (form keeps primary default).
 */
export async function suggestCategoryFromMerchant(
	merchantId: string
): Promise<CategorySuggestion | null> {
	const id = merchantId.trim();
	if (!id) return null;
	try {
		const res = await listTransactions({
			type: 'expense',
			merchant_id: id,
			sort: 'date_desc',
			limit: String(HISTORY_LIMIT),
			page: '1'
		});
		return pickCategorySuggestion(res.data ?? []);
	} catch {
		return null;
	}
}
