import { describe, expect, it } from 'vitest';
import { formatCreditPaymentDateForDisplay } from './dates';

describe('formatCreditPaymentDateForDisplay', () => {
	it('combines midnight payment date with debit_time_local', () => {
		expect(formatCreditPaymentDateForDisplay('2026-06-30 00:00:00', 'Europe/Moscow', '11:19')).toBe(
			'30.06.2026 11:19'
		);
	});

	it('keeps explicit payment time when not midnight', () => {
		expect(formatCreditPaymentDateForDisplay('2026-06-30 08:30:00', 'UTC', '11:19')).toBe(
			'30.06.2026 08:30'
		);
	});

	it('falls back to default formatting without debit time', () => {
		expect(formatCreditPaymentDateForDisplay('2026-06-30 00:00:00', 'UTC', null)).toBe(
			'30.06.2026 00:00'
		);
	});
});
