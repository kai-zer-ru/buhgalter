import { draftTxType } from './prefill';
import type { InterceptDraft, TransferCreatePrefill } from './types';

/** Own-bank transfer: debit push + credit push often arrive within this window. */
export const TRANSFER_PAIR_WINDOW_MS = 2 * 60 * 60 * 1000;

function occurredMs(draft: InterceptDraft): number {
	return Date.parse(draft.parsed.occurredAt) || Date.parse(draft.createdAt) || 0;
}

function earlierIso(a: string, b: string): string {
	return Date.parse(a) <= Date.parse(b) ? a : b;
}

/**
 * Complementary drafts from an inter-account transfer: expense + income,
 * same amount, different accounts (or different banks if accounts unknown).
 */
export function canPairDraftsAsTransfer(a: InterceptDraft, b: InterceptDraft): boolean {
	if (a.id === b.id) return false;
	if (draftTxType(a) === draftTxType(b)) return false;
	if (a.parsed.kind === 'cancel' || b.parsed.kind === 'cancel') return false;
	if (a.parsed.amount !== b.parsed.amount) return false;
	if (a.accountId && b.accountId && a.accountId === b.accountId) return false;
	if (!(a.accountId && b.accountId && a.accountId !== b.accountId)) {
		if (a.parsed.bankId === b.parsed.bankId) return false;
	}
	return Math.abs(occurredMs(a) - occurredMs(b)) <= TRANSFER_PAIR_WINDOW_MS;
}

/** Other draft that uniquely pairs with this one, or null if none / several. */
export function findUniqueTransferComplement(
	draft: InterceptDraft,
	drafts: InterceptDraft[]
): InterceptDraft | null {
	const hits = drafts.filter((other) => canPairDraftsAsTransfer(draft, other));
	return hits.length === 1 ? hits[0] : null;
}

export type InterceptTransferPair = {
	from: InterceptDraft;
	to: InterceptDraft;
};

/** Greedy unique pairs (each side has exactly one complement among unused). */
export function listUniqueTransferPairs(drafts: InterceptDraft[]): InterceptTransferPair[] {
	const used = new Set<string>();
	const pairs: InterceptTransferPair[] = [];
	for (const draft of drafts) {
		if (used.has(draft.id)) continue;
		const unused = drafts.filter((d) => !used.has(d.id));
		const complement = findUniqueTransferComplement(draft, unused);
		if (!complement) continue;
		const reverse = findUniqueTransferComplement(complement, unused);
		if (reverse?.id !== draft.id) continue;
		const from = draftTxType(draft) === 'expense' ? draft : complement;
		const to = draftTxType(draft) === 'income' ? draft : complement;
		pairs.push({ from, to });
		used.add(from.id);
		used.add(to.id);
	}
	return pairs;
}

export function pairedDraftIds(pairs: InterceptTransferPair[]): Set<string> {
	const ids = new Set<string>();
	for (const pair of pairs) {
		ids.add(pair.from.id);
		ids.add(pair.to.id);
	}
	return ids;
}

/** Drafts that are not already shown as a unique transfer pair. */
export function unpairedInterceptDrafts(drafts: InterceptDraft[]): InterceptDraft[] {
	const paired = pairedDraftIds(listUniqueTransferPairs(drafts));
	return drafts.filter((d) => !paired.has(d.id));
}

/** Cards the user sees: each unique pair counts as one, plus unpaired drafts. */
export function countVisibleInterceptItems(drafts: InterceptDraft[]): number {
	return listUniqueTransferPairs(drafts).length + unpairedInterceptDrafts(drafts).length;
}

export function resolveTransferDraftSides(
	a: InterceptDraft,
	b?: InterceptDraft | null
): { from?: InterceptDraft; to?: InterceptDraft } | null {
	if (!b) {
		return draftTxType(a) === 'income' ? { to: a } : { from: a };
	}
	if (!canPairDraftsAsTransfer(a, b)) return null;
	return draftTxType(a) === 'expense' ? { from: a, to: b } : { from: b, to: a };
}

export function transferPrefillFromDrafts(
	a: InterceptDraft,
	b?: InterceptDraft | null
): TransferCreatePrefill | null {
	const sides = resolveTransferDraftSides(a, b);
	if (!sides) return null;
	const amountDraft = sides.from ?? sides.to;
	if (!amountDraft) return null;
	const occurredAt =
		sides.from && sides.to
			? earlierIso(sides.from.parsed.occurredAt, sides.to.parsed.occurredAt)
			: amountDraft.parsed.occurredAt;
	const draftIds = [sides.from?.id, sides.to?.id].filter((id): id is string => Boolean(id));
	return {
		fromAccountId: sides.from?.accountId,
		toAccountId: sides.to?.accountId,
		amount: amountDraft.parsed.amount,
		occurredAt,
		draftIds
	};
}

/** One draft (one side) or two complementary drafts. */
export function transferPrefillFromSelection(
	selected: InterceptDraft[]
): TransferCreatePrefill | null {
	if (selected.length === 1) return transferPrefillFromDrafts(selected[0]);
	if (selected.length === 2) return transferPrefillFromDrafts(selected[0], selected[1]);
	return null;
}
