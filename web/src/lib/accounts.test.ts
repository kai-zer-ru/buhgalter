import { describe, expect, it } from 'vitest';
import type { Account } from '$lib/api/client';
import { defaultAccountId } from './accounts';

function account(id: string, name: string, isPrimary = false): Account {
	return {
		id,
		name,
		type: 'cash',
		balance: 0,
		balance_display: '0.00',
		is_primary: isPrimary,
		is_archived: false,
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
