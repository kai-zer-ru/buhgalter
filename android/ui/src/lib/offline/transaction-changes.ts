import type {
	Dashboard,
	DebtorDetail,
	Transaction,
	TransactionChange,
	TransactionList
} from '$lib/api/client';
import { listTransactionChanges } from '$lib/api/client';
import {
	AUTH_ME_PATH,
	invalidateRefCache,
	listRefCachePaths,
	publishRefCachePath,
	readRefCache,
	writeRefCache
} from '$lib/offline/ref-cache';
import { isServerOfflineMode } from '$lib/offline/server-connectivity';
import { sortTransactionsDateDesc } from '$lib/offline/pending-display';
import { removeIndexedTransaction, upsertIndexedTransaction } from '$lib/offline/transaction-index';
import { debugLogWarn } from '$lib/platform/debug-log';
import { isDashboardRefPath } from '$lib/state-utils';

const TX_CHANGES_CURSOR_KEY = '__internal_tx_changes_cursor__';
const TX_LIST_PATH = '/api/v1/transactions';
const TX_DETAIL_PREFIX = '/api/v1/transactions/';
const DEBTOR_DETAIL_PREFIX = '/api/v1/debtors/';

type TxChangesCursor = { user_id: string; since_id: number };

function isHiddenCommission(tx: Transaction): boolean {
	return tx.type === 'expense' && !!tx.transfer_group_id;
}

function parseListQuery(path: string): URLSearchParams | null {
	const qIndex = path.indexOf('?');
	const pathname = qIndex >= 0 ? path.slice(0, qIndex) : path;
	if (pathname !== TX_LIST_PATH) return null;
	return new URLSearchParams(qIndex >= 0 ? path.slice(qIndex + 1) : '');
}

function debtorDetailId(path: string): string | null {
	if (path.includes('?')) return null;
	if (!path.startsWith(DEBTOR_DETAIL_PREFIX)) return null;
	const id = path.slice(DEBTOR_DETAIL_PREFIX.length);
	if (!id || id.includes('/')) return null;
	return id;
}

function listPage(params: URLSearchParams): number {
	const page = Number.parseInt(params.get('page') || '1', 10);
	return Number.isFinite(page) && page > 0 ? page : 1;
}

/** Incoming transfer leg is the same operation as the outgoing one, except on a single-account list. */
function isCollapsedTransferInLeg(tx: Transaction, params: URLSearchParams): boolean {
	if (params.get('account_id')) return false;
	return tx.type === 'transfer' && !!tx.transfer_group_id && tx.transfer_is_out === false;
}

function transactionMatchesListQuery(tx: Transaction, params: URLSearchParams): boolean {
	if (isHiddenCommission(tx) || isCollapsedTransferInLeg(tx, params)) return false;
	const accountId = params.get('account_id');
	if (accountId && tx.account_id !== accountId) return false;
	const type = params.get('type');
	if (type && tx.type !== type) return false;
	const categoryId = params.get('category_id');
	if (categoryId && tx.category_id !== categoryId) return false;
	const kind = params.get('kind');
	if (kind && tx.kind !== kind) return false;
	const from = params.get('from');
	if (from && tx.transaction_date < from) return false;
	const to = params.get('to');
	if (to && tx.transaction_date > to) return false;
	const search = params.get('search');
	if (search && !(tx.description ?? '').includes(search)) return false;
	const merchantId = params.get('merchant_id');
	if (merchantId && tx.merchant_id !== merchantId) return false;
	const tagId = params.get('tag_id');
	if (tagId && !(tx.tags ?? []).some((tag) => tag.id === tagId)) return false;
	return true;
}

function sortList(txs: Transaction[], params: URLSearchParams): Transaction[] {
	const sorted = sortTransactionsDateDesc(txs);
	return params.get('sort') === 'date_asc' ? sorted.reverse() : sorted;
}

function pageLimit(params: URLSearchParams, fallback: number): number {
	const limit = Number.parseInt(params.get('limit') || '', 10);
	if (Number.isFinite(limit) && limit > 0) return limit;
	return fallback;
}

function patchTransactionList(
	path: string,
	params: URLSearchParams,
	change: TransactionChange
): void {
	const cached = readRefCache<TransactionList>(path);
	if (!cached || !Array.isArray(cached.data)) return;
	const page = listPage(params);
	const data = [...cached.data];
	const total = cached.meta?.total ?? data.length;

	if (change.action === 'deleted') {
		const next = data.filter((row) => row.id !== change.entity_id);
		if (next.length === data.length) return;
		publishRefCachePath(path, {
			data: next,
			meta: { ...cached.meta, total: Math.max(0, total - 1) }
		});
		return;
	}

	const tx = change.transaction;
	if (!tx) return;
	const matches = transactionMatchesListQuery(tx, params);
	const idx = data.findIndex((row) => row.id === tx.id);
	if (!matches) {
		if (idx < 0) return;
		data.splice(idx, 1);
		publishRefCachePath(path, {
			data: sortList(data, params),
			meta: { ...cached.meta, total: Math.max(0, total - 1) }
		});
		return;
	}
	if (idx >= 0) {
		data[idx] = tx;
		publishRefCachePath(path, { data: sortList(data, params), meta: cached.meta });
		return;
	}
	if (page > 1) return;
	// A row older than this page is already inside meta.total from the list GET.
	// Counting it again is how Android drifted above the web journal (replay of
	// transfer legs and older operations after warm).
	const cap = pageLimit(params, data.length);
	const kept = sortList([...data, tx], params).slice(0, cap > 0 ? cap : undefined);
	if (!kept.some((row) => row.id === tx.id)) return;
	publishRefCachePath(path, {
		data: kept,
		meta: { ...cached.meta, total: total + 1 }
	});
}

