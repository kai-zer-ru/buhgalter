import type { Transaction } from '$lib/api/client';
import { isCurrentMonthApiDatetime, isFutureApiDatetime } from '$lib/dates';
import { toCents } from '$lib/money';
import { getOutboxEntries } from '$lib/offline/store';
import type { OutboxEntry, TransactionPayload, TransferPayload } from '$lib/offline/types';
import {
	findTransferCommissionKopecks,
	lookupServerTransaction
} from '$lib/offline/transaction-index';

export type AccountDeltas = {
	balance: Record<string, number>;
	forecast: Record<string, number>;
};

function emptyDeltas(): AccountDeltas {
	return { balance: {}, forecast: {} };
}

function addDelta(
	target: AccountDeltas,
	accountId: string,
	balanceDelta: number,
	forecastDelta: number
): void {
	if (balanceDelta) {
		target.balance[accountId] = (target.balance[accountId] ?? 0) + balanceDelta;
	}
	if (forecastDelta) {
		target.forecast[accountId] = (target.forecast[accountId] ?? 0) + forecastDelta;
	}
}

function mergeDeltas(into: AccountDeltas, from: AccountDeltas): void {
	for (const [id, delta] of Object.entries(from.balance)) {
		addDelta(into, id, delta, 0);
	}
	for (const [id, delta] of Object.entries(from.forecast)) {
		addDelta(into, id, 0, delta);
	}
}

function subtractDeltas(into: AccountDeltas, from: AccountDeltas): void {
	for (const [id, delta] of Object.entries(from.balance)) {
		addDelta(into, id, -delta, 0);
	}
	for (const [id, delta] of Object.entries(from.forecast)) {
		addDelta(into, id, 0, -delta);
	}
}

function signedAmount(type: 'income' | 'expense', amountKopecks: number): number {
	return type === 'income' ? amountKopecks : -amountKopecks;
}

function effectForManualOrFuture(
	accountId: string,
	signedKopecks: number,
	isManual: boolean,
	isFutureInMonth: boolean
): AccountDeltas {
	const out = emptyDeltas();
	if (isManual) {
		addDelta(out, accountId, signedKopecks, signedKopecks);
	} else if (isFutureInMonth) {
		addDelta(out, accountId, 0, signedKopecks);
	}
	return out;
}

export function effectFromTransactionPayload(
	payload: TransactionPayload,
	tz: string
): AccountDeltas {
	const amount = toCents(payload.amount);
	const signed = signedAmount(payload.type, amount);
	const future = isFutureApiDatetime(payload.transaction_date, tz);
	const inMonth = isCurrentMonthApiDatetime(payload.transaction_date, tz);
	return effectForManualOrFuture(payload.account_id, signed, !future, future && inMonth);
}

export function effectFromTransferPayload(payload: TransferPayload, tz: string): AccountDeltas {
	const amount = toCents(payload.amount);
	const commission = toCents(payload.commission ?? '0');
	const fromTotal = amount + commission;
	const future = isFutureApiDatetime(payload.transaction_date, tz);
	const inMonth = isCurrentMonthApiDatetime(payload.transaction_date, tz);
	const out = emptyDeltas();
	if (!future) {
		addDelta(out, payload.from_account_id, -fromTotal, -fromTotal);
		addDelta(out, payload.to_account_id, amount, amount);
	} else if (inMonth) {
		addDelta(out, payload.from_account_id, 0, -fromTotal);
		addDelta(out, payload.to_account_id, 0, amount);
	}
	return out;
}

export function effectFromServerTransaction(tx: Transaction, tz: string): AccountDeltas {
	if (tx.type === 'transfer') {
		if (!tx.transfer_is_out || !tx.transfer_account_id) return emptyDeltas();
		const commission = tx.transfer_group_id
			? findTransferCommissionKopecks(tx.transfer_group_id)
			: 0;
		const isManual = tx.kind !== 'future';
		const inMonth = isCurrentMonthApiDatetime(tx.transaction_date, tz);
		const out = emptyDeltas();
		if (isManual) {
			addDelta(out, tx.account_id, -(tx.amount + commission), -(tx.amount + commission));
			addDelta(out, tx.transfer_account_id, tx.amount, tx.amount);
		} else if (inMonth) {
			addDelta(out, tx.account_id, 0, -(tx.amount + commission));
			addDelta(out, tx.transfer_account_id, 0, tx.amount);
		}
		return out;
	}
	const signed = signedAmount(tx.type as 'income' | 'expense', tx.amount);
	const isManual = tx.kind !== 'future';
	const inMonth = isCurrentMonthApiDatetime(tx.transaction_date, tz);
	return effectForManualOrFuture(tx.account_id, signed, isManual, !isManual && inMonth);
}

function effectFromOutboxEntry(entry: OutboxEntry, tz: string): AccountDeltas | null {
	if (!entry.payload || entry.op === 'delete') return null;
	if (entry.kind === 'transaction') {
		return effectFromTransactionPayload(entry.payload as TransactionPayload, tz);
	}
	return effectFromTransferPayload(entry.payload as TransferPayload, tz);
}

function baselineEffect(entry: OutboxEntry, tz: string): AccountDeltas | null {
	if (entry.isLocalOnly && entry.op === 'update') {
		return null;
	}
	if (entry.op === 'delete' || entry.op === 'update') {
		const tx = lookupServerTransaction(entry.entityKey);
		if (tx) return effectFromServerTransaction(tx, tz);
	}
	return null;
}

/** Net balance/forecast deltas from pending outbox (relative to last server snapshot). */
export function computeOutboxAccountDeltas(tz: string): AccountDeltas {
	const net = emptyDeltas();
	for (const entry of getOutboxEntries()) {
		if (entry.op === 'delete') {
			const old = baselineEffect(entry, tz);
			if (old) subtractDeltas(net, old);
			continue;
		}
		if (entry.op === 'update') {
			const old = baselineEffect(entry, tz);
			if (old) subtractDeltas(net, old);
			const next = effectFromOutboxEntry(entry, tz);
			if (next) mergeDeltas(net, next);
			continue;
		}
		if (entry.op === 'create') {
			const created = effectFromOutboxEntry(entry, tz);
			if (created) mergeDeltas(net, created);
		}
	}
	return net;
}

export function applyAccountDeltas(
	balanceKopecks: number,
	forecastKopecks: number,
	accountId: string,
	deltas: AccountDeltas
): { balance: number; forecast: number } {
	return {
		balance: balanceKopecks + (deltas.balance[accountId] ?? 0),
		forecast: forecastKopecks + (deltas.forecast[accountId] ?? 0)
	};
}
