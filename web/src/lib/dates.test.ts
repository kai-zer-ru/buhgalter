import { describe, expect, it } from 'vitest';
import {
	DISPLAY_DATE_FORMAT,
	DISPLAY_DATETIME_FORMAT,
	DISPLAY_DATETIME_SHORT_FORMAT,
	formatAPIDateForDisplay,
	formatAPIDateTimeForDisplay,
	formatAPIOperationDateTimeForDisplay,
	formatCreditPaymentDateForDisplay,
	formatDisplayDate,
	formatDisplayDateTime,
	formatDisplayDateTimeShort,
	formatISODateForDisplay
} from './dates';

describe('display format constants', () => {
	it('documents the three user-facing formats', () => {
		expect(DISPLAY_DATE_FORMAT).toBe('dd.MM.yyyy');
		expect(DISPLAY_DATETIME_FORMAT).toBe('dd.MM.yyyy HH:mm:ss');
		expect(DISPLAY_DATETIME_SHORT_FORMAT).toBe('dd.MM.yyyy HH:mm');
	});
});

describe('formatDisplayDate', () => {
	it('formats as dd.MM.yyyy', () => {
		expect(formatDisplayDate(new Date(2026, 11, 31))).toBe('31.12.2026');
	});
});

describe('formatDisplayDateTime', () => {
	it('formats with seconds', () => {
		expect(formatDisplayDateTime(new Date(2026, 11, 31, 12, 0, 5))).toBe('31.12.2026 12:00:05');
	});
});

describe('formatDisplayDateTimeShort', () => {
	it('formats without seconds', () => {
		expect(formatDisplayDateTimeShort(new Date(2026, 11, 31, 12, 0, 5))).toBe('31.12.2026 12:00');
	});
});

describe('formatISODateForDisplay', () => {
	it('converts yyyy-MM-dd to dd.MM.yyyy', () => {
		expect(formatISODateForDisplay('2026-12-31')).toBe('31.12.2026');
	});
});

describe('formatAPIDateForDisplay', () => {
	it('formats API date in user timezone', () => {
		expect(formatAPIDateForDisplay('2026-12-31 00:00:00', 'UTC')).toBe('31.12.2026');
	});
});

describe('formatAPIDateTimeForDisplay', () => {
	it('formats API datetime in user timezone', () => {
		expect(formatAPIDateTimeForDisplay('2026-12-31 08:30:00', 'UTC')).toBe('31.12.2026 08:30:00');
	});

	it('formats RFC3339 build timestamps', () => {
		expect(formatAPIDateTimeForDisplay('2026-12-31T04:22:46Z', 'UTC')).toBe('31.12.2026 04:22:46');
	});
});

describe('formatAPIOperationDateTimeForDisplay', () => {
	it('formats without seconds in user timezone', () => {
		expect(formatAPIOperationDateTimeForDisplay('2026-12-31 08:30:45', 'UTC')).toBe(
			'31.12.2026 08:30'
		);
	});
});

describe('formatCreditPaymentDateForDisplay', () => {
	it('combines midnight payment date with debit_time_local', () => {
		expect(formatCreditPaymentDateForDisplay('2026-12-31 00:00:00', 'UTC', '11:19')).toBe(
			'31.12.2026 11:19'
		);
	});

	it('keeps explicit payment time when not midnight', () => {
		expect(formatCreditPaymentDateForDisplay('2026-12-31 08:30:00', 'UTC', '11:19')).toBe(
			'31.12.2026 08:30'
		);
	});

	it('falls back to default formatting without debit time', () => {
		expect(formatCreditPaymentDateForDisplay('2026-12-31 00:00:00', 'UTC', null)).toBe(
			'31.12.2026 00:00'
		);
	});
});
