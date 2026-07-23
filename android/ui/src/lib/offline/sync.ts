import { get, writable } from 'svelte/store';
import { invalidateApiCache } from '$lib/api/cache';
import {
	getDashboard,
	getDebtsSummary,
	getUIMeta,
	getBudgetSummary,
	listAccounts,
	listCredits,
	listDebts,
	listTransactions,
	createTransaction as apiCreateTransaction,
	createTransfer as apiCreateTransfer,
	createCategory as apiCreateCategory,
	createDebt as apiCreateDebt,
	createAccount as apiCreateAccount,
	createBudget as apiCreateBudget,
	deleteTransaction as apiDeleteTransaction,
	deleteTransfer as apiDeleteTransfer,
	deleteCategory as apiDeleteCategory,
	deleteDebt as apiDeleteDebt,
	deleteBudget as apiDeleteBudget,
	updateTransaction as apiUpdateTransaction,
	updateTransfer as apiUpdateTransfer,
	updateCategory as apiUpdateCategory,
	updateAccount as apiUpdateAccount,
	updateBudget as apiUpdateBudget,
	archiveAccount as apiArchiveAccount,
	unarchiveAccount as apiUnarchiveAccount,
	ApiError,
	isTransientHttpError
} from '$lib/api/client';
import {
	getOutboxEntries,
	hasFailedOutbox,
	hasPendingOutbox,
	markOutboxFailed,
	removeOutboxEntry
} from '$lib/offline/store';
import type {
	TransactionPayload,
	TransferPayload,
	CategoryPayload,
	CategoryUpdatePayload,
	DebtPayload,
	AccountCreatePayload,
	AccountUpdatePayload,
	BudgetPayload
} from '$lib/offline/types';
import { isAccountStatusPayload } from '$lib/offline/types';
import type { OutboxEntry } from '$lib/offline/types';
import {
	markServerOffline,
	markServerOnline,
	probeServerReachability,
	shouldTryServer
} from '$lib/offline/server-connectivity';
import { debugLogError, debugLogInfo, debugLogWarn } from '$lib/platform/debug-log';

export const syncState = writable<'idle' | 'syncing'>('idle');

export type PullStatusKind = 'idle' | 'syncing' | 'ok' | 'pending' | 'failed' | 'offline' | 'error';

export type PullStatus = {
	kind: PullStatusKind;
	lastOkAt: number | null;
	pendingCount: number;
};

export const pullStatus = writable<PullStatus>({
	kind: 'idle',
	lastOkAt: null,
	pendingCount: 0
});

/** Bumped after local outbox edits — lists re-merge without a server round-trip. */
export const localDataTick = writable(0);

/** Bumped after manual pull / successful sync — pages reload lists from the server. */
export const dataRefreshTick = writable(0);

/** Index warmup page size for manual sync (balance overlay / offline edit-delete). */
export const TX_INDEX_WARMUP_LIMIT = '200';

let syncPromise: Promise<boolean> | null = null;

function pendingCount(): number {
	return getOutboxEntries().filter((e) => !e.failed).length;
}

function refreshPullStatus(override?: PullStatusKind) {
	pullStatus.update((s) => {
		const pending = pendingCount();
		const failed = hasFailedOutbox();
		let kind = override ?? s.kind;
		if (!override) {
			if (failed) kind = 'failed';
			else if (pending > 0) kind = 'pending';
			else if (!['ok', 'syncing', 'error', 'offline'].includes(s.kind)) kind = 'idle';
		}
		return { ...s, kind, pendingCount: pending };
	});
}

function isRetryable(err: unknown): boolean {
	if (!(err instanceof ApiError)) return true;
	return isTransientHttpError(err.status) || err.status === 0;
}

async function replayEntry(entry: OutboxEntry) {
	const { entityKey, kind, op, payload } = entry;
	try {
		if (op === 'delete') {
			if (kind === 'transaction') {
				await apiDeleteTransaction(entityKey);
			} else if (kind === 'transfer') {
				await apiDeleteTransfer(entityKey);
			} else if (kind === 'category') {
				await apiDeleteCategory(entityKey);
			} else if (kind === 'debt') {
				await apiDeleteDebt(entityKey);
			} else if (kind === 'budget') {
				await apiDeleteBudget(entityKey);
			}
			removeOutboxEntry(entityKey);
			return;
		}
		if (op === 'create') {
			if (kind === 'transaction') {
				await apiCreateTransaction(payload as TransactionPayload);
			} else if (kind === 'transfer') {
				await apiCreateTransfer(payload as TransferPayload);
			} else if (kind === 'category') {
				await apiCreateCategory(payload as CategoryPayload);
			} else if (kind === 'debt') {
				await apiCreateDebt(payload as DebtPayload);
			} else if (kind === 'account') {
				await apiCreateAccount(payload as AccountCreatePayload);
			} else if (kind === 'budget') {
				const bp = payload as BudgetPayload;
				const { month, ...body } = bp;
				await apiCreateBudget(body, month);
			}
			removeOutboxEntry(entityKey);
			return;
		}
		if (op === 'update') {
			if (kind === 'transaction') {
				await apiUpdateTransaction(entityKey, payload as TransactionPayload);
			} else if (kind === 'transfer') {
				await apiUpdateTransfer(entityKey, payload as TransferPayload);
			} else if (kind === 'category') {
				await apiUpdateCategory(entityKey, payload as CategoryUpdatePayload);
			} else if (kind === 'account') {
				if (isAccountStatusPayload(payload)) {
					if (payload.action === 'archive') {
						await apiArchiveAccount(entityKey, payload.transfer_to_account_id);
					} else {
						await apiUnarchiveAccount(entityKey);
					}
				} else {
					await apiUpdateAccount(entityKey, payload as AccountUpdatePayload);
				}
			} else if (kind === 'budget') {
				const bp = payload as BudgetPayload;
				const { month, ...body } = bp;
				await apiUpdateBudget(entityKey, body, month);
			}
			removeOutboxEntry(entityKey);
		}
	} catch (err) {
		if (isRetryable(err)) throw err;
		const message = err instanceof ApiError ? err.message : String(err);
		markOutboxFailed(entityKey, message);
	}
}

