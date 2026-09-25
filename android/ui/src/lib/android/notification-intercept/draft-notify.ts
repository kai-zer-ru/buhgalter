import { get } from 'svelte/store';
import { createTransaction, createTransfer } from '$lib/offline/transactions-api';
import { user } from '$lib/stores/auth';
import {
	listUniqueTransferPairs,
	transferPrefillFromDrafts,
	unpairedInterceptDrafts,
	type InterceptTransferPair
} from './draft-transfer';
import { deleteInterceptDraft, deleteInterceptDrafts, listInterceptDrafts } from './drafts';
import { draftTxType } from './prefill';
import {
	clearDraftNotifications,
	consumeDraftNotifySync,
	removeDraftNotification,
	setShadeNotificationsEnabled,
	syncDraftNotifications
} from './plugin';
import { loadInterceptSettings } from './settings';
import type { InterceptDraft } from './types';

/** Shape mirrored into native SharedPreferences for shade Accept. */
export type NativeDraftPayload = {
	id: string;
	type: 'expense' | 'income' | 'transfer';
	amount: string;
	occurredAt: string;
	accountId?: string;
	merchantId?: string;
	merchantName?: string;
	merchantText?: string;
	description?: string;
	categoryId?: string;
	subcategoryId?: string;
	/** Transfer pair sides (type === 'transfer'). */
	fromAccountId?: string;
	toAccountId?: string;
	fromDraftId?: string;
	toDraftId?: string;
};

export function shadeTransferItemId(fromId: string, toId: string): string {
	return `xfer:${fromId}:${toId}`;
}

export function toNativeDraftPayload(draft: InterceptDraft): NativeDraftPayload {
	const type = draftTxType(draft);
	const merchantName = (draft.merchantName || draft.parsed.merchantText || '').trim() || undefined;
	const description =
		type === 'income' && !draft.merchantId && merchantName
			? merchantName.slice(0, 2000)
			: undefined;
	return {
		id: draft.id,
		type,
		amount: draft.parsed.amount,
		occurredAt: draft.parsed.occurredAt,
		accountId: draft.accountId,
		merchantId: draft.merchantId,
		merchantName: draft.merchantId ? undefined : type === 'income' ? undefined : draft.merchantName,
		merchantText: draft.parsed.merchantText,
		description,
		categoryId: draft.categoryId,
		subcategoryId: draft.subcategoryId
	};
}

export function toNativeTransferPayload(pair: InterceptTransferPair): NativeDraftPayload {
	const prefill = transferPrefillFromDrafts(pair.from, pair.to);
	return {
		id: shadeTransferItemId(pair.from.id, pair.to.id),
		type: 'transfer',
		amount: pair.from.parsed.amount,
		occurredAt: prefill?.occurredAt ?? pair.from.parsed.occurredAt,
		fromAccountId: pair.from.accountId,
		toAccountId: pair.to.accountId,
		fromDraftId: pair.from.id,
		toDraftId: pair.to.id
	};
}

/**
 * Shade items match the drafts UI: one card per unique transfer pair + unpaired drafts.
 */
export function buildShadeNotifyItems(drafts: InterceptDraft[]): NativeDraftPayload[] {
	const pairs = listUniqueTransferPairs(drafts);
	const unpaired = unpairedInterceptDrafts(drafts);
	return [...pairs.map(toNativeTransferPayload), ...unpaired.map(toNativeDraftPayload)];
}

/** Push current localStorage drafts + shade flag into native. */
export async function syncDraftNotifyToNative(userId?: string | null): Promise<void> {
	const id = userId ?? get(user)?.id;
	const settings = loadInterceptSettings(id);
	await setShadeNotificationsEnabled(settings.enabled && settings.shadeNotifications !== false);
	if (!id || !settings.enabled || settings.shadeNotifications === false) {
		await clearDraftNotifications();
		return;
	}
	const drafts = listInterceptDrafts(id);
	await syncDraftNotifications(buildShadeNotifyItems(drafts));
}

export async function notifyDraftRemoved(draftId: string): Promise<void> {
	await removeDraftNotification(draftId);
}

export async function notifyDraftsRemoved(draftIds: string[]): Promise<void> {
	for (const id of draftIds) {
		await removeDraftNotification(id);
	}
}

type OfflineAcceptPayload = {
	kind?: 'transaction' | 'transfer';
	draftId?: string;
	draftIds?: string[];
	account_id?: string;
	from_account_id?: string;
	to_account_id?: string;
	type?: 'income' | 'expense';
	amount: string;
	transaction_date: string;
	category_id?: string;
	subcategory_id?: string;
	merchant_id?: string;
	merchant_name?: string;
	description?: string;
};

/**
 * Apply Accept/Reject side-effects from native shade actions into JS drafts / outbox.
 */
export async function applyNativeDraftNotifySync(userId?: string | null): Promise<{
	removed: number;
	created: number;
}> {
	const id = userId ?? get(user)?.id;
	const sync = await consumeDraftNotifySync();
	if (!sync) return { removed: 0, created: 0 };

	const drop = [...sync.rejectedIds, ...sync.acceptedIds];
	let removed = 0;
	if (id && drop.length) {
		removed = deleteInterceptDrafts(drop, id);
	}

	let created = 0;
	for (const raw of sync.offlineAccepts) {
		const payload = raw as OfflineAcceptPayload;
		const draftIds =
			Array.isArray(payload.draftIds) && payload.draftIds.length
				? payload.draftIds.filter(Boolean)
				: payload.draftId
					? [payload.draftId]
					: [];

		if (payload.kind === 'transfer' || (payload.from_account_id && payload.to_account_id)) {
			if (!payload.from_account_id || !payload.to_account_id || !payload.amount) {
				if (id && draftIds.length) deleteInterceptDrafts(draftIds, id);
				continue;
			}
			try {
				await createTransfer({
					from_account_id: payload.from_account_id,
					to_account_id: payload.to_account_id,
					amount: payload.amount,
					transaction_date: payload.transaction_date,
					description: payload.description
				});
				created += 1;
			} catch {
				continue;
			}
			if (id && draftIds.length) {
				removed += deleteInterceptDrafts(draftIds, id);
			}
			continue;
		}

		if (!payload.account_id || !payload.amount) {
			if (id && draftIds.length) deleteInterceptDrafts(draftIds, id);
			continue;
		}
		try {
			await createTransaction({
				account_id: payload.account_id,
				type: payload.type === 'income' ? 'income' : 'expense',
				amount: payload.amount,
				transaction_date: payload.transaction_date,
				category_id: payload.category_id,
				subcategory_id: payload.subcategory_id,
				merchant_id: payload.merchant_id,
				merchant_name: payload.merchant_name,
				description: payload.description
			});
			created += 1;
		} catch {
			continue;
		}
		if (id && draftIds.length) {
			for (const draftId of draftIds) {
				removed += deleteInterceptDraft(draftId, id) ? 1 : 0;
			}
		}
	}

	return { removed, created };
}
