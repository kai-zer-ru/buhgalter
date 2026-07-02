import { describe, expect, it } from 'vitest';
import {
	autoTopupSourceOptions,
	defaultAutoTopupSourceId,
	isAutoTopupEligible,
	resolveAutoTopupSourceName,
	validateAutoTopupForm
} from './auto-topup';
import type { Account } from '$lib/api/client';

const base = (overrides: Partial<Account>): Account => ({
	id: 'a1',
	name: 'Test',
	type: 'bank',
	bank_id: null,
	initial_balance: 0,
	balance: 0,
	balance_display: '0.00',
	status: 'active',
	is_primary: false,
	created_at: '',
	updated_at: '',
	...overrides
});

describe('isAutoTopupEligible', () => {
	it('allows only active bank accounts', () => {
		expect(isAutoTopupEligible(base({ type: 'bank' }))).toBe(true);
		expect(isAutoTopupEligible(base({ type: 'cash' }))).toBe(false);
		expect(isAutoTopupEligible(base({ type: 'credit_card' }))).toBe(false);
		expect(isAutoTopupEligible(base({ status: 'archived' }))).toBe(false);
	});
});

describe('resolveAutoTopupSourceName', () => {
	it('returns source name when auto top-up is enabled', () => {
		const accounts = [base({ id: 'b1', name: 'Яндекс' }), base({ id: 'b2', name: 'Сбер' })];
		expect(
			resolveAutoTopupSourceName(
				{ auto_topup_enabled: true, auto_topup_source_account_id: 'b1' },
				accounts
			)
		).toBe('Яндекс');
	});

	it('returns null when disabled or source is missing', () => {
		const accounts = [base({ id: 'b1', name: 'Яндекс' })];
		expect(
			resolveAutoTopupSourceName(
				{ auto_topup_enabled: false, auto_topup_source_account_id: 'b1' },
				accounts
			)
		).toBeNull();
		expect(
			resolveAutoTopupSourceName(
				{ auto_topup_enabled: true, auto_topup_source_account_id: null },
				accounts
			)
		).toBeNull();
		expect(
			resolveAutoTopupSourceName(
				{ auto_topup_enabled: true, auto_topup_source_account_id: 'missing' },
				accounts
			)
		).toBeNull();
	});
});

describe('autoTopupSourceOptions', () => {
	it('excludes beneficiary and non-bank accounts', () => {
		const accounts = [
			base({ id: 'b1', name: 'Bank', type: 'bank' }),
			base({ id: 'c1', name: 'Cash', type: 'cash' }),
			base({ id: 'b2', name: 'Other', type: 'bank' })
		];
		const opts = autoTopupSourceOptions(accounts, 'b1');
		expect(opts.map((o) => o.value)).toEqual(['b2']);
	});
});

describe('defaultAutoTopupSourceId', () => {
	it('prefers primary bank account', () => {
		const accounts = [
			base({ id: 'b1', type: 'bank', is_primary: false }),
			base({ id: 'b2', type: 'bank', is_primary: true })
		];
		expect(defaultAutoTopupSourceId(accounts, 'x')).toBe('b2');
	});
});

describe('validateAutoTopupForm', () => {
	it('requires fields when enabled', () => {
		expect(validateAutoTopupForm(true, '', '5000', 'b1')).toBe('required');
	});
	it('checks threshold less than target', () => {
		expect(validateAutoTopupForm(true, '5000', '3000', 'b1')).toBe('range');
		expect(validateAutoTopupForm(true, '3000', '5000', 'b1')).toBeNull();
	});
});
