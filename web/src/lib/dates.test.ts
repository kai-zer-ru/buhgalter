import { describe, expect, it } from 'vitest';
import { formatAPIDateTimeForDisplay, formatCreditPaymentDateForDisplay } from './dates';

describe('formatAPIDateTimeForDisplay', () => {
	it('formats API datetime in user timezone', () => {
		expect(formatAPIDateTimeForDisplay('2026-06-30 08:30:00', 'UTC')).toBe('2026-06-30 08:30:00');
	});

	it('formats RFC3339 build timestamps', () => {
		expect(formatAPIDateTimeForDisplay('2026-06-30T04:22:46Z', 'UTC')).toBe('2026-06-30 04:22:46');
	});
});

describe('formatCreditPaymentDateForDisplay', () => {
	it('combines midnight payment date with debit_time_local', () => {
		expect(formatCreditPaymentDateForDisplay('2026-06-30 00:00:00', 'UTC', '11:19')).toBe(
			'2026-06-30 11:19:00'
		);
	});

	it('keeps explicit payment time when not midnight', () => {
		expect(formatCreditPaymentDateForDisplay('2026-06-30 08:30:00', 'UTC', '11:19')).toBe(
			'2026-06-30 08:30:00'
		);
	});

	it('falls back to default formatting without debit time', () => {
		expect(formatCreditPaymentDateForDisplay('2026-06-30 00:00:00', 'UTC', null)).toBe(
			'2026-06-30 00:00:00'
		);
	});
});
