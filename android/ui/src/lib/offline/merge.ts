import {
	getUIMeta,
	type AccountBalanceSummary,
	type Category,
	type Transaction
} from '$lib/api/client';
import { accountsFromUIMeta } from '$lib/select-options';
import { mergeTransactionLists, type AccountNameRef } from '$lib/offline/pending-display';

let mergeAccounts: AccountNameRef[] = [];
let mergeCategories: Category[] = [];

export async function refreshMergeMeta(): Promise<void> {
	const meta = await getUIMeta();
	mergeAccounts = accountsFromUIMeta(meta.accounts, meta.banks);
	mergeCategories = [...meta.expense_categories, ...meta.income_categories];
}

export function mergeOutboxTransactions(transactions: Transaction[]): Transaction[] {
	return mergeTransactionLists(transactions, mergeAccounts, mergeCategories);
}

export function mergeMetaAccounts(): AccountNameRef[] {
	return mergeAccounts;
}

export function mergeMetaCategories(): Category[] {
	return mergeCategories;
}

export function mergeAccountsFallback(accounts: AccountBalanceSummary[]): void {
	if (mergeAccounts.length === 0) {
		mergeAccounts = accounts.map((a) => ({ id: a.id, name: a.name }));
	}
}
