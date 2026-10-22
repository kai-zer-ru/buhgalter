import { describe, expect, it } from 'vitest';
import { formatStatsPeriod } from './stats-period';

describe('formatStatsPeriod', () => {
	it('keeps day grouping as ISO date', () => {
		expect(formatStatsPeriod('2026-12-31', 'day', 'ru')).toBe('31.12.2026');
	});

	it('formats month grouping in Russian', () => {
		expect(formatStatsPeriod('2026-06-01', 'month', 'ru')).toBe('Июнь 2026');
	});

	it('formats month grouping in English', () => {
		expect(formatStatsPeriod('2026-06-01', 'month', 'en')).toBe('June 2026');
	});

	it('formats week within one month in Russian', () => {
		expect(formatStatsPeriod('2026-06-01', 'week', 'ru')).toBe('1-7 июня 2026');
	});

	it('formats week within one month in English', () => {
		expect(formatStatsPeriod('2026-06-01', 'week', 'en')).toBe('June 1-7, 2026');
	});

	it('formats week spanning months in Russian', () => {
		expect(formatStatsPeriod('2026-06-29', 'week', 'ru')).toBe('29 июня – 5 июля 2026');
	});

	it('formats week spanning months in English', () => {
		expect(formatStatsPeriod('2026-06-29', 'week', 'en')).toBe('June 29 – July 5, 2026');
	});
});
