import { describe, expect, it } from 'vitest';
import {
	calendarCells,
	compareCalendarDates,
	formatDatetimeButtonLabel,
	isCalendarDayDisabled,
	nextCalendarDay
} from './datetime-picker';

describe('formatDatetimeButtonLabel', () => {
	it('omits seconds from datetime values', () => {
		expect(formatDatetimeButtonLabel('2026-12-31T08:30')).toBe('31.12.2026 08:30');
	});
});

describe('compareCalendarDates', () => {
	it('orders year, month, day', () => {
		expect(
			compareCalendarDates({ year: 2026, month: 3, day: 1 }, { year: 2026, month: 2, day: 28 })
		).toBeGreaterThan(0);
		expect(
			compareCalendarDates({ year: 2026, month: 6, day: 15 }, { year: 2026, month: 6, day: 15 })
		).toBe(0);
	});
});

describe('isCalendarDayDisabled', () => {
	const min = { year: 2026, month: 7, day: 2 };

	it('disables days before min', () => {
		expect(isCalendarDayDisabled({ year: 2026, month: 7, day: 1 }, min)).toBe(true);
		expect(isCalendarDayDisabled({ year: 2026, month: 6, day: 30 }, min)).toBe(true);
	});

	it('allows min day and later', () => {
		expect(isCalendarDayDisabled(min, min)).toBe(false);
		expect(isCalendarDayDisabled({ year: 2026, month: 7, day: 3 }, min)).toBe(false);
	});

	it('allows all days when min is null', () => {
		expect(isCalendarDayDisabled({ year: 2020, month: 1, day: 1 }, null)).toBe(false);
	});
});

describe('nextCalendarDay', () => {
	it('rolls month boundary', () => {
		expect(nextCalendarDay({ year: 2026, month: 1, day: 31 })).toEqual({
			year: 2026,
			month: 2,
			day: 1
		});
	});
});

describe('calendarCells', () => {
	it('pads June 2026 to a stable 6-week grid with July days', () => {
		const cells = calendarCells(2026, 6);
		const trailing = cells.filter((c) => !c.inMonth).map((c) => c.day);
		expect(trailing).toEqual([1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12]);
		expect(cells).toHaveLength(42);
	});

	it('includes leading days from previous month', () => {
		// May 2026 starts on Friday → Mon–Thu from April
		const cells = calendarCells(2026, 5);
		const leading = cells.slice(0, 4);
		expect(leading.map((c) => c.day)).toEqual([27, 28, 29, 30]);
		expect(leading.every((c) => c.month === 4 && c.year === 2026)).toBe(true);
		expect(cells[4]).toEqual({ day: 1, month: 5, year: 2026, inMonth: true });
	});

	it('assigns year and month to trailing days', () => {
		const cells = calendarCells(2026, 6);
		const trailing = cells.filter((c) => !c.inMonth && c.month === 7);
		expect(trailing.map((c) => c.day)).toEqual([1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12]);
		expect(trailing.every((c) => c.year === 2026)).toBe(true);
	});
});
