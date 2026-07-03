import { describe, expect, it } from 'vitest';
import type { Account } from '$lib/api/client';
import { groupAccountsByType } from './group-by-type';

function account(id: string, type: Account['type']): Account {
	return {
		id,
		name: id,
		type,
		bank_id: null,
		initial_balance: 0,
		balance: 0,
		balance_display: '0.00',
		status: 'active',
		is_primary: false,
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
			['cash-1'],
			['bank-2', 'bank-1'],
			['card-1']
		]);
	});

	it('omits empty groups', () => {
		const groups = groupAccountsByType([account('cash-1', 'cash'), account('bank-1', 'bank')]);

		expect(groups.map((g) => g.map((a) => a.type))).toEqual([['cash'], ['bank']]);
	});
});
