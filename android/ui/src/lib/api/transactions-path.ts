/** Stable GET path for transaction lists — query keys sorted so ref-cache hits match. */
export function transactionsListPath(params?: Record<string, string>): string {
	if (!params) return '/api/v1/transactions';
	const keys = Object.keys(params).sort();
	if (keys.length === 0) return '/api/v1/transactions';
	const sp = new URLSearchParams();
	for (const key of keys) {
		sp.set(key, params[key]!);
	}
	return `/api/v1/transactions?${sp.toString()}`;
}

/** Home recent past ops — must match `listTransactions` / warmRefCache cache key. */
export const HOME_PAST_TRANSACTIONS_PATH = transactionsListPath({
	kind: 'manual',
	limit: '10',
	page: '1',
	sort: 'date_desc'
});

/** Home planned ops — must match `listTransactions` / warmRefCache cache key. */
export const HOME_PLANNED_TRANSACTIONS_PATH = transactionsListPath({
	kind: 'future',
	limit: '10',
	page: '1',
	sort: 'date_desc'
});
