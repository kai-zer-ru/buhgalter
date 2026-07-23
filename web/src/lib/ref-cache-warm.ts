import {
	getBudgetSummary,
	getDashboard,
	getUIMeta,
	listAccounts,
	listTransactions
} from '$lib/api/client';

/** Background prefetch of common GET paths into ref-cache. */
export async function warmRefCache(): Promise<void> {
	await Promise.allSettled([
		getDashboard(),
		getUIMeta(),
		listAccounts(),
		listAccounts('active'),
		getBudgetSummary(),
		listTransactions({
			kind: 'manual',
			sort: 'date_desc',
			page: '1',
			limit: '20'
		}),
		listTransactions({
			kind: 'future',
			sort: 'date_desc',
			page: '1',
			limit: '20'
		})
	]);
}
