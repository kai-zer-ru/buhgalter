import { writable } from 'svelte/store';
import {
	isLocalEntityKey,
	makeLocalKey,
	type AccountCreatePayload,
	type AccountStatusPayload,
	type AccountUpdatePayload,
	type BudgetPayload,
	type CategoryPayload,
	type CategoryUpdatePayload,
	type DebtPayload,
	type EntityKind,
	type OutboxEntry,
	type OutboxSnapshot,
	type TransactionPayload,
	type TransferPayload
} from '$lib/offline/types';

const STORAGE_KEY = 'buhgalter.outbox.v1';

export const outboxTick = writable(0);

function bump() {
	outboxTick.update((n) => n + 1);
	void import('$lib/offline/sync').then((m) => m.notifyOutboxChanged());
}

let entries: OutboxEntry[] = [];
let nextSeq = 1;

function persist() {
	if (typeof localStorage === 'undefined') return;
	try {
		const snap: OutboxSnapshot = { entries, nextSeq };
		localStorage.setItem(STORAGE_KEY, JSON.stringify(snap));
	} catch {
		// ignore quota
	}
}

function load() {
	if (typeof localStorage === 'undefined') return;
	try {
		const raw = localStorage.getItem(STORAGE_KEY);
		if (!raw) return;
		const snap = JSON.parse(raw) as OutboxSnapshot;
		entries = snap.entries ?? [];
		nextSeq = snap.nextSeq ?? 1;
	} catch {
		entries = [];
		nextSeq = 1;
	}
}

load();

export function resetOutboxForTests() {
	entries = [];
	nextSeq = 1;
	if (typeof localStorage !== 'undefined') {
		try {
			localStorage.removeItem(STORAGE_KEY);
		} catch {
			// ignore
		}
	}
	bump();
}

function findEntry(entityKey: string): OutboxEntry | undefined {
	return entries.find((e) => e.entityKey === entityKey);
}

function removeEntry(entityKey: string) {
	const before = entries.length;
	entries = entries.filter((e) => e.entityKey !== entityKey);
	if (entries.length !== before) {
		persist();
		bump();
	}
}

function pushEntry(entry: OutboxEntry) {
	entries.push(entry);
	persist();
	bump();
}

function clearFailed(entityKey: string) {
	const e = findEntry(entityKey);
	if (e?.failed) {
		delete e.failed;
		persist();
		bump();
	}
}

export function getOutboxEntries(): OutboxEntry[] {
	return [...entries].sort((a, b) => a.seq - b.seq);
}

export function hasPendingOutbox(): boolean {
	return entries.length > 0;
}

export function pendingOutboxCount(): number {
	return entries.filter((e) => !e.failed).length;
}

export function failedOutboxCount(): number {
	return entries.filter((e) => e.failed).length;
}

export function hasFailedOutbox(): boolean {
	return entries.some((e) => e.failed);
}

export function markOutboxFailed(entityKey: string, message: string) {
	const e = findEntry(entityKey);
	if (!e) return;
	e.failed = { message };
	persist();
	bump();
}

export function removeOutboxEntry(entityKey: string) {
	removeEntry(entityKey);
}

function enqueueServerMutation(
	entityKey: string,
	kind: EntityKind,
	op: 'update' | 'delete',
	payload?: OutboxEntry['payload']
) {
	const existing = findEntry(entityKey);
	if (op === 'delete') {
		if (existing?.op === 'update') {
			removeEntry(entityKey);
		}
		if (findEntry(entityKey)?.op === 'delete') return;
		pushEntry({
			entityKey,
			kind,
			op: 'delete',
			isLocalOnly: false,
			seq: nextSeq++
		});
		return;
	}

	if (existing?.op === 'delete') return;

	if (existing?.op === 'update' && existing.kind === kind) {
		existing.payload = payload;
		clearFailed(entityKey);
		persist();
		bump();
		return;
	}

	pushEntry({
		entityKey,
		kind,
		op: 'update',
		isLocalOnly: false,
		payload,
		seq: nextSeq++
	});
}

function enqueueLocalCreate(entityKey: string, kind: EntityKind, payload: OutboxEntry['payload']) {
	const existing = findEntry(entityKey);
	if (existing?.op === 'create' && existing.isLocalOnly) {
		existing.payload = payload;
		clearFailed(entityKey);
		persist();
		bump();
		return;
	}
	pushEntry({
		entityKey,
		kind,
		op: 'create',
		isLocalOnly: true,
		payload,
		seq: nextSeq++
	});
}

function enqueueLocalDelete(entityKey: string) {
	const existing = findEntry(entityKey);
	if (existing?.isLocalOnly && existing.op === 'create') {
		removeEntry(entityKey);
		return;
	}
}

export function enqueueTransactionCreate(
	localKey: string,
	payload: TransactionPayload
): OutboxEntry {
	enqueueLocalCreate(localKey, 'transaction', payload);
	return findEntry(localKey)!;
}

export function enqueueTransactionUpdate(entityKey: string, payload: TransactionPayload) {
	if (isLocalEntityKey(entityKey)) {
		enqueueLocalCreate(entityKey, 'transaction', payload);
		return;
	}
	enqueueServerMutation(entityKey, 'transaction', 'update', payload);
}

