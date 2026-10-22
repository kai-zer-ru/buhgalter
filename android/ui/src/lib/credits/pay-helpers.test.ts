import { describe, expect, it } from 'vitest';
import { defaultPayAmount, defaultPayDate, nextPendingPayment } from './pay-helpers';
import type { Credit } from '$lib/api/client';

function credit(overrides: Partial<Credit> = {}): Credit {
	return {
		id: 'c1',
		name: 'Test',
		status: 'active',
		principal_amount: 100_000,
		remaining_amount: 50_000,
		monthly_payment: 10_000,
		next_payment_amount: 10_000,
		term_months: 12,
		is_installment: false,
		credit_kind: 'consumer',
		debit_account_id: 'a1',
		debit_account_name: 'Main',
		schedule: [],
		...overrides
	} as Credit;
}

describe('credit pay helpers', () => {
	it('finds next pending scheduled payment', () => {
		const c = credit({
			schedule: [
				{ id: 'p1', amount: 5000, is_applied: true, kind: 'scheduled' } as never,
				{ id: 'p2', amount: 7000, is_applied: false, kind: 'scheduled' } as never
			]
		});
		expect(nextPendingPayment(c)?.id).toBe('p2');
	});

	it('caps default pay amount by remaining', () => {
		const c = credit({ remaining_amount: 3000, monthly_payment: 10_000 });
		expect(defaultPayAmount(c)).toBe('30.00');
	});

	it('uses today when no pending payment date', () => {
		const date = defaultPayDate(credit(), 'Europe/Moscow');
		expect(date.length).toBeGreaterThan(0);
	});
});
