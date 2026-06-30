import { describe, expect, it } from 'vitest';
import { calendarCells, formatDatetimeButtonLabel } from './datetime-picker';

describe('formatDatetimeButtonLabel', () => {
	it('omits seconds from datetime values', () => {
		expect(formatDatetimeButtonLabel('2026-12-31T08:30')).toBe('31.12.2026 08:30');
	});
});

describe('calendarCells', () => {
	it('pads June 2026 with July 1–4 instead of 31–34', () => {
		const cells = calendarCells(2026, 6);
		const trailing = cells.filter((c) => !c.inMonth).map((c) => c.day);
		expect(trailing).toEqual([1, 2, 3, 4, 5]);
		expect(cells).toHaveLength(35);
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
		expect(trailing.map((c) => c.day)).toEqual([1, 2, 3, 4, 5]);
		expect(trailing.every((c) => c.year === 2026)).toBe(true);
	});
});
