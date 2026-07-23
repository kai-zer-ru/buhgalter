import {
	createDebt as apiCreateDebt,
	deleteDebt as apiDeleteDebt,
	settleDebt as apiSettleDebt,
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
import { onDebtCreated, onDebtDeleted, onDebtUpdated } from '$lib/offline/ref-cache-mutations';
import {
	enqueueDebtCreate,
	enqueueDebtDelete,
	enqueueDebtSettle,
	getOutboxEntry,
	makeLocalKey,
	patchLocalDebtCreateAmount,
	removeOutboxEntry
} from '$lib/offline/store';
import type { DebtPayload, DebtSettlePayload } from '$lib/offline/types';
import { isLocalEntityKey } from '$lib/offline/types';
import { readRefCache } from '$lib/offline/ref-cache';
import { scheduleSyncOutbox } from '$lib/offline/sync';
import { formatMoneyForInput, fromCents } from '$lib/money';

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

function settledOptimistic(debt: Debt, payload: DebtSettlePayload): Debt {
	const settleCents = payload.amount !== undefined ? amountToCents(payload.amount) : debt.amount;
	const remaining = Math.max(0, debt.amount - settleCents);
	if (remaining <= 0) {
		return {
			...debt,
			amount: 0,
			amount_display: '0.00',
			is_settled: true,
			settled_at: payload.settled_at,
			is_overdue: false
		};
	}
	return {
		...debt,
		amount: remaining,
		amount_display: formatMoneyForInput(fromCents(remaining)),
		is_settled: false,
		settled_at: null
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

export async function settleDebt(
	id: string,
	body: {
		amount?: string;
		settled_at: string;
		affects_balance: boolean;
		account_id?: string;
	}
): Promise<Debt> {
	const payload: DebtSettlePayload = { action: 'settle', ...body };
	if (!shouldUseOfflineQueue()) {
		const debt = await apiSettleDebt(id, body);
		onDebtUpdated(debt);
		return debt;
	}
	if (await shouldTryServer()) {
		const res = await tryOnline(() => apiSettleDebt(id, body));
		if (res) {
			onDebtUpdated(res);
			scheduleSyncOutbox();
			return res;
		}
	}

	if (isLocalEntityKey(id)) {
		const entry = getOutboxEntry(id);
		const createPayload =
			entry?.op === 'create' && entry.kind === 'debt' ? (entry.payload as DebtPayload) : null;
		if (!createPayload) {
			throw new ApiError('OFFLINE', 'Local debt not found in outbox', 0);
		}
		const currentCents = amountToCents(createPayload.amount);
		const settleCents = body.amount !== undefined ? amountToCents(body.amount) : currentCents;
		if (settleCents >= currentCents) {
			removeOutboxEntry(id);
			const settled = settledOptimistic(localDebt(id, createPayload), payload);
			onDebtUpdated(settled);
			return settled;
		}
		const remaining = currentCents - settleCents;
		const remainingDisplay = formatMoneyForInput(fromCents(remaining));
		patchLocalDebtCreateAmount(id, remainingDisplay);
		const partial = {
			...localDebt(id, { ...createPayload, amount: remainingDisplay }),
			amount: remaining,
			amount_display: remainingDisplay
		};
		onDebtUpdated(partial);
		return partial;
	}

	enqueueDebtSettle(id, payload);
	const cached =
		readRefCache<Debt[]>('/api/v1/debts?settled=false')?.find((d) => d.id === id) ??
		readRefCache<Debt[]>('/api/v1/debts?settled=true')?.find((d) => d.id === id);
	const optimistic = cached
		? settledOptimistic(cached, payload)
		: ({
				id,
				debtor_id: '',
				debtor_name: '',
				direction: 'lent' as const,
				amount: 0,
				amount_display: '0.00',
				affects_balance: body.affects_balance,
				debt_date: '',
				due_date: '',
				description: null,
				transaction_id: null,
				is_settled: true,
				settled_at: body.settled_at,
				is_overdue: false,
				created_at: new Date().toISOString(),
				account_id: body.account_id ?? null
			} satisfies Debt);
	onDebtUpdated(optimistic);
	return optimistic;
}
