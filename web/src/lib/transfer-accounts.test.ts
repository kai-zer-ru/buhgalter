import { describe, expect, it } from 'vitest';
import type { Account } from '$lib/api/client';
import {
	defaultTransferAccountId,
	pickOtherAccountId,
	transferAccountOptions
} from './transfer-accounts';

function account(id: string, name: string, isPrimary = false): Account {
	return {
		id,
		name,
		type: 'cash',
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

describe('transferAccountOptions', () => {
	it('excludes the selected account from the opposite select', () => {
		const accounts = [account('a', 'Yandex'), account('b', 'Cash'), account('c', 'Bank')];
		expect(transferAccountOptions(accounts, 'a')).toEqual([
			{ value: 'b', label: 'Cash' },
			{ value: 'c', label: 'Bank' }
		]);
		expect(transferAccountOptions(accounts, 'b')).toEqual([
			{ value: 'a', label: 'Yandex' },
			{ value: 'c', label: 'Bank' }
		]);
	});

	it('returns all accounts when exclude id is empty', () => {
		const accounts = [account('a', 'A'), account('b', 'B')];
		expect(transferAccountOptions(accounts, '')).toEqual([
			{ value: 'a', label: 'A' },
			{ value: 'b', label: 'B' }
		]);
	});
});

describe('defaultTransferAccountId', () => {
	it('prefers primary account when not excluded', () => {
		const accounts = [account('a', 'A'), account('b', 'B', true), account('c', 'C')];
		expect(defaultTransferAccountId(accounts, 'a')).toBe('b');
	});

	it('falls back to first option when primary is excluded', () => {
		const accounts = [account('a', 'A', true), account('b', 'B')];
		expect(defaultTransferAccountId(accounts, 'a')).toBe('b');
	});

	it('returns empty string when no other account exists', () => {
		expect(defaultTransferAccountId([account('a', 'A')], 'a')).toBe('');
	});
});

describe('pickOtherAccountId', () => {
	it('returns another account id', () => {
		const accounts = [account('a', 'A'), account('b', 'B')];
		expect(pickOtherAccountId(accounts, 'a')).toBe('b');
		expect(pickOtherAccountId(accounts, 'b')).toBe('a');
	});

	it('returns empty string when no other account exists', () => {
		expect(pickOtherAccountId([account('a', 'A')], 'a')).toBe('');
	});
});
