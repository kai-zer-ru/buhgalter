import type { AccountType } from '$lib/api/client';

type WithAccountType = { type: AccountType; is_primary?: boolean; scope?: string };

export type AccountGroupKind = 'my_funds' | 'credit_funds';
export type AccountOwnership = 'personal' | 'household';

export type AccountDisplayGroup<T> = {
	ownership: AccountOwnership;
	kind: AccountGroupKind;
	accounts: T[];
};

export function accountOwnership<T extends { scope?: string }>(acc: T): AccountOwnership {
	return acc.scope === 'household' ? 'household' : 'personal';
}

export function accountOwnershipLabelKey(ownership: AccountOwnership, groupKind?: string): string {
	if (ownership === 'household') {
		return groupKind === 'team' ? 'household.kind.team' : 'household.kind.family';
	}
	return 'household.accounts.personal';
}

export function accountGroupKind(group: readonly WithAccountType[]): AccountGroupKind {
	return group.some((a) => a.type === 'credit_card') ? 'credit_funds' : 'my_funds';
}

export function accountGroupLabelKey(kind: AccountGroupKind): string {
	return kind === 'credit_funds' ? 'accounts.group.creditFunds' : 'accounts.group.myFunds';
}

/**
 * «Мои средства»: cash before bank; within each type primary first.
 * «Кредитные средства»: credit cards (unchanged order).
 */
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
		if (a.type !== b.type) {
			return a.type === 'cash' ? -1 : 1;
		}
		const ap = a.is_primary ? 1 : 0;
		const bp = b.is_primary ? 1 : 0;
		return bp - ap;
	});
	const groups: T[][] = [];
	if (cashAndBank.length > 0) groups.push(cashAndBank);
	if (creditCards.length > 0) groups.push(creditCards);
	return groups;
}

/** Personal cash/bank then credit, then the same groups for household accounts. */
export function groupAccountsForDisplay<T extends WithAccountType>(
	accounts: readonly T[]
): AccountDisplayGroup<T>[] {
	const personal = accounts.filter((acc) => accountOwnership(acc) === 'personal');
	const household = accounts.filter((acc) => accountOwnership(acc) === 'household');
	const out: AccountDisplayGroup<T>[] = [];
	for (const group of groupAccountsByType(personal)) {
		out.push({ ownership: 'personal', kind: accountGroupKind(group), accounts: group });
	}
	for (const group of groupAccountsByType(household)) {
		out.push({ ownership: 'household', kind: accountGroupKind(group), accounts: group });
	}
	return out;
}

export type AccountOwnershipSection<T> = {
	ownership: AccountOwnership;
	groups: AccountDisplayGroup<T>[];
};

/** One heading per ownership (personal / household), with fund groups nested. */
export function groupAccountsByOwnership<T>(
	groups: readonly AccountDisplayGroup<T>[]
): AccountOwnershipSection<T>[] {
	const byOwnership = new Map<AccountOwnership, AccountDisplayGroup<T>[]>();
	for (const group of groups) {
		const list = byOwnership.get(group.ownership) ?? [];
		list.push(group);
		byOwnership.set(group.ownership, list);
	}
	const sections: AccountOwnershipSection<T>[] = [];
	for (const ownership of ['personal', 'household'] as const) {
		const list = byOwnership.get(ownership);
		if (list?.length) sections.push({ ownership, groups: list });
	}
	return sections;
}
