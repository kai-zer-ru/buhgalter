import { resolveAccountId } from './account-resolve';
import { findUniqueTransferComplement, transferPrefillFromDrafts } from './draft-transfer';
import { matchMerchant } from './merchant-match';
import { parseBankNotification } from './parsers';
import { prefillFromDraftWithSuggestions } from './prefill';
import type {
	InterceptDraft,
	InterceptSettings,
	NotificationHistoryItem,
	TransactionCreatePrefill,
	TransferCreatePrefill
} from './types';

type MerchantRef = { id: string; name: string };

export const HISTORY_CREATE_FROM = '/settings/bank-notifications/history';

export function historyRowKey(row: NotificationHistoryItem): string {
	return row.dedupeKey || `${row.packageName}|${row.postedAt}|${row.title}|${row.text}`;
}

/** True when the row would become a purchase/income draft (not cancel / unparsed). */
export function canCreateFromHistory(row: NotificationHistoryItem): boolean {
	if (!row.inAllowlist) return false;
	const parsed = parseBankNotification(row);
	return Boolean(parsed && parsed.kind !== 'cancel');
}

/** Build a transient draft-like object for prefill / transfer pairing (not stored). */
export function draftFromHistoryItem(
	row: NotificationHistoryItem,
	settings: InterceptSettings,
	merchants: MerchantRef[]
): InterceptDraft | null {
	if (!row.inAllowlist) return null;
	const parsed = parseBankNotification(row);
	if (!parsed || parsed.kind === 'cancel') return null;
	const merchant = matchMerchant(parsed.merchantText, merchants);
	return {
		id: `hist:${historyRowKey(row)}`,
		createdAt: row.postedAt ? new Date(row.postedAt).toISOString() : new Date().toISOString(),
		parsed,
		accountId: resolveAccountId(parsed, settings),
		merchantId: merchant.merchantId,
		merchantName: merchant.merchantName
	};
}

export function draftsFromHistory(
	items: NotificationHistoryItem[],
	settings: InterceptSettings,
	merchants: MerchantRef[]
): InterceptDraft[] {
	const out: InterceptDraft[] = [];
	for (const row of items) {
		const draft = draftFromHistoryItem(row, settings, merchants);
		if (draft) out.push(draft);
	}
	return out;
}

export function historyTransferComplement(
	row: NotificationHistoryItem,
	items: NotificationHistoryItem[],
	settings: InterceptSettings,
	merchants: MerchantRef[]
): InterceptDraft | null {
	const draft = draftFromHistoryItem(row, settings, merchants);
	if (!draft) return null;
	return findUniqueTransferComplement(draft, draftsFromHistory(items, settings, merchants));
}

/** Expense/income form prefill; no draftId (history recreate is not a draft). */
export async function prefillFromHistoryItem(
	row: NotificationHistoryItem,
	settings: InterceptSettings,
	merchants: MerchantRef[]
): Promise<TransactionCreatePrefill | null> {
	const draft = draftFromHistoryItem(row, settings, merchants);
	if (!draft) return null;
	const prefill = await prefillFromDraftWithSuggestions(draft);
	return {
		description: prefill.description,
		amount: prefill.amount,
		accountId: prefill.accountId,
		merchantId: prefill.merchantId,
		merchantName: prefill.merchantName,
		categoryId: prefill.categoryId,
		subcategoryId: prefill.subcategoryId,
		occurredAt: prefill.occurredAt,
		type: prefill.type
	};
}

/** Transfer form prefill only when this row uniquely pairs with another history row. */
export function transferPrefillFromHistory(
	row: NotificationHistoryItem,
	items: NotificationHistoryItem[],
	settings: InterceptSettings,
	merchants: MerchantRef[]
): TransferCreatePrefill | null {
	const draft = draftFromHistoryItem(row, settings, merchants);
	if (!draft) return null;
	const complement = findUniqueTransferComplement(
		draft,
		draftsFromHistory(items, settings, merchants)
	);
	if (!complement) return null;
	const prefill = transferPrefillFromDrafts(draft, complement);
	if (!prefill) return null;
	return {
		fromAccountId: prefill.fromAccountId,
		toAccountId: prefill.toAccountId,
		amount: prefill.amount,
		description: prefill.description,
		occurredAt: prefill.occurredAt
	};
}
