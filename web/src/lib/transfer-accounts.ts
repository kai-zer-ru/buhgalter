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

/** First account id other than excludeId, or empty string if none. */
export function pickOtherAccountId(accounts: readonly Account[], excludeId: string): string {
	return accounts.find((acc) => acc.id !== excludeId)?.id ?? '';
}
