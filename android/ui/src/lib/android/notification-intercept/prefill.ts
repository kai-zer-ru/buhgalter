import { suggestCategoryFromMerchant } from './category-suggest';
import type { InterceptDraft, TransactionCreatePrefill } from './types';

let pendingPrefill: TransactionCreatePrefill | null = null;

export function setInterceptPrefill(prefill: TransactionCreatePrefill): void {
	pendingPrefill = {
		description: prefill.description?.trim().slice(0, 2000) || undefined,
		amount: prefill.amount,
		accountId: prefill.accountId,
		merchantId: prefill.merchantId,
		merchantName: prefill.merchantName?.trim().slice(0, 120) || undefined,
		categoryId: prefill.categoryId || undefined,
		subcategoryId: prefill.subcategoryId || undefined,
		occurredAt: prefill.occurredAt,
		draftId: prefill.draftId
	};
}

export function takeInterceptPrefill(): TransactionCreatePrefill | null {
	const v = pendingPrefill;
	pendingPrefill = null;
	return v;
}

export function resetInterceptPrefillForTests(): void {
	pendingPrefill = null;
}

export function prefillFromDraft(draft: InterceptDraft): TransactionCreatePrefill {
	// Merchant goes into the merchant field only — do not duplicate into description/comment.
	return {
		amount: draft.parsed.amount,
		accountId: draft.accountId,
		merchantId: draft.merchantId,
		merchantName: draft.merchantId ? undefined : draft.merchantName,
		occurredAt: draft.parsed.occurredAt,
		draftId: draft.id
	};
}

/** Prefill + category/subcategory from prior expenses of matched merchant. */
export async function prefillFromDraftWithSuggestions(
	draft: InterceptDraft
): Promise<TransactionCreatePrefill> {
	const base = prefillFromDraft(draft);
	if (!draft.merchantId) return base;
	const suggest = await suggestCategoryFromMerchant(draft.merchantId);
	if (!suggest) return base;
	return {
		...base,
		categoryId: suggest.categoryId,
		subcategoryId: suggest.subcategoryId
	};
}

export const INTERCEPT_EXPENSE_ROUTE =
	'/transactions/new?type=expense&from=/settings/bank-notifications/drafts';
