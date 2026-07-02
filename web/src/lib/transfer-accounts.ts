import type { Account } from '$lib/api/client';

export type AccountSelectOption = { value: string; label: string };

/** Account options for transfer selects; excludes the account chosen in the opposite field. */
export function transferAccountOptions(
	accounts: readonly Account[],
	excludeId: string
): AccountSelectOption[] {
	return accounts
		.filter((acc) => acc.id !== excludeId)
		.map((acc) => ({ value: acc.id, label: acc.name }));
}

/** Cash/bank with positive balance must transfer funds before archive or delete. */
export function needsBalanceTransfer(acc: Account): boolean {
	return acc.type !== 'credit_card' && acc.balance > 0;
}

/** Default target for balance transfer: primary account, else first in the list. */
export function defaultTransferAccountId(accounts: readonly Account[], excludeId: string): string {
	const options = transferAccountOptions(accounts, excludeId);
	if (options.length === 0) return '';
	const primary = accounts.find((acc) => acc.is_primary && acc.id !== excludeId);
	if (primary) return primary.id;
	return options[0].value;
}

/** @deprecated Use defaultTransferAccountId */
export function pickOtherAccountId(accounts: readonly Account[], excludeId: string): string {
	return defaultTransferAccountId(accounts, excludeId);
}
