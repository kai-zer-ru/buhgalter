import { describe, expect, it, beforeEach } from 'vitest';
import {
	buildCreatePayload,
	canToggleRetroDebit,
	clearCreditCreate,
	effectivePrincipal,
	emptyCreditCreateDraft,
	hasPastSchedulePayments,
	isRetroDebited,
	toggleRetroDebit,
	validateBasics,
	type CreditCreateDraft
} from './create-draft';

const tz = 'Europe/Moscow';

function draft(partial: Partial<CreditCreateDraft> = {}): CreditCreateDraft {
	return { ...emptyCreditCreateDraft(tz), ...partial };
}

describe('credit create draft', () => {
	beforeEach(() => {
		clearCreditCreate();
	});

	it('computes mortgage principal as price minus down payment', () => {
		expect(
			effectivePrincipal(
				draft({
					productType: 'mortgage',
					propertyPrice: '5000000.00',
					downPayment: '1000000.00'
				})
			)
		).toBe('4 000 000.00');
	});

	it('rejects mortgage when down payment covers price', () => {
		const d = draft({
			productType: 'mortgage',
			propertyPrice: '1000.00',
			downPayment: '1000.00',
			issueDateLocal: '2026-01-15T00:00'
		});
		expect(validateBasics(d)).toBe('credits.error.mortgageFields');
	});

	it('builds consumer create payload', () => {
		const d = draft({
			name: 'Auto',
			productType: 'credit',
			principal: '120000.00',
			issueDateLocal: '2026-01-15T00:00',
			termMonths: '12',
			interestRate: '12',
			interval: 'month',
			debitAccountId: 'acc-1',
			createTransactions: true,
			debitTimeLocal: '08:00',
			scheduleRows: [
				{ date: '2026-02-15T00:00', amount: '10000.00' },
				{ date: '2026-03-15T00:00', amount: '10000.00' }
			]
		});
		const payload = buildCreatePayload(d, tz);
		expect(payload.credit_kind).toBe('consumer');
		expect(payload.principal_amount).toBe('120000.00');
		expect(payload.debit_account_id).toBe('acc-1');
		expect(payload.first_payment_today).toBe(false);
		expect(payload.schedule_seed).toEqual([]);
	});

	it('detects past schedule dates for retroactive toggle', () => {
		expect(
			hasPastSchedulePayments(
				draft({
					scheduleRows: [
						{ date: '2099-01-01T00:00', amount: '1.00' },
						{ date: '2099-02-01T00:00', amount: '1.00' }
					]
				}),
				tz
			)
		).toBe(false);
		expect(
			hasPastSchedulePayments(
				draft({
					scheduleRows: [
						{ date: '2020-01-01T00:00', amount: '1.00' },
						{ date: '2099-01-01T00:00', amount: '1.00' }
					]
				}),
				tz
			)
		).toBe(true);
		expect(hasPastSchedulePayments(draft({ scheduleRows: [] }), tz)).toBe(false);
	});

	it('toggles retro debit from the bottom of past rows', () => {
		let d = draft({
			retroactive: true,
			retroactiveDebitCount: 0,
			scheduleRows: [
				{ date: '2020-01-01T00:00', amount: '1.00' },
				{ date: '2020-02-01T00:00', amount: '1.00' },
				{ date: '2099-01-01T00:00', amount: '1.00' }
			]
		});
		expect(canToggleRetroDebit(d, 1, tz)).toBe(true);
		expect(canToggleRetroDebit(d, 0, tz)).toBe(false);
		d = toggleRetroDebit(d, 1, tz);
		expect(d.retroactiveDebitCount).toBe(1);
		expect(isRetroDebited(d, 1, tz)).toBe(true);
		expect(isRetroDebited(d, 0, tz)).toBe(false);
	});
});