export function enqueueTransactionDelete(entityKey: string) {
	if (isLocalEntityKey(entityKey)) {
		enqueueLocalDelete(entityKey);
		return;
	}
	enqueueServerMutation(entityKey, 'transaction', 'delete');
}

export function enqueueTransferCreate(localKey: string, payload: TransferPayload): OutboxEntry {
	enqueueLocalCreate(localKey, 'transfer', payload);
	return findEntry(localKey)!;
}

export function enqueueTransferUpdate(entityKey: string, payload: TransferPayload) {
	if (isLocalEntityKey(entityKey)) {
		enqueueLocalCreate(entityKey, 'transfer', payload);
		return;
	}
	enqueueServerMutation(entityKey, 'transfer', 'update', payload);
}

export function enqueueTransferDelete(entityKey: string) {
	if (isLocalEntityKey(entityKey)) {
		enqueueLocalDelete(entityKey);
		return;
	}
	enqueueServerMutation(entityKey, 'transfer', 'delete');
}

export function enqueueCategoryCreate(localKey: string, payload: CategoryPayload): OutboxEntry {
	enqueueLocalCreate(localKey, 'category', payload);
	return findEntry(localKey)!;
}

export function enqueueCategoryUpdate(
	entityKey: string,
	payload: CategoryUpdatePayload & { type?: 'income' | 'expense' }
) {
	if (isLocalEntityKey(entityKey)) {
		const existing = findEntry(entityKey);
		const base =
			existing?.payload && 'type' in existing.payload
				? (existing.payload as CategoryPayload)
				: null;
		const merged: CategoryPayload = {
			name: payload.name,
			icon: payload.icon,
			sort_order: payload.sort_order ?? base?.sort_order,
			type: payload.type ?? base?.type ?? 'expense'
		};
		enqueueLocalCreate(entityKey, 'category', merged);
		return;
	}
	enqueueServerMutation(entityKey, 'category', 'update', payload);
}

export function enqueueCategoryDelete(entityKey: string) {
	if (isLocalEntityKey(entityKey)) {
		enqueueLocalDelete(entityKey);
		return;
	}
	enqueueServerMutation(entityKey, 'category', 'delete');
}

export function enqueueDebtCreate(localKey: string, payload: DebtPayload): OutboxEntry {
	enqueueLocalCreate(localKey, 'debt', payload);
	return findEntry(localKey)!;
}

export function enqueueDebtDelete(entityKey: string) {
	if (isLocalEntityKey(entityKey)) {
		enqueueLocalDelete(entityKey);
		return;
	}
	enqueueServerMutation(entityKey, 'debt', 'delete');
}

export function enqueueAccountCreate(localKey: string, payload: AccountCreatePayload): OutboxEntry {
	enqueueLocalCreate(localKey, 'account', payload);
	return findEntry(localKey)!;
}

export function enqueueAccountUpdate(entityKey: string, payload: AccountUpdatePayload) {
	if (isLocalEntityKey(entityKey)) {
		const existing = findEntry(entityKey);
		const base =
			existing?.payload && 'type' in existing.payload
				? (existing.payload as AccountCreatePayload)
				: null;
		const merged: AccountCreatePayload = {
			name: payload.name,
			type: base?.type ?? 'cash',
			bank_id: payload.bank_id ?? base?.bank_id,
			initial_balance: payload.initial_balance ?? base?.initial_balance ?? '0',
			credit_limit: payload.credit_limit ?? base?.credit_limit,
			payment_account_id:
				payload.payment_account_id === null
					? undefined
					: (payload.payment_account_id ?? base?.payment_account_id)
		};
		enqueueLocalCreate(entityKey, 'account', merged);
		return;
	}
	enqueueServerMutation(entityKey, 'account', 'update', payload);
}

export function enqueueAccountArchive(entityKey: string, transferToAccountId?: string) {
	if (isLocalEntityKey(entityKey)) {
		enqueueLocalDelete(entityKey);
		return;
	}
	const payload: AccountStatusPayload = {
		action: 'archive',
		transfer_to_account_id: transferToAccountId
	};
	enqueueServerMutation(entityKey, 'account', 'update', payload);
}

export function enqueueAccountUnarchive(entityKey: string) {
	const payload: AccountStatusPayload = { action: 'unarchive' };
	enqueueServerMutation(entityKey, 'account', 'update', payload);
}

export function enqueueBudgetCreate(localKey: string, payload: BudgetPayload): OutboxEntry {
	enqueueLocalCreate(localKey, 'budget', payload);
	return findEntry(localKey)!;
}

export function enqueueBudgetUpdate(entityKey: string, payload: BudgetPayload) {
	if (isLocalEntityKey(entityKey)) {
		enqueueLocalCreate(entityKey, 'budget', payload);
		return;
	}
	enqueueServerMutation(entityKey, 'budget', 'update', payload);
}

export function enqueueBudgetDelete(entityKey: string) {
	if (isLocalEntityKey(entityKey)) {
		enqueueLocalDelete(entityKey);
		return;
	}
	enqueueServerMutation(entityKey, 'budget', 'delete');
}

export { makeLocalKey, isLocalEntityKey };
