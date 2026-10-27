import type { Account } from '$lib/api/client';
import { groupAccountsForDisplay } from '$lib/accounts/group-by-type';
import { formatMoneyForInput, fromCents } from '$lib/money';

/** Stored initial balance (kopecks) → value for the account edit form. */
export function formatAccountInitialBalanceForEdit(initialBalanceKopecks: number): string {
	return formatMoneyForInput(fromCents(initialBalanceKopecks));
}

/** Default active account: primary, else first in list. */
export function defaultAccountId(accounts: readonly Account[], explicitId = ''): string {
	if (explicitId) return explicitId;
	return accounts.find((a) => a.is_primary)?.id ?? accounts[0]?.id ?? '';
}

/** Cash/bank only — credit cards cannot be the default account. */
export function canSetAsPrimary(acc: {
	type: Account['type'];
	status?: Account['status'];
	is_primary: boolean;
	scope?: string;
}): boolean {
	if (acc.is_primary) return false;
	if (acc.status != null && acc.status !== 'active') return false;
	if (acc.scope === 'household') return false;
	return acc.type !== 'credit_card';
}

/**
 * Select lists: same order as home /accounts —
 * personal then household; cash → bank (primary first within type), then credit cards.
 */
export function sortAccountsForSelect<
	T extends { type: Account['type']; is_primary?: boolean; scope?: string }
>(accounts: readonly T[]): T[] {
	return groupAccountsForDisplay(accounts).flatMap((group) => group.accounts);
}
