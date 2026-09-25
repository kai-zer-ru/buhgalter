import { get } from 'svelte/store';
import { listMerchants } from '$lib/api/client';
import { user } from '$lib/stores/auth';
import { resolveAccountId } from './account-resolve';
import {
	allKnownPackages,
	allKnownSmsSenderEntries,
	isAllowlistedPackage,
	resolveRawBankNotification
} from './banks';
import { suggestCategoryFromMerchant } from './category-suggest';
import { applyNativeDraftNotifySync, syncDraftNotifyToNative } from './draft-notify';
import { addInterceptDraft, removeDraftMatchingCancel, updateInterceptDraft } from './drafts';
import { appendLocalHistoryFromRaw } from './history-local';
import { matchMerchant } from './merchant-match';
import { parseBankNotification } from './parsers';
import {
	acknowledgeNativePending,
	consumeNativePending,
	peekNativePending,
	syncNativeCapture
} from './plugin';
import { draftTxType } from './prefill';
import { getCurrentInterceptSettings, loadInterceptSettings } from './settings';
import type { InterceptDraft, RawBankNotification } from './types';

/** Push current user's enabled flag + allowlist into native store. */
export async function syncInterceptNativeFromSettings(userId?: string | null): Promise<void> {
	const id = userId ?? get(user)?.id;
	const settings = id ? loadInterceptSettings(id) : getCurrentInterceptSettings();
	await syncNativeCapture({
		enabled: settings.enabled,
		packages: allKnownPackages(),
		smsSenders: allKnownSmsSenderEntries()
	});
	await syncDraftNotifyToNative(id);
}

export type ProcessPendingResult = {
	added: number;
	cancelled: number;
};

/**
 * Consume native pending notifications → parse → local drafts (if user enabled).
 * Cancel/refund pushes remove a matching purchase draft instead of creating one.
 */
export async function processPendingBankNotifications(): Promise<number> {
	const r = await processPendingBankNotificationsDetailed();
	return r.added;
}

export async function processPendingBankNotificationsDetailed(): Promise<ProcessPendingResult> {
	const u = get(user);
	if (!u?.id) return { added: 0, cancelled: 0 };
	const settings = loadInterceptSettings(u.id);
	if (!settings.enabled) {
		const pending = await peekNativePending();
		for (const raw of pending) {
			const resolved = resolveRawBankNotification(raw);
			appendLocalHistoryFromRaw(raw, {
				inAllowlist: isAllowlistedPackage(resolved.packageName),
				queued: false
			});
		}
		await consumeNativePending();
		await syncDraftNotifyToNative(u.id);
		return { added: 0, cancelled: 0 };
	}

	const items = await peekNativePending();
	if (!items.length) {
		await syncDraftNotifyToNative(u.id);
		return { added: 0, cancelled: 0 };
	}

	let merchants: { id: string; name: string }[];
	try {
		merchants = await listMerchants();
	} catch {
		merchants = [];
	}

	let added = 0;
	let cancelled = 0;
	const ackKeys: string[] = [];
	for (const raw of items) {
		const resolved = resolveRawBankNotification(raw);
		const key = raw.dedupeKey || `${raw.packageName}|${raw.postedAt}|${raw.title}|${raw.text}`;
		const parsed = parseBankNotification(raw);
		// Always mirror into JS history — native history prefs were empty on some OEM builds.
		appendLocalHistoryFromRaw(raw, {
			inAllowlist: isAllowlistedPackage(resolved.packageName),
			queued: Boolean(parsed)
		});
		if (!parsed) {
			ackKeys.push(key);
			continue;
		}
		if (parsed.kind === 'cancel') {
			cancelled += removeDraftMatchingCancel(parsed, u.id);
			ackKeys.push(key);
			continue;
		}
		if (await ingestRawNotification(raw, settings, merchants, u.id)) {
			added += 1;
		}
		ackKeys.push(key);
	}
	if (ackKeys.length) {
		await acknowledgeNativePending(ackKeys);
	}
	await syncDraftNotifyToNative(u.id);
	return { added, cancelled };
}

export async function ingestRawNotification(
	raw: RawBankNotification,
	settings: ReturnType<typeof loadInterceptSettings>,
	merchants: { id: string; name: string }[],
	userId: string
): Promise<boolean> {
	const parsed = parseBankNotification(raw);
	if (!parsed || parsed.kind === 'cancel') return false;
	const accountId = resolveAccountId(parsed, settings);
	const merchant = matchMerchant(parsed.merchantText, merchants);
	const draft = addInterceptDraft(
		parsed,
		{
			accountId,
			merchantId: merchant.merchantId,
			merchantName: merchant.merchantName
		},
		userId
	);
	if (!draft) return false;
	await attachCategorySuggestion(draft, userId);
	return true;
}

async function attachCategorySuggestion(draft: InterceptDraft, userId: string): Promise<void> {
	if (!draft.merchantId) return;
	const suggest = await suggestCategoryFromMerchant(draft.merchantId, draftTxType(draft));
	if (!suggest) return;
	updateInterceptDraft(
		draft.id,
		{ categoryId: suggest.categoryId, subcategoryId: suggest.subcategoryId },
		userId
	);
}

/** Pull shade Accept/Reject results, then refresh native draft mirror. */
export async function syncDraftNotifyFromNative(userId?: string | null): Promise<{
	removed: number;
	created: number;
}> {
	const id = userId ?? get(user)?.id;
	const result = await applyNativeDraftNotifySync(id);
	await syncDraftNotifyToNative(id);
	return result;
}
