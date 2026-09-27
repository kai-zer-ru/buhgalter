/** Advance / sync GET /sync/transaction-changes cursor for web after reconnect. */

import { listTransactionChanges } from '$lib/api/client';
import { clearRefCache, notifyRealtimeInvalidate } from '$lib/ref-cache';

export const TX_CHANGES_CURSOR_KEY = 'buhgalter.web_tx_changes_cursor';

function storageGet(key: string): string | null {
	if (typeof localStorage === 'undefined') return null;
	try {
		return localStorage.getItem(key);
	} catch {
		return null;
	}
}

function storageSet(key: string, value: string): void {
	if (typeof localStorage === 'undefined') return;
	try {
		localStorage.setItem(key, value);
	} catch {
		// ignore
	}
}

function readCursor(): number | null {
	const raw = storageGet(TX_CHANGES_CURSOR_KEY);
	if (raw === null || raw === '') return null;
	const n = Number(raw);
	return Number.isFinite(n) && n >= 0 ? n : null;
}

function writeCursor(sinceId: number): void {
	storageSet(TX_CHANGES_CURSOR_KEY, String(sinceId));
}

/** Pull pages until has_more is false; returns whether any change was seen. */
async function pullFrom(sinceId: number): Promise<{ sinceId: number; hadChanges: boolean }> {
	let cursor = sinceId;
	let hadChanges = false;
	for (let i = 0; i < 40; i++) {
		const res = await listTransactionChanges({ since_id: cursor, limit: 200 });
		if (res.changes?.length) hadChanges = true;
		cursor = res.since_id;
		if (!res.has_more) break;
	}
	return { sinceId: cursor, hadChanges };
}

/** First connect: move cursor to tip without invalidating UI. */
export async function seedTransactionChangesCursor(): Promise<void> {
	if (readCursor() !== null) return;
	try {
		const { sinceId } = await pullFrom(0);
		writeCursor(sinceId);
	} catch {
		// offline / not ready — next reconnect will refetch open screen
	}
}

/**
 * After WebSocket reconnect: advance change feed and soft-reload open screens
 * so the tab does not keep a hole from the outage window.
 */
export async function catchUpAfterReconnect(): Promise<void> {
	const stored = readCursor();
	try {
		const from = stored ?? 0;
		const result = await pullFrom(from);
		writeCursor(result.sinceId);
	} catch {
		// still soft-refetch below
	}
	// Soft-refetch open page after outage (balances / non-tx entities).
	clearRefCache();
	notifyRealtimeInvalidate();
}

export function resetTxChangesCursorForTests(): void {
	if (typeof localStorage === 'undefined') return;
	try {
		localStorage.removeItem(TX_CHANGES_CURSOR_KEY);
	} catch {
		// ignore
	}
}
