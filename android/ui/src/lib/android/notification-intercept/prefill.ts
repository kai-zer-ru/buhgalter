import { suggestCategoryFromMerchant } from './category-suggest';
import type { InterceptDraft, TransactionCreatePrefill } from './types';

let pendingPrefill: TransactionCreatePrefill | null = null;

export function draftTxType(draft: InterceptDraft): 'expense' | 'income' {
	return draft.parsed.kind === 'income' ? 'income' : 'expense';
}

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
		type: prefill.type === 'income' ? 'income' : 'expense',
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
	const type = draftTxType(draft);
	const label = (draft.merchantName || draft.parsed.merchantText || '').trim();
	// Income without catalog merchant → comment; purchase merchant stays in merchant field.
	const useAsDescription = type === 'income' && !draft.merchantId && Boolean(label);
	return {
		amount: draft.parsed.amount,
		accountId: draft.accountId,
		merchantId: draft.merchantId,
		merchantName: draft.merchantId ? undefined : type === 'income' ? undefined : draft.merchantName,
		description: useAsDescription ? label.slice(0, 2000) : undefined,
		occurredAt: draft.parsed.occurredAt,
		type,
		draftId: draft.id
	};
}

/** Prefill + category/subcategory from prior txs of matched merchant (same type). */
export async function prefillFromDraftWithSuggestions(
	draft: InterceptDraft
): Promise<TransactionCreatePrefill> {
	const base = prefillFromDraft(draft);
	if (!draft.merchantId) return base;
	const suggest = await suggestCategoryFromMerchant(draft.merchantId, base.type ?? 'expense');
	if (!suggest) return base;
	return {
		...base,
		categoryId: suggest.categoryId,
		subcategoryId: suggest.subcategoryId
	};
}

export function interceptCreateRoute(type: 'expense' | 'income' = 'expense'): string {
	return `/transactions/new?type=${type}&from=/settings/bank-notifications/drafts`;
}

/** @deprecated use interceptCreateRoute('expense') */
export const INTERCEPT_EXPENSE_ROUTE = interceptCreateRoute('expense');
