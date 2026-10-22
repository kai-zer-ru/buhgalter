import type { Account } from '$lib/api/client';
import { sortAccountsForSelect } from '$lib/accounts';
import { accountSelectOption, type SelectOption } from '$lib/select-options';

export type AccountSelectOption = SelectOption;

/** Account options for transfer selects; excludes the account chosen in the opposite field. */
export function transferAccountOptions(
	accounts: readonly Account[],
	excludeId: string
): AccountSelectOption[] {
	return sortAccountsForSelect(accounts)
		.filter((acc) => acc.id !== excludeId)
		.map((acc) => accountSelectOption(acc));
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
