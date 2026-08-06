import type { NotificationHistoryItem, RawBankNotification } from './types';

const STORAGE_KEY = 'buhgalter.notification_intercept.history.js.v1';
const MAX_ITEMS = 80;

function read(): NotificationHistoryItem[] {
	try {
		if (typeof localStorage === 'undefined') return [];
		const raw = localStorage.getItem(STORAGE_KEY);
		if (!raw) return [];
		const parsed = JSON.parse(raw) as NotificationHistoryItem[];
		return Array.isArray(parsed) ? parsed : [];
	} catch {
		return [];
	}
}

function write(items: NotificationHistoryItem[]): void {
	try {
		if (typeof localStorage === 'undefined') return;
		localStorage.setItem(STORAGE_KEY, JSON.stringify(items.slice(0, MAX_ITEMS)));
	} catch {
		// ignore
	}
}

/** Mirror raw notifications into JS history when the native queue is processed. */
export function appendLocalHistoryFromRaw(
	raw: RawBankNotification,
	extra: { inAllowlist: boolean; queued: boolean }
): void {
	const item: NotificationHistoryItem = {
		packageName: raw.packageName,
		title: raw.title,
		text: raw.text,
		bigText: raw.bigText,
		postedAt: raw.postedAt,
		dedupeKey: raw.dedupeKey,
		inAllowlist: extra.inAllowlist,
		queued: extra.queued
	};
	const prev = read();
	if (item.dedupeKey && prev.some((p) => p.dedupeKey === item.dedupeKey)) {
		return;
	}
	write([item, ...prev]);
}

export function listLocalHistory(): NotificationHistoryItem[] {
	return read();
}

export function clearLocalHistory(): void {
	try {
		if (typeof localStorage !== 'undefined') {
			localStorage.removeItem(STORAGE_KEY);
		}
	} catch {
		// ignore
	}
}

export function mergeHistoryLists(
	...lists: NotificationHistoryItem[][]
): NotificationHistoryItem[] {
	const seen = new Set<string>();
	const out: NotificationHistoryItem[] = [];
	for (const list of lists) {
		for (const row of list) {
			const key = row.dedupeKey || `${row.packageName}|${row.postedAt}|${row.title}|${row.text}`;
			if (seen.has(key)) continue;
			seen.add(key);
			out.push(row);
			if (out.length >= MAX_ITEMS) return out;
		}
	}
	return out;
}
