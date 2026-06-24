import type { Account } from '$lib/api/client';

/** Default active account: primary, else first in list. */
export function defaultAccountId(accounts: readonly Account[], explicitId = ''): string {
	if (explicitId) return explicitId;
	return accounts.find((a) => a.is_primary)?.id ?? accounts[0]?.id ?? '';
}
