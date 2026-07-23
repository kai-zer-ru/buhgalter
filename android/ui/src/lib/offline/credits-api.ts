import {
	addCreditPayment as apiAddCreditPayment,
	completeCredit as apiCompleteCredit,
	deleteCredit as apiDeleteCredit,
	deleteCreditPayment as apiDeleteCreditPayment,
	updateCredit as apiUpdateCredit,
	updateCreditSchedule as apiUpdateCreditSchedule,
	ApiError,
	isTransientHttpError,
	type Credit
} from '$lib/api/client';
import {
	isConnectionError,
	markServerOffline,
	shouldTryServer
} from '$lib/offline/server-connectivity';
import { shouldUseOfflineQueue } from '$lib/offline/network';
import {
	onCreditDeleted,
	onCreditUpdated,
	touchBalancesAfterCreditMutation
} from '$lib/offline/ref-cache-mutations';
import {
	enqueueCreditComplete,
	enqueueCreditDelete,
	enqueueCreditDeletePayment,
	enqueueCreditMetaUpdate,
	enqueueCreditPay,
	enqueueCreditSchedule,
	makeLocalKey
} from '$lib/offline/store';
import type {
	CreditCompletePayload,
	CreditDeletePayload,
	CreditDeletePaymentPayload,
	CreditMetaUpdatePayload,
	CreditPayPayload,
	CreditSchedulePayload
} from '$lib/offline/types';
import { readRefCache } from '$lib/offline/ref-cache';
import { scheduleSyncOutbox } from '$lib/offline/sync';
import { fromCents, formatMoneyForInput } from '$lib/money';

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

function creditDetailPath(id: string): string {
	return `/api/v1/credits/${id}`;
}

function readCachedCredit(id: string): Credit | null {
	return readRefCache<Credit>(creditDetailPath(id));
}

function applyMetaOptimistic(credit: Credit, payload: CreditMetaUpdatePayload): Credit {
	const next = { ...credit };
	if (payload.name !== undefined) next.name = payload.name;
	if (payload.debit_account_id !== undefined) next.debit_account_id = payload.debit_account_id;
	if (payload.debit_time_local !== undefined) next.debit_time_local = payload.debit_time_local;
	if (payload.bank_id !== undefined) next.bank_id = payload.bank_id;
	next.updated_at = new Date().toISOString();
	return next;
}

function applyPayOptimistic(credit: Credit, payload: CreditPayPayload): Credit {
	const amountCents = Math.round(Number(payload.amount.replace(/\s/g, '').replace(',', '.')) * 100);
	const paid = Math.min(amountCents, credit.remaining_amount);
	const remaining = Math.max(0, credit.remaining_amount - paid);
	const schedule = (credit.schedule ?? []).map((p) => ({ ...p }));
	const pending = schedule.find((p) => p.kind === 'scheduled' && !p.is_applied);
	if (pending) {
		pending.is_applied = true;
		pending.amount = paid;
		pending.amount_display = formatMoneyForInput(fromCents(paid));
		pending.payment_date = payload.payment_date;
	}
	return {
		...credit,
		paid_amount: credit.paid_amount + paid,
		paid_amount_display: formatMoneyForInput(fromCents(credit.paid_amount + paid)),
		remaining_amount: remaining,
		remaining_amount_display: formatMoneyForInput(fromCents(remaining)),
		schedule,
		updated_at: new Date().toISOString()
	};
}

function applyCompleteOptimistic(credit: Credit): Credit {
	return {
		...credit,
		status: 'closed',
		remaining_amount: 0,
		remaining_amount_display: '0.00',
		closed_at: new Date().toISOString(),
		updated_at: new Date().toISOString()
	};
}

function applyScheduleOptimistic(credit: Credit, payload: CreditSchedulePayload): Credit {
	const byId = new Map(payload.payments.map((p) => [p.id, p.amount]));
	const schedule = (credit.schedule ?? []).map((row) => {
		const nextAmount = byId.get(row.id);
		if (nextAmount === undefined) return row;
		const cents = Math.round(Number(nextAmount.replace(/\s/g, '').replace(',', '.')) * 100);
		return {
			...row,
			amount: cents,
			amount_display: formatMoneyForInput(fromCents(cents))
		};
	});
	return { ...credit, schedule, updated_at: new Date().toISOString() };
}

function applyDeletePaymentOptimistic(credit: Credit, paymentId: string): Credit {
	const schedule = (credit.schedule ?? []).map((p) => {
		if (p.id !== paymentId) return p;
		return {
			...p,
			is_applied: false,
			transaction_id: null,
			transaction_kind: null
		};
	});
	const restored = credit.schedule?.find((p) => p.id === paymentId);
	const restoreCents = restored?.amount ?? 0;
	const remaining = credit.remaining_amount + restoreCents;
	const paid = Math.max(0, credit.paid_amount - restoreCents);
	return {
		...credit,
		schedule,
		remaining_amount: remaining,
		remaining_amount_display: formatMoneyForInput(fromCents(remaining)),
		paid_amount: paid,
		paid_amount_display: formatMoneyForInput(fromCents(paid)),
		updated_at: new Date().toISOString()
	};
}

