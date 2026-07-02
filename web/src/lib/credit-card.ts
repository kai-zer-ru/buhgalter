import type { Account } from '$lib/api/client';

export const COMMISSION_USAGE_COMMENT = 'Комиссия за использование карты';

export function isCreditCard(acc: Pick<Account, 'type'>): boolean {
	return acc.type === 'credit_card';
}

export function debitAccounts(accounts: Account[]): Account[] {
	return accounts.filter((a) => a.status === 'active' && a.type !== 'credit_card');
}

export function resolvePaymentAccountId(card: Account, accounts: Account[]): string | undefined {
	if (card.payment_account_id) {
		const linked = accounts.find((a) => a.id === card.payment_account_id && a.status === 'active');
		if (linked && linked.type !== 'credit_card') return linked.id;
	}
	const primary = accounts.find(
		(a) => a.is_primary && a.status === 'active' && a.type !== 'credit_card'
	);
	if (primary) return primary.id;
	return debitAccounts(accounts)[0]?.id;
}

export function isCreditCardFullyPaid(
	acc: Pick<Account, 'type' | 'balance' | 'credit_limit'>
): boolean {
	if (!isCreditCard(acc)) return true;
	if (acc.credit_limit == null) return false;
	return acc.balance >= acc.credit_limit;
}

export function maxCreditCardPaymentKopecks(card: Account): number | null {
	if (!isCreditCard(card) || card.credit_limit == null) return null;
	return Math.max(0, card.credit_limit - card.balance);
}

export function creditCardExpenseWarning(balanceKopecks: number, amountKopecks: number): boolean {
	return balanceKopecks - amountKopecks < 0;
}
