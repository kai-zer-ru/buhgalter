import { get, writable } from 'svelte/store';
import { user } from '$lib/stores/auth';
import type { InterceptDraft, ParsedPurchase } from './types';

const STORAGE_PREFIX = 'buhgalter.notification_intercept.drafts.v1:';
const MAX_DRAFTS = 50;

/** Tick so UI can react to draft mutations. */
export const interceptDraftsTick = writable(0);

function storageKey(userId: string): string {
	return `${STORAGE_PREFIX}${userId}`;
}

function bump(): void {
	interceptDraftsTick.update((n) => n + 1);
}

const memoryDrafts = new Map<string, string>();
const parsedDraftsCache = new Map<string, InterceptDraft[]>();

function readDrafts(userId: string): InterceptDraft[] {
	const cached = parsedDraftsCache.get(userId);
	if (cached) return cached;
	try {
		let raw: string | null = null;
		try {
			if (typeof localStorage !== 'undefined') {
				raw = localStorage.getItem(storageKey(userId));
			}
		} catch {
			raw = null;
		}
		if (raw == null) raw = memoryDrafts.get(storageKey(userId)) ?? null;
		if (!raw) {
			parsedDraftsCache.set(userId, []);
			return [];
		}
		const parsed = JSON.parse(raw) as InterceptDraft[];
		const list = Array.isArray(parsed) ? parsed : [];
		parsedDraftsCache.set(userId, list);
		return list;
	} catch {
		parsedDraftsCache.set(userId, []);
		return [];
	}
}

function writeDrafts(userId: string, drafts: InterceptDraft[]): void {
	const payload = JSON.stringify(drafts.slice(0, MAX_DRAFTS));
	memoryDrafts.set(storageKey(userId), payload);
	parsedDraftsCache.set(userId, drafts.slice(0, MAX_DRAFTS));
	try {
		if (typeof localStorage !== 'undefined') {
			localStorage.setItem(storageKey(userId), payload);
		}
	} catch {
		// ignore
	}
	bump();
}

function newId(): string {
	if (typeof crypto !== 'undefined' && 'randomUUID' in crypto) {
		return crypto.randomUUID();
	}
	return `draft-${Date.now()}-${Math.random().toString(36).slice(2, 10)}`;
}

export function listInterceptDrafts(userId?: string | null): InterceptDraft[] {
	const id = userId ?? get(user)?.id;
	if (!id) return [];
	return readDrafts(id);
}

export function countInterceptDrafts(userId?: string | null): number {
	return listInterceptDrafts(userId).length;
}

export function getInterceptDraft(draftId: string, userId?: string | null): InterceptDraft | null {
	return listInterceptDrafts(userId).find((d) => d.id === draftId) ?? null;
}

/** Drop duplicate push↔SMS drafts for the same purchase within this window. */
const SEMANTIC_DEDUP_WINDOW_MS = 2 * 60 * 60 * 1000;

function isSemanticDuplicate(
	existing: InterceptDraft,
	incoming: ParsedPurchase,
	extraMerchantName?: string
): boolean {
	if (existing.parsed.bankId !== incoming.bankId) return false;
	if (existing.parsed.amount !== incoming.amount) return false;
	const existingKind = existing.parsed.kind ?? 'purchase';
	const incomingKind = incoming.kind ?? 'purchase';
	if (existingKind !== incomingKind) return false;
	if (existingKind === 'cancel') return false;

	if (existing.parsed.last4 && incoming.last4 && existing.parsed.last4 !== incoming.last4) {
		return false;
	}

	const a = normMerchant(existing.merchantName || existing.parsed.merchantText || '');
	const b = normMerchant(extraMerchantName || incoming.merchantText || '');
	if (a && b && a !== b && !a.includes(b) && !b.includes(a)) {
		return false;
	}

	const existingAt = Date.parse(existing.parsed.occurredAt) || Date.parse(existing.createdAt) || 0;
	const incomingAt = Date.parse(incoming.occurredAt) || Date.now();
	return Math.abs(existingAt - incomingAt) <= SEMANTIC_DEDUP_WINDOW_MS;
}

