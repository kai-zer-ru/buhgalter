import {
	getUIMeta,
	type AccountBalanceSummary,
	type Category,
	type Transaction,
	type UIMeta
} from '$lib/api/client';
import { readRefCache } from '$lib/offline/ref-cache';
import { accountsFromUIMeta } from '$lib/select-options';
import { mergeTransactionLists, type AccountNameRef } from '$lib/offline/pending-display';

const UI_META_PATH = '/api/v1/ui/meta';

let mergeAccounts: AccountNameRef[] = [];
let mergeCategories: Category[] = [];

function applyMergeMeta(meta: UIMeta): void {
	mergeAccounts = accountsFromUIMeta(meta.accounts, meta.banks);
	mergeCategories = [...meta.expense_categories, ...meta.income_categories];
}

/** Load account/category names for outbox merge — cache-only unless meta missing. */
export async function refreshMergeMeta(): Promise<void> {
	const cached = readRefCache<UIMeta>(UI_META_PATH);
	if (cached) {
		applyMergeMeta(cached);
		return;
	}
	const meta = await getUIMeta();
	applyMergeMeta(meta);
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
