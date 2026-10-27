import { describe, expect, it } from 'vitest';
import type { Account } from '$lib/api/client';
import { formatShareLinkedLines } from './share-account';

function account(id: string, name: string): Account {
	return {
		id,
		name,
		type: 'bank',
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

describe('formatShareLinkedLines', () => {
	it('formats payment and auto-topup links with names from the list', () => {
		const current = account('card', 'Карта');
		const lines = formatShareLinkedLines(
			current,
			[
				{ kind: 'payment_account', id: 'debit' },
				{ kind: 'auto_topup_source', id: 'src' }
			],
			[current, account('debit', 'Дебет'), account('src', 'Нал')],
			(key, opts) => `${key}:${opts?.values?.from ?? ''}->${opts?.values?.to ?? ''}`
		);
		expect(lines).toEqual([
			'accounts.share.linked.payment:Карта->Дебет',
			'accounts.share.linked.autoTopup:Карта->Нал'
		]);
	});
});
