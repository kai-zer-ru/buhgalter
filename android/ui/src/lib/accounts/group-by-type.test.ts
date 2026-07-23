import { describe, expect, it } from 'vitest';
import type { Account } from '$lib/api/client';
import { accountGroupKind, accountGroupLabelKey, groupAccountsByType } from './group-by-type';

function account(id: string, type: Account['type'], isPrimary = false): Account {
	return {
		id,
		name: id,
		type,
		bank_id: null,
		initial_balance: 0,
		balance: 0,
		balance_display: '0.00',
		status: 'active',
		is_primary: isPrimary,
		created_at: '2026-01-01T00:00:00Z',
		updated_at: '2026-01-01T00:00:00Z'
	};
}

describe('groupAccountsByType', () => {
	it('groups cash, bank, and credit cards in order', () => {
		const groups = groupAccountsByType([
			account('bank-2', 'bank'),
			account('card-1', 'credit_card'),
			account('cash-1', 'cash'),
			account('bank-1', 'bank')
		]);

		expect(groups.map((g) => g.map((a) => a.id))).toEqual([
			['cash-1', 'bank-2', 'bank-1'],
			['card-1']
		]);
	});

	it('puts primary bank after all cash accounts', () => {
		const groups = groupAccountsByType([
			account('bank-other', 'bank'),
			account('cash-1', 'cash'),
			account('bank-primary', 'bank', true),
			account('bank-2', 'bank')
		]);

		expect(groups[0]?.map((a) => a.id)).toEqual(['cash-1', 'bank-primary', 'bank-other', 'bank-2']);
	});

	it('puts primary cash first', () => {
		const groups = groupAccountsByType([
			account('cash-2', 'cash'),
			account('bank-1', 'bank'),
			account('cash-primary', 'cash', true)
		]);

		expect(groups[0]?.map((a) => a.id)).toEqual(['cash-primary', 'cash-2', 'bank-1']);
	});

	it('omits empty groups', () => {
		const groups = groupAccountsByType([account('cash-1', 'cash'), account('bank-1', 'bank')]);

		expect(groups.map((g) => g.map((a) => a.type))).toEqual([['cash', 'bank']]);
	});
});

describe('accountGroupKind', () => {
	it('detects my funds and credit funds groups', () => {
		const groups = groupAccountsByType([
			account('cash-1', 'cash'),
			account('card-1', 'credit_card')
		]);

		expect(accountGroupKind(groups[0])).toBe('my_funds');
		expect(accountGroupKind(groups[1])).toBe('credit_funds');
	});

	it('maps kinds to i18n keys', () => {
		expect(accountGroupLabelKey('my_funds')).toBe('accounts.group.myFunds');
		expect(accountGroupLabelKey('credit_funds')).toBe('accounts.group.creditFunds');
	});
});
