import {
	createDebt as apiCreateDebt,
	deleteDebt as apiDeleteDebt,
	ApiError,
	isTransientHttpError,
	type Debt
} from '$lib/api/client';
import {
	isConnectionError,
	markServerOffline,
	shouldTryServer
} from '$lib/offline/server-connectivity';
import { shouldUseOfflineQueue } from '$lib/offline/network';
import { onDebtCreated, onDebtDeleted } from '$lib/offline/ref-cache-mutations';
import { enqueueDebtCreate, enqueueDebtDelete, makeLocalKey } from '$lib/offline/store';
import type { DebtPayload } from '$lib/offline/types';
import { isLocalEntityKey } from '$lib/offline/types';
import { scheduleSyncOutbox } from '$lib/offline/sync';

function isOfflineError(err: unknown): boolean {
	return isConnectionError(err) || (err instanceof ApiError && isTransientHttpError(err.status));
}

async function tryOnline<T>(fn: () => Promise<T>): Promise<T | null> {
	try {
		return await fn();
	} catch (err) {
		if (isOfflineError(err)) {
			markServerOffline();
			return null;
		}
		throw err;
	}
}

function amountToCents(amount: string): number {
	const n = Number(amount.replace(/\s/g, '').replace(',', '.'));
	return Math.round(n * 100);
}

function localDebt(id: string, payload: DebtPayload): Debt {
	const ts = new Date().toISOString();
	const debtorId = payload.debtor_id ?? makeLocalKey();
	return {
		id,
		debtor_id: debtorId,
		debtor_name: payload.debtor_name ?? '',
		direction: payload.direction,
		amount: amountToCents(payload.amount),
		amount_display: payload.amount,
		affects_balance: payload.affects_balance,
		debt_date: payload.debt_date,
		due_date: payload.due_date,
		description: payload.description ?? null,
		transaction_id: null,
		is_settled: false,
		settled_at: null,
		is_overdue: false,
		created_at: ts,
		account_id: payload.account_id ?? null
	};
}

export async function createDebt(payload: DebtPayload): Promise<Debt> {
	if (!shouldUseOfflineQueue()) {
		const debt = await apiCreateDebt(payload);
		onDebtCreated(debt);
		return debt;
	}
	if (await shouldTryServer()) {
		const res = await tryOnline(() => apiCreateDebt(payload));
		if (res) {
			onDebtCreated(res);
			scheduleSyncOutbox();
			return res;
		}
	}
	const localKey = makeLocalKey();
	enqueueDebtCreate(localKey, payload);
	const debt = localDebt(localKey, payload);
	onDebtCreated(debt);
	return debt;
}

export async function deleteDebt(id: string): Promise<void> {
	if (!shouldUseOfflineQueue()) {
		await apiDeleteDebt(id);
		onDebtDeleted(id);
		return;
	}
	if (isLocalEntityKey(id)) {
		enqueueDebtDelete(id);
		onDebtDeleted(id);
		return;
	}
	if (await shouldTryServer()) {
		const res = await tryOnline(() => apiDeleteDebt(id));
		if (res !== null) {
			onDebtDeleted(id);
			scheduleSyncOutbox();
			return;
		}
	}
	enqueueDebtDelete(id);
	onDebtDeleted(id);
}
