import { get, writable } from 'svelte/store';
import { user } from '$lib/stores/auth';
import { isWalletId, isWalletPackage } from './banks';
import { countVisibleInterceptItems } from './draft-transfer';
import type { InterceptDraft, ParsedPurchase } from './types';

function isWalletSource(parsed: ParsedPurchase): boolean {
	return isWalletId(parsed.bankId) || isWalletPackage(parsed.packageName);
}

/** Same purchase from bank app and wallet (or matching last4). */
function samePurchaseSource(a: ParsedPurchase, b: ParsedPurchase): boolean {
	if (a.bankId === b.bankId) return true;
	if (a.last4 && b.last4 && a.last4 === b.last4) return true;
	return isWalletSource(a) || isWalletSource(b);
}

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
	return countVisibleInterceptItems(listInterceptDrafts(userId));
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
	if (!samePurchaseSource(existing.parsed, incoming)) return false;
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
	extra: {
		accountId?: string;
		merchantId?: string;
		merchantName?: string;
		categoryId?: string;
		subcategoryId?: string;
	},
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
		merchantName: extra.merchantName,
		categoryId: extra.categoryId,
		subcategoryId: extra.subcategoryId
	};
	writeDrafts(id, [draft, ...drafts]);
	return draft;
}

/** Patch fields on an existing draft (e.g. category after async suggest). */
export function updateInterceptDraft(
	draftId: string,
	patch: Partial<
		Pick<
			InterceptDraft,
			'accountId' | 'merchantId' | 'merchantName' | 'categoryId' | 'subcategoryId'
		>
	>,
	userId?: string | null
): InterceptDraft | null {
	const id = userId ?? get(user)?.id;
	if (!id || !draftId) return null;
	const drafts = readDrafts(id);
	const idx = drafts.findIndex((d) => d.id === draftId);
	if (idx < 0) return null;
	const next = { ...drafts[idx], ...patch };
	const list = [...drafts];
	list[idx] = next;
	writeDrafts(id, list);
	return next;
}

/** Remove draft without creating a transaction (subscriptions / false positives). */
export function deleteInterceptDraft(draftId: string, userId?: string | null): boolean {
	return deleteInterceptDrafts([draftId], userId) > 0;
}

/** Drop several drafts (e.g. expense+income pair after creating a transfer). */
export function deleteInterceptDrafts(draftIds: string[], userId?: string | null): number {
	const id = userId ?? get(user)?.id;
	if (!id || draftIds.length === 0) return 0;
	const drop = new Set(draftIds);
	const drafts = readDrafts(id);
	const next = drafts.filter((d) => !drop.has(d.id));
	const removed = drafts.length - next.length;
	if (removed === 0) return 0;
	writeDrafts(id, next);
	return removed;
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
		if (!samePurchaseSource(d.parsed, cancel)) continue;
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

export function clearInterceptDrafts(userId?: string | null): void {
	const id = userId ?? get(user)?.id;
	if (id) {
		writeDrafts(id, []);
		return;
	}
	memoryDrafts.clear();
	parsedDraftsCache.clear();
	bump();
}

export function clearInterceptDraftsForTests(userId?: string): void {
	clearInterceptDrafts(userId);
}
