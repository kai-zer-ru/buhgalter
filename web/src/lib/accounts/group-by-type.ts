import type { AccountType } from '$lib/api/client';

type WithAccountType = { type: AccountType };

export type AccountGroupKind = 'my_funds' | 'credit_funds';

export function accountGroupKind(group: readonly WithAccountType[]): AccountGroupKind {
	return group.some((a) => a.type === 'credit_card') ? 'credit_funds' : 'my_funds';
}

export function accountGroupLabelKey(kind: AccountGroupKind): string {
	return kind === 'credit_funds' ? 'accounts.group.creditFunds' : 'accounts.group.myFunds';
}

export function groupAccountsByType<T extends WithAccountType>(accounts: readonly T[]): T[][] {
	const cashAndBank: T[] = [];
	const creditCards: T[] = [];
	for (const acc of accounts) {
		if (acc.type === 'credit_card') {
			creditCards.push(acc);
		} else {
			cashAndBank.push(acc);
		}
	}
	cashAndBank.sort((a, b) => {
		if (a.type === b.type) return 0;
		return a.type === 'cash' ? -1 : 1;
	});
	const groups: T[][] = [];
	if (cashAndBank.length > 0) groups.push(cashAndBank);
	if (creditCards.length > 0) groups.push(creditCards);
	return groups;
}
