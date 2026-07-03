import type { AccountType } from '$lib/api/client';

const accountTypeOrder: AccountType[] = ['cash', 'bank', 'credit_card'];

type WithAccountType = { type: AccountType };

export function groupAccountsByType<T extends WithAccountType>(accounts: readonly T[]): T[][] {
	const byType = new Map<AccountType, T[]>();
	for (const type of accountTypeOrder) {
		byType.set(type, []);
	}
	for (const acc of accounts) {
		byType.get(acc.type)?.push(acc);
	}
	return accountTypeOrder.map((type) => byType.get(type) ?? []).filter((group) => group.length > 0);
}
