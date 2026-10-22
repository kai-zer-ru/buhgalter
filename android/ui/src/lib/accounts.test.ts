import { describe, expect, it } from 'vitest';
import type { Account } from '$lib/api/client';
import { canSetAsPrimary, defaultAccountId, sortAccountsForSelect } from './accounts';

function account(
	id: string,
	name: string,
	isPrimary = false,
	type: Account['type'] = 'cash'
): Account {
	return {
		id,
		name,
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

describe('defaultAccountId', () => {
	const accounts = [account('cash', 'Наличные'), account('bank', 'Яндекс', true)];

	it('returns explicit account when provided', () => {
		expect(defaultAccountId(accounts, 'cash')).toBe('cash');
	});

	it('falls back to primary when explicit id is empty', () => {
		expect(defaultAccountId(accounts, '')).toBe('bank');
	});

	it('falls back to first account when no primary', () => {
		const list = [account('a', 'A'), account('b', 'B')];
		expect(defaultAccountId(list, '')).toBe('a');
	});
});

describe('sortAccountsForSelect', () => {
	it('matches home/accounts order: cash, bank (primary first within type), then credit cards', () => {
		const list = [
			account('bank-other', 'Другой банк', false, 'bank'),
			account('card', 'Карта', false, 'credit_card'),
			account('cash', 'Наличные', false, 'cash'),
			account('bank', 'Яндекс', true, 'bank')
		];
		expect(sortAccountsForSelect(list).map((a) => a.id)).toEqual([
			'cash',
			'bank',
			'bank-other',
			'card'
		]);
	});
});

describe('canSetAsPrimary', () => {
	it('allows active cash/bank that is not already primary', () => {
		expect(canSetAsPrimary(account('a', 'Cash'))).toBe(true);
		expect(canSetAsPrimary(account('b', 'Bank', true))).toBe(false);
		expect(canSetAsPrimary({ ...account('c', 'Card'), type: 'credit_card' })).toBe(false);
	});
});