export async function updateCredit(
	id: string,
	fields: {
		name?: string | null;
		debit_account_id?: string;
		debit_time_local?: string | null;
		bank_id?: string | null;
	}
): Promise<Credit> {
	const payload: CreditMetaUpdatePayload = { action: 'update', credit_id: id, ...fields };
	if (!shouldUseOfflineQueue()) {
		const credit = await apiUpdateCredit(id, fields);
		onCreditUpdated(credit);
		return credit;
	}
	if (await shouldTryServer()) {
		const res = await tryOnline(() => apiUpdateCredit(id, fields));
		if (res) {
			onCreditUpdated(res);
			scheduleSyncOutbox();
			return res;
		}
	}
	enqueueCreditMetaUpdate(payload);
	const cached = readCachedCredit(id);
	const optimistic = cached ? applyMetaOptimistic(cached, payload) : ({ id, ...fields } as Credit);
	onCreditUpdated(optimistic);
	return optimistic;
}

export async function addCreditPayment(
	id: string,
	body: { amount: string; payment_date: string; account_id?: string }
): Promise<Credit> {
	const payload: CreditPayPayload = { action: 'pay', credit_id: id, ...body };
	if (!shouldUseOfflineQueue()) {
		const credit = await apiAddCreditPayment(id, body);
		onCreditUpdated(credit);
		touchBalancesAfterCreditMutation();
		return credit;
	}
	if (await shouldTryServer()) {
		const res = await tryOnline(() => apiAddCreditPayment(id, body));
		if (res) {
			onCreditUpdated(res);
			touchBalancesAfterCreditMutation();
			scheduleSyncOutbox();
			return res;
		}
	}
	enqueueCreditPay(payload, makeLocalKey());
	const cached = readCachedCredit(id);
	const optimistic = cached ? applyPayOptimistic(cached, payload) : ({ id } as Credit);
	onCreditUpdated(optimistic);
	touchBalancesAfterCreditMutation();
	return optimistic;
}

export async function completeCredit(
	id: string,
	body: { affects_balance: boolean; payment_date: string }
): Promise<Credit> {
	const payload: CreditCompletePayload = { action: 'complete', credit_id: id, ...body };
	if (!shouldUseOfflineQueue()) {
		const credit = await apiCompleteCredit(id, body);
		onCreditUpdated(credit);
		touchBalancesAfterCreditMutation();
		return credit;
	}
	if (await shouldTryServer()) {
		const res = await tryOnline(() => apiCompleteCredit(id, body));
		if (res) {
			onCreditUpdated(res);
			touchBalancesAfterCreditMutation();
			scheduleSyncOutbox();
			return res;
		}
	}
	enqueueCreditComplete(payload);
	const cached = readCachedCredit(id);
	const optimistic = cached
		? applyCompleteOptimistic(cached)
		: ({ id, status: 'closed' } as Credit);
	onCreditUpdated(optimistic);
	touchBalancesAfterCreditMutation();
	return optimistic;
}

export async function updateCreditSchedule(
	creditId: string,
	body: { payments: { id: string; amount: string }[] }
): Promise<Credit> {
	const payload: CreditSchedulePayload = {
		action: 'schedule',
		credit_id: creditId,
		payments: body.payments
	};
	if (!shouldUseOfflineQueue()) {
		const credit = await apiUpdateCreditSchedule(creditId, body);
		onCreditUpdated(credit);
		return credit;
	}
	if (await shouldTryServer()) {
		const res = await tryOnline(() => apiUpdateCreditSchedule(creditId, body));
		if (res) {
			onCreditUpdated(res);
			scheduleSyncOutbox();
			return res;
		}
	}
	enqueueCreditSchedule(payload);
	const cached = readCachedCredit(creditId);
	const optimistic = cached
		? applyScheduleOptimistic(cached, payload)
		: ({ id: creditId } as Credit);
	onCreditUpdated(optimistic);
	return optimistic;
}

export async function deleteCreditPayment(creditId: string, paymentId: string): Promise<Credit> {
	const payload: CreditDeletePaymentPayload = {
		action: 'delete_payment',
		credit_id: creditId,
		payment_id: paymentId
	};
	if (!shouldUseOfflineQueue()) {
		const credit = await apiDeleteCreditPayment(creditId, paymentId);
		onCreditUpdated(credit);
		touchBalancesAfterCreditMutation();
		return credit;
	}
	if (await shouldTryServer()) {
		const res = await tryOnline(() => apiDeleteCreditPayment(creditId, paymentId));
		if (res) {
			onCreditUpdated(res);
			touchBalancesAfterCreditMutation();
			scheduleSyncOutbox();
			return res;
		}
	}
	enqueueCreditDeletePayment(payload);
	const cached = readCachedCredit(creditId);
	const optimistic = cached
		? applyDeletePaymentOptimistic(cached, paymentId)
		: ({ id: creditId } as Credit);
	onCreditUpdated(optimistic);
	touchBalancesAfterCreditMutation();
	return optimistic;
}

export async function deleteCredit(
	id: string,
	mode: 'cascade' | 'keep_transactions'
): Promise<void> {
	const payload: CreditDeletePayload = { action: 'delete', credit_id: id, mode };
	if (!shouldUseOfflineQueue()) {
		await apiDeleteCredit(id, mode);
		onCreditDeleted(id);
		touchBalancesAfterCreditMutation();
		return;
	}
	if (await shouldTryServer()) {
		const res = await tryOnline(() => apiDeleteCredit(id, mode));
		if (res !== null) {
			onCreditDeleted(id);
			touchBalancesAfterCreditMutation();
			scheduleSyncOutbox();
			return;
		}
	}
	enqueueCreditDelete(payload);
	onCreditDeleted(id);
	touchBalancesAfterCreditMutation();
}
