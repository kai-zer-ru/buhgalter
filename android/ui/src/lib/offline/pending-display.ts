import type { Category, Transaction } from '$lib/api/client';
import { getOutboxEntries, isLocalEntityKey } from '$lib/offline/store';
import type { OutboxEntry, TransactionPayload, TransferPayload } from '$lib/offline/types';
import { dedupeTransferLegs } from '$lib/transaction-display';

export type AccountNameRef = { id: string; name: string };

const nowIso = () => new Date().toISOString();

function accountName(accounts: AccountNameRef[], id: string): string {
	return accounts.find((a) => a.id === id)?.name ?? '';
}

function categoryMeta(categories: Category[], id?: string) {
	if (!id) return { name: undefined as string | undefined, icon: undefined as string | undefined };
	const c = categories.find((x) => x.id === id);
	return { name: c?.name, icon: c?.icon };
}

function amountToCents(amount: string): number {
	const n = Number(amount.replace(/\s/g, '').replace(',', '.'));
	return Math.round(n * 100);
}

function transactionFromCreate(
	entry: OutboxEntry,
	payload: TransactionPayload,
	accounts: AccountNameRef[],
	categories: Category[]
): Transaction {
	const cat = categoryMeta(categories, payload.category_id);
	const ts = nowIso();
	return {
		id: entry.entityKey,
		account_id: payload.account_id,
		account_name: accountName(accounts, payload.account_id),
		account_status: 'active',
		type: payload.type,
		kind: 'manual',
		amount: amountToCents(payload.amount),
		amount_display: payload.amount,
		description: payload.description ?? null,
		category_id: payload.category_id ?? null,
		category_name: cat.name ?? null,
		category_icon: cat.icon ?? null,
		category_is_system: false,
		subcategory_id: payload.subcategory_id ?? null,
		subcategory_name: payload.subcategory_name ?? null,
		transfer_group_id: undefined,
		transfer_account_id: undefined,
		transfer_account_name: undefined,
		transfer_is_out: undefined,
		transaction_date: payload.transaction_date,
		created_at: ts,
		updated_at: ts,
		credit_payment_linked: false,
		deletable: true
	};
}

function systemTransferCategory(categories: Category[]) {
	return categories.find((c) => c.is_system && c.icon === 'transfer');
}

function transferFromCreate(
	entry: OutboxEntry,
	payload: TransferPayload,
	accounts: AccountNameRef[],
	categories: Category[]
): Transaction {
	const transferCat = systemTransferCategory(categories);
	const ts = nowIso();
	return {
		id: `${entry.entityKey}:out`,
		account_id: payload.from_account_id,
		account_name: accountName(accounts, payload.from_account_id),
		account_status: 'active',
		type: 'transfer',
		kind: 'manual',
		amount: amountToCents(payload.amount),
		amount_display: payload.amount,
		description: payload.description ?? null,
		category_id: transferCat?.id ?? null,
		category_name: null,
		category_icon: transferCat?.icon ?? 'transfer',
		category_is_system: true,
		subcategory_id: null,
		subcategory_name: null,
		transfer_group_id: entry.entityKey,
		transfer_account_id: payload.to_account_id,
		transfer_account_name: accountName(accounts, payload.to_account_id),
		transfer_account_status: 'active',
		transfer_is_out: true,
		transaction_date: payload.transaction_date,
		created_at: ts,
		updated_at: ts,
		credit_payment_linked: false,
		deletable: true
	};
}

export function pendingTransactions(
	accounts: AccountNameRef[],
	categories: Category[]
): Transaction[] {
	const result: Transaction[] = [];
	for (const entry of getOutboxEntries()) {
		if (entry.op === 'delete' || !entry.payload) continue;
		if (entry.kind === 'transaction') {
			result.push(
				transactionFromCreate(entry, entry.payload as TransactionPayload, accounts, categories)
			);
		} else {
			result.push(
				transferFromCreate(entry, entry.payload as TransferPayload, accounts, categories)
			);
		}
	}
	return result;
}