function patchDashboard(change: TransactionChange): void {
	const dash = readRefCache<Dashboard>('/api/v1/dashboard');
	if (!dash || !Array.isArray(dash.recent_transactions)) return;
	let recent = [...dash.recent_transactions];
	if (change.action === 'deleted') {
		const next = recent.filter((row) => row.id !== change.entity_id);
		if (next.length === recent.length) return;
		publishRefCachePath('/api/v1/dashboard', { ...dash, recent_transactions: next });
		return;
	}
	const tx = change.transaction;
	if (!tx || isHiddenCommission(tx)) {
		const next = recent.filter((row) => row.id !== change.entity_id);
		if (next.length === recent.length) return;
		publishRefCachePath('/api/v1/dashboard', { ...dash, recent_transactions: next });
		return;
	}
	const idx = recent.findIndex((row) => row.id === tx.id);
	const cap = Math.max(10, recent.length);
	if (idx >= 0) {
		recent[idx] = tx;
		recent = sortTransactionsDateDesc(recent);
	} else {
		const merged = sortTransactionsDateDesc([tx, ...recent]);
		if (merged.findIndex((row) => row.id === tx.id) >= cap) return;
		recent = merged.slice(0, cap);
	}
	publishRefCachePath('/api/v1/dashboard', { ...dash, recent_transactions: recent });
}

function patchDebtorDetail(path: string, change: TransactionChange): void {
	const detail = readRefCache<DebtorDetail>(path);
	if (!detail || !Array.isArray(detail.transactions)) return;
	const txs = [...detail.transactions];
	if (change.action === 'deleted') {
		const next = txs.filter((row) => row.id !== change.entity_id);
		if (next.length === txs.length) return;
		publishRefCachePath(path, { ...detail, transactions: next });
		return;
	}
	const tx = change.transaction;
	if (!tx) return;
	const idx = txs.findIndex((row) => row.id === tx.id);
	if (idx < 0) return;
	txs[idx] = {
		...txs[idx],
		id: tx.id,
		account_id: tx.account_id,
		account_name: tx.account_name,
		type: tx.type,
		kind: tx.kind,
		amount: tx.amount,
		amount_display: tx.amount_display,
		description: tx.description,
		category_name: tx.category_name ?? undefined,
		transaction_date: tx.transaction_date
	};
	publishRefCachePath(path, { ...detail, transactions: txs });
}

function applyDeleted(id: string): void {
	removeIndexedTransaction(id);
	invalidateRefCache(`${TX_DETAIL_PREFIX}${id}`);
	for (const path of listRefCachePaths()) {
		const listQuery = parseListQuery(path);
		if (listQuery) {
			patchTransactionList(path, listQuery, {
				id: 0,
				action: 'deleted',
				entity_id: id,
				occurred_at: ''
			});
			continue;
		}
		if (isDashboardRefPath(path)) {
			patchDashboard({ id: 0, action: 'deleted', entity_id: id, occurred_at: '' });
			continue;
		}
		if (debtorDetailId(path)) {
			patchDebtorDetail(path, { id: 0, action: 'deleted', entity_id: id, occurred_at: '' });
		}
	}
}

function applyUpsert(tx: Transaction): void {
	upsertIndexedTransaction(tx);
	publishRefCachePath(`${TX_DETAIL_PREFIX}${tx.id}`, tx);
	const change: TransactionChange = {
		id: 0,
		action: 'upsert',
		entity_id: tx.id,
		occurred_at: '',
		transaction: tx
	};
	for (const path of listRefCachePaths()) {
		const listQuery = parseListQuery(path);
		if (listQuery) {
			patchTransactionList(path, listQuery, change);
			continue;
		}
		if (isDashboardRefPath(path)) {
			patchDashboard(change);
			continue;
		}
		if (debtorDetailId(path)) {
			patchDebtorDetail(path, change);
		}
	}
}

/** Apply a page of server change-feed events to ref-cache lists and the tx index. */
export function applyTransactionChanges(changes: TransactionChange[]): void {
	for (const change of changes) {
		if (change.action === 'deleted') {
			applyDeleted(change.entity_id);
			continue;
		}
		if (change.transaction) applyUpsert(change.transaction);
	}
}

function readCursor(userId: string): TxChangesCursor {
	const stored = readRefCache<TxChangesCursor>(TX_CHANGES_CURSOR_KEY);
	if (!stored || (userId && stored.user_id && stored.user_id !== userId)) {
		return { user_id: userId, since_id: 0 };
	}
	return { user_id: userId || stored.user_id, since_id: stored.since_id ?? 0 };
}

/** Pull GET /sync/transaction-changes and patch cached operations. */
export async function pullTransactionChanges(): Promise<boolean> {
	if (isServerOfflineMode()) return false;
	const me = readRefCache<{ id?: string }>(AUTH_ME_PATH);
	const userId = me?.id ?? '';
	let cursor = readCursor(userId);
	let applied = false;
	try {
		for (let page = 0; page < 20; page++) {
			const res = await listTransactionChanges({ since_id: cursor.since_id, limit: 200 });
			if (res.changes.length) {
				applyTransactionChanges(res.changes);
				applied = true;
			}
			cursor = { user_id: userId || cursor.user_id, since_id: res.since_id };
			writeRefCache(TX_CHANGES_CURSOR_KEY, cursor);
			if (!res.has_more) break;
		}
	} catch (err) {
		debugLogWarn('sync', 'transaction changes pull failed', { error: String(err) });
		return applied;
	}
	return applied;
}

export function resetTransactionChangesCursorForTests(): void {
	writeRefCache(TX_CHANGES_CURSOR_KEY, { user_id: '', since_id: 0 });
}
