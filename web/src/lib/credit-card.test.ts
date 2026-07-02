import { describe, expect, it } from 'vitest';
import {
	debitAccounts,
	isCreditCardFullyPaid,
	maxCreditCardPaymentKopecks,
	resolvePaymentAccountId,
	creditCardExpenseWarning,
	type Account
} from './credit-card';

function acc(partial: Partial<Account> & Pick<Account, 'id' | 'type'>): Account {
	return {
		id: partial.id,
		name: partial.name ?? partial.id,
		type: partial.type,
		bank_id: null,
		initial_balance: 0,
		balance: partial.balance ?? 0,
		balance_display: '0.00',
		status: partial.status ?? 'active',
		is_primary: partial.is_primary ?? false,
		created_at: '',
		updated_at: '',
		...partial
	};
}

describe('credit-card helpers', () => {
	it('resolvePaymentAccountId prefers linked debit account', () => {
		const card = acc({
			id: 'cc',
			type: 'credit_card',
			payment_account_id: 'debit-2'
		});
		const accounts = [
			acc({ id: 'debit-1', type: 'cash', is_primary: true }),
			acc({ id: 'debit-2', type: 'bank' }),
			card
		];
		expect(resolvePaymentAccountId(card, accounts)).toBe('debit-2');
	});

	it('resolvePaymentAccountId falls back to primary non-credit account', () => {
		const card = acc({ id: 'cc', type: 'credit_card' });
		const accounts = [
			acc({ id: 'cc-primary', type: 'credit_card', is_primary: true }),
			acc({ id: 'debit', type: 'bank' }),
			card
		];
		expect(resolvePaymentAccountId(card, accounts)).toBe('debit');
	});

	it('maxCreditCardPaymentKopecks uses limit minus balance', () => {
		const card = acc({
			id: 'cc',
			type: 'credit_card',
			balance: 1_500_000,
			credit_limit: 6_500_000
		});
		expect(maxCreditCardPaymentKopecks(card)).toBe(5_000_000);
	});

	it('debitAccounts excludes credit cards', () => {
		const accounts = [acc({ id: 'cash', type: 'cash' }), acc({ id: 'cc', type: 'credit_card' })];
		expect(debitAccounts(accounts).map((a) => a.id)).toEqual(['cash']);
	});

	it('creditCardExpenseWarning when expense exceeds balance', () => {
		expect(creditCardExpenseWarning(1_000, 1_500)).toBe(true);
		expect(creditCardExpenseWarning(1_000, 500)).toBe(false);
	});

	it('isCreditCardFullyPaid when balance reaches limit', () => {
		const card = acc({
			id: 'cc',
			type: 'credit_card',
			balance: 6_500_000,
			credit_limit: 6_500_000
		});
		expect(isCreditCardFullyPaid(card)).toBe(true);
		expect(isCreditCardFullyPaid({ ...card, balance: 1_000_000 })).toBe(false);
	});
});
