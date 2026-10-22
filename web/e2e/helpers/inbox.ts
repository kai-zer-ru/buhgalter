import type { Page } from '@playwright/test';
import { apiJSON } from './auth';

type InboxTx = {
	id: string;
	account_id: string;
	type: 'income' | 'expense';
	amount_display: string;
	transaction_date: string;
	description?: string | null;
};

/** Assign a category to every item in the inbox queue (shared e2e DB may leave stragglers). */
export async function clearInboxQueue(page: Page): Promise<void> {
	let expenseCategoryId: string | undefined;
	let incomeCategoryId: string | undefined;

	for (;;) {
		const { data } = await apiJSON<{ data: InboxTx[] }>(
			page,
			'GET',
			'/api/v1/transactions/inbox?limit=50'
		);
		if (data.length === 0) return;

		for (const tx of data) {
			if (tx.type === 'expense' && !expenseCategoryId) {
				const cats = await apiJSON<{ id: string }[]>(
					page,
					'GET',
					'/api/v1/categories?type=expense'
				);
				expenseCategoryId = cats[0]?.id;
			}
			if (tx.type === 'income' && !incomeCategoryId) {
				const cats = await apiJSON<{ id: string }[]>(page, 'GET', '/api/v1/categories?type=income');
				incomeCategoryId = cats[0]?.id;
			}
			const categoryId = tx.type === 'income' ? incomeCategoryId : expenseCategoryId;
			if (!categoryId) {
				throw new Error(`no ${tx.type} category for inbox cleanup`);
			}
			await apiJSON(page, 'PUT', `/api/v1/transactions/${tx.id}`, {
				account_id: tx.account_id,
				type: tx.type,
				amount: tx.amount_display,
				transaction_date: tx.transaction_date,
				...(tx.description != null && tx.description !== '' ? { description: tx.description } : {}),
				category_id: categoryId
			});
		}
	}
}