export function mergeTransactionLists(
	server: Transaction[],
	accounts: AccountNameRef[],
	categories: Category[],
	opts?: { hidePendingDeletes?: boolean }
): Transaction[] {
	const hideDeletes = opts?.hidePendingDeletes ?? true;
	const pendingDeletes = new Set(
		getOutboxEntries()
			.filter((e) => e.op === 'delete' && !e.isLocalOnly)
			.map((e) => e.entityKey)
	);
	const pendingTransferDeletes = new Set(
		getOutboxEntries()
			.filter((e) => e.op === 'delete' && e.kind === 'transfer' && !e.isLocalOnly)
			.map((e) => e.entityKey)
	);

	const updates = new Map(
		getOutboxEntries()
			.filter((e) => e.op === 'update' && e.payload)
			.map((e) => [e.entityKey, e])
	);

	let merged = server.filter((tx) => {
		if (!hideDeletes) return true;
		if (pendingDeletes.has(tx.id)) return false;
		if (tx.transfer_group_id && pendingTransferDeletes.has(tx.transfer_group_id)) return false;
		return true;
	});

	merged = merged.map((tx) => {
		const key = tx.transfer_group_id && tx.type === 'transfer' ? tx.transfer_group_id : tx.id;
		const upd = updates.get(key);
		if (!upd?.payload) return tx;
		if (upd.kind === 'transfer' && tx.type === 'transfer') {
			return transferFromCreate(upd, upd.payload as TransferPayload, accounts, categories);
		}
		if (upd.kind === 'transaction' && tx.type !== 'transfer') {
			return transactionFromCreate(upd, upd.payload as TransactionPayload, accounts, categories);
		}
		return tx;
	});

	const pending = pendingTransactions(accounts, categories);
	const combined = [...pending, ...merged];
	return sortTransactionsDateDesc(dedupeTransferLegs(combined));
}

/** Match server ListTransactionsFilteredDateDesc: newest transaction_date first. */
export function sortTransactionsDateDesc(txs: Transaction[]): Transaction[] {
	return [...txs].sort((a, b) => {
		const byDate = b.transaction_date.localeCompare(a.transaction_date);
		if (byDate !== 0) return byDate;
		return b.created_at.localeCompare(a.created_at);
	});
}

function pendingEntryForKey(id: string): OutboxEntry | undefined {
	const entries = getOutboxEntries();
	for (let i = entries.length - 1; i >= 0; i--) {
		const entry = entries[i];
		if (entry.entityKey !== id) continue;
		if (entry.op === 'delete') return undefined;
		return entry;
	}
	return undefined;
}

/** Resolve a pending outbox transaction or transfer out-leg by entity / group id. */
export function lookupPendingTransaction(
	id: string,
	accounts: AccountNameRef[],
	categories: Category[]
): Transaction | null {
	const entry = pendingEntryForKey(id);
	if (!entry?.payload) return null;
	if (entry.kind === 'transfer') {
		return transferFromCreate(entry, entry.payload as TransferPayload, accounts, categories);
	}
	if (entry.kind === 'transaction') {
		return transactionFromCreate(entry, entry.payload as TransactionPayload, accounts, categories);
	}
	return null;
}

/** Both legs for a pending local transfer (edit form). */
export function lookupPendingTransferLegs(
	groupId: string,
	accounts: AccountNameRef[],
	categories: Category[]
): Transaction[] {
	const entry = pendingEntryForKey(groupId);
	if (!entry?.payload || entry.kind !== 'transfer') return [];
	const payload = entry.payload as TransferPayload;
	const out = transferFromCreate(entry, payload, accounts, categories);
	const inLeg: Transaction = {
		...out,
		id: `${entry.entityKey}:in`,
		account_id: payload.to_account_id,
		account_name: accountName(accounts, payload.to_account_id),
		transfer_is_out: false
	};
	return [out, inLeg];
}

export function isPendingTransaction(tx: Transaction): boolean {
	return isLocalEntityKey(tx.id) || isLocalEntityKey(tx.transfer_group_id ?? '');
}

export function pendingSyncFailed(tx: Transaction): string | undefined {
	const key = tx.type === 'transfer' && tx.transfer_group_id ? tx.transfer_group_id : tx.id;
	const entry = getOutboxEntries().find((e) => e.entityKey === key);
	return entry?.failed?.message;
}