async function replayOutbox(): Promise<void> {
	const ordered = getOutboxEntries().filter((e) => !e.failed);
	for (const entry of ordered) {
		await replayEntry(entry);
	}
}

function setPullResult(kind: PullStatusKind) {
	const prev = get(pullStatus);
	const pending = pendingCount();
	pullStatus.set({
		kind,
		lastOkAt: kind === 'ok' || kind === 'pending' ? Date.now() : prev.lastOkAt,
		pendingCount: pending
	});
}

async function withSyncLock(run: () => Promise<boolean>): Promise<boolean> {
	if (syncPromise) return syncPromise;

	syncPromise = (async () => {
		syncState.set('syncing');
		try {
			return await run();
		} finally {
			syncState.set('idle');
			syncPromise = null;
		}
	})();

	return syncPromise;
}

/** Replay pending outbox only (background / after local edits). No dashboard pull. */
export async function syncOutbox(): Promise<boolean> {
	if (!hasPendingOutbox()) {
		refreshPullStatus();
		return true;
	}
	if (!(await shouldTryServer())) {
		debugLogWarn('sync', 'syncOutbox skipped — server offline');
		refreshPullStatus('offline');
		return false;
	}

	return withSyncLock(async () => {
		debugLogInfo('sync', 'syncOutbox started');
		pullStatus.update((s) => ({ ...s, kind: 'syncing' }));
		try {
			invalidateApiCache();
			await replayOutbox();
			const failed = hasFailedOutbox();
			const pending = pendingCount();
			const kind: PullStatusKind = failed ? 'failed' : pending > 0 ? 'pending' : 'ok';
			setPullResult(kind);
			markServerOnline();
			dataRefreshTick.update((n) => n + 1);
			debugLogInfo('sync', `syncOutbox finished: ${kind}`);
			return kind === 'ok' || kind === 'pending';
		} catch (err) {
			markServerOffline();
			debugLogError('sync', 'syncOutbox failed', { error: String(err) });
			refreshPullStatus('error');
			return false;
		}
	});
}

/** Manual sync: force reconnect if offline, then warm cache + outbox replay. */
export async function pullFromServer(): Promise<boolean> {
	return withSyncLock(async () => {
		debugLogInfo('sync', 'pullFromServer started');
		pullStatus.update((s) => ({ ...s, kind: 'syncing' }));
		try {
			if (!(await shouldTryServer())) {
				debugLogInfo('sync', 'pullFromServer probing (was offline)');
				const online = await probeServerReachability();
				if (!online) {
					debugLogWarn('sync', 'pullFromServer — server still offline');
					refreshPullStatus('offline');
					return false;
				}
			}

			invalidateApiCache();
			await warmRefCache();
			if (hasPendingOutbox()) {
				await replayOutbox();
			}

			const failed = hasFailedOutbox();
			const pending = pendingCount();
			const kind: PullStatusKind = failed ? 'failed' : pending > 0 ? 'pending' : 'ok';
			setPullResult(kind);
			markServerOnline();
			dataRefreshTick.update((n) => n + 1);
			debugLogInfo('sync', `pullFromServer finished: ${kind}`);
			return kind === 'ok' || kind === 'pending';
		} catch (err) {
			markServerOffline();
			debugLogError('sync', 'pullFromServer failed', { error: String(err) });
			refreshPullStatus('error');
			return false;
		}
	});
}

export function scheduleSyncOutbox() {
	void syncOutbox().catch(() => {
		// retry on next network event
	});
}

export function updatePullStatusFromOutbox() {
	refreshPullStatus();
}

/** Called from outbox store on every queue change — instant UI refresh without server pull. */
export function notifyOutboxChanged() {
	localDataTick.update((n) => n + 1);
	refreshPullStatus();
}

async function warmupTransactionIndex(): Promise<void> {
	await listTransactions({
		sort: 'date_desc',
		page: '1',
		limit: TX_INDEX_WARMUP_LIMIT
	});
}

/** Prefetch main GET endpoints into ref-cache (startup / manual sync). */
export async function warmRefCache(): Promise<void> {
	debugLogInfo('sync', 'warmRefCache started');
	await Promise.allSettled([
		getDashboard(),
		getUIMeta(),
		listAccounts(),
		listAccounts('active'),
		listAccounts('archived'),
		listCredits({ status: 'active' }),
		listCredits({ status: 'closed' }),
		getDebtsSummary(),
		listDebts({ settled: 'false' }),
		listDebts({ settled: 'true' }),
		getBudgetSummary(),
		warmupTransactionIndex(),
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
		}),
		listTransactions({
			kind: 'manual',
			sort: 'date_desc',
			page: '1',
			limit: '10'
		}),
		listTransactions({
			kind: 'future',
			sort: 'date_desc',
			page: '1',
			limit: '10'
		})
	]);
	debugLogInfo('sync', 'warmRefCache finished');
	const { publishWidgetSnapshot } = await import('$lib/widgets/publish');
	void publishWidgetSnapshot();
}
