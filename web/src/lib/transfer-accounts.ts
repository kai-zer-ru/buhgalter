import type { Account } from '$lib/api/client';
import { accountSelectOptions, type SelectOption } from '$lib/select-options';

export type AccountSelectOption = SelectOption;

/** Account options for transfer selects; excludes the account chosen in the opposite field. */
export function transferAccountOptions(
	accounts: readonly Account[],
	excludeId: string
): AccountSelectOption[] {
	return accountSelectOptions(accounts.filter((acc) => acc.id !== excludeId));
}

/** Cash/bank with positive balance must transfer funds before archive or delete. */
export function needsBalanceTransfer(acc: Account): boolean {
	return acc.type !== 'credit_card' && acc.balance > 0;
}

/** Default target for balance transfer: primary account, else first in the list. */
export function defaultTransferAccountId(accounts: readonly Account[], excludeId: string): string {
	const options = transferAccountOptions(accounts, excludeId);
	const first = options.find((option) => !option.header && !option.disabled);
	if (!first) return '';
	const primary = accounts.find((acc) => acc.is_primary && acc.id !== excludeId);
	if (primary) return primary.id;
	return first.value;
}

/** @deprecated Use defaultTransferAccountId */
export function pickOtherAccountId(accounts: readonly Account[], excludeId: string): string {
	return defaultTransferAccountId(accounts, excludeId);
}
