import type { Account } from '$lib/api/client';
import { groupAccountsByType } from '$lib/accounts/group-by-type';
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
}): boolean {
	if (acc.is_primary) return false;
	if (acc.status != null && acc.status !== 'active') return false;
	return acc.type !== 'credit_card';
}

/**
 * Select lists: same order as home /accounts —
 * cash → bank (primary first within type), then credit cards.
 */
export function sortAccountsForSelect<T extends { type: Account['type']; is_primary?: boolean }>(
	accounts: readonly T[]
): T[] {
	return groupAccountsByType(accounts).flat();
}