export function addInterceptDraft(
	parsed: ParsedPurchase,
	extra: { accountId?: string; merchantId?: string; merchantName?: string },
	userId?: string | null
): InterceptDraft | null {
	const id = userId ?? get(user)?.id;
	if (!id) return null;
	const drafts = readDrafts(id);
	if (drafts.some((d) => d.parsed.rawHash === parsed.rawHash)) {
		return null;
	}
	if (drafts.some((d) => isSemanticDuplicate(d, parsed, extra.merchantName))) {
		return null;
	}
	const draft: InterceptDraft = {
		id: newId(),
		createdAt: new Date().toISOString(),
		parsed,
		accountId: extra.accountId,
		merchantId: extra.merchantId,
		merchantName: extra.merchantName
	};
	writeDrafts(id, [draft, ...drafts]);
	return draft;
}

/** Remove draft without creating a transaction (subscriptions / false positives). */
export function deleteInterceptDraft(draftId: string, userId?: string | null): boolean {
	const id = userId ?? get(user)?.id;
	if (!id) return false;
	const drafts = readDrafts(id);
	const next = drafts.filter((d) => d.id !== draftId);
	if (next.length === drafts.length) return false;
	writeDrafts(id, next);
	return true;
}

function normMerchant(s: string): string {
	return s
		.toLowerCase()
		.replace(/[!?.…]+/g, ' ')
		.replace(/\s+/g, ' ')
		.trim();
}

const CANCEL_MATCH_WINDOW_MS = 48 * 60 * 60 * 1000;

/**
 * Drop a purchase draft that matches a cancel/refund push (same amount + bank,
 * preferably merchant / last4). Returns how many drafts were removed (0 or 1).
 */
export function removeDraftMatchingCancel(cancel: ParsedPurchase, userId?: string | null): number {
	const id = userId ?? get(user)?.id;
	if (!id) return 0;
	const drafts = readDrafts(id);
	const cancelAt = Date.parse(cancel.occurredAt) || Date.now();
	const cancelMerchant = normMerchant(cancel.merchantText);

	let bestIdx = -1;
	let bestScore = -1;
	for (let i = 0; i < drafts.length; i++) {
		const d = drafts[i];
		if (d.parsed.bankId !== cancel.bankId) continue;
		if (d.parsed.amount !== cancel.amount) continue;
		// Only purchase drafts are cancelled by refund pushes — not income.
		if (d.parsed.kind === 'cancel' || d.parsed.kind === 'income') continue;

		const draftAt = Date.parse(d.parsed.occurredAt) || Date.parse(d.createdAt) || 0;
		if (Math.abs(cancelAt - draftAt) > CANCEL_MATCH_WINDOW_MS) continue;

		if (cancel.last4 && d.parsed.last4 && cancel.last4 !== d.parsed.last4) continue;

		const draftMerchant = normMerchant(d.merchantName || d.parsed.merchantText || '');
		let score = 1;
		if (cancel.last4 && d.parsed.last4 && cancel.last4 === d.parsed.last4) score += 3;
		if (cancelMerchant && draftMerchant) {
			if (
				cancelMerchant === draftMerchant ||
				cancelMerchant.includes(draftMerchant) ||
				draftMerchant.includes(cancelMerchant)
			) {
				score += 4;
			} else {
				continue; // both have merchant names but they disagree
			}
		}
		// Prefer newest draft when several match.
		score += Math.min(2, Math.max(0, 2 - i * 0.01));
		if (score > bestScore) {
			bestScore = score;
			bestIdx = i;
		}
	}

	if (bestIdx < 0) return 0;
	const next = drafts.filter((_, i) => i !== bestIdx);
	writeDrafts(id, next);
	return 1;
}

export function clearInterceptDraftsForTests(userId?: string): void {
	const id = userId ?? get(user)?.id;
	if (id) {
		memoryDrafts.delete(storageKey(id));
		parsedDraftsCache.delete(id);
		try {
			if (typeof localStorage !== 'undefined') {
				localStorage.removeItem(storageKey(id));
			}
		} catch {
			// ignore
		}
	} else {
		memoryDrafts.clear();
		parsedDraftsCache.clear();
	}
	bump();
}
