import type { Transaction } from '$lib/api/client';
import { readRefCache, writeRefCache } from '$lib/offline/ref-cache';

const TX_INDEX_KEY = '__internal_tx_index__';

export function indexTransactions(txs: Transaction[]): void {
	if (!txs.length) return;
	const index = readRefCache<Record<string, Transaction>>(TX_INDEX_KEY) ?? {};
	for (const tx of txs) {
		index[tx.id] = tx;
		if (tx.transfer_group_id) {
			index[`tg:${tx.transfer_group_id}`] = tx;
		}
	}
	writeRefCache(TX_INDEX_KEY, index);
}

export function lookupServerTransaction(entityKey: string): Transaction | null {
	const index = readRefCache<Record<string, Transaction>>(TX_INDEX_KEY);
	if (index) {
		const hit = index[entityKey] ?? index[`tg:${entityKey}`];
		if (hit) return hit;
	}
	return readRefCache<Transaction>(`/api/v1/transactions/${entityKey}`);
}

export function listIndexedTransferLegs(groupId: string): Transaction[] {
	const index = readRefCache<Record<string, Transaction>>(TX_INDEX_KEY);
	if (!index) return [];
	return Object.values(index).filter((tx) => tx.transfer_group_id === groupId);
}

export function findTransferCommissionKopecks(groupId: string): number {
	const index = readRefCache<Record<string, Transaction>>(TX_INDEX_KEY);
	if (!index) return 0;
	for (const tx of Object.values(index)) {
		if (tx.transfer_group_id === groupId && tx.type === 'expense' && tx.category_is_system) {
			return tx.amount;
		}
	}
	return 0;
}

export function resetTransactionIndexForTests(): void {
	writeRefCache(TX_INDEX_KEY, {});
}
