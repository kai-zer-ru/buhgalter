import { beforeEach, describe, expect, it } from 'vitest';
import { setCurrencyCatalog } from './finance';
import { formatMoneyForDisplay } from './money-display';

beforeEach(() => {
	setCurrencyCatalog([{ code: 'RUB', symbol: '₽' }]);
});

describe('formatMoneyForDisplay', () => {
	it('formats API display string with thousands separator', () => {
		expect(formatMoneyForDisplay({ value: '10000.00' })).toBe('10 000.00');
	});

	it('formats cents without currency', () => {
		expect(formatMoneyForDisplay({ cents: 1_234_567 })).toBe('12 345.67');
	});

	it('formats with currency symbol', () => {
		expect(formatMoneyForDisplay({ value: '1234.56', currency: 'RUB' })).toBe('1 234.56 ₽');
	});

	it('uses the catalog symbol for USD', () => {
		setCurrencyCatalog([
			{ code: 'RUB', symbol: '₽' },
			{ code: 'USD', symbol: '$' }
		]);
		expect(formatMoneyForDisplay({ value: '10.00', currency: 'USD' })).toBe('10.00 $');
		expect(formatMoneyForDisplay({ cents: 500, currency: 'XYZ' })).toBe('5.00 XYZ');
	});

	it('formats cents with currency', () => {
		expect(formatMoneyForDisplay({ cents: 50_000, currency: 'RUB' })).toBe('500.00 ₽');
	});

	it('returns empty for missing value', () => {
		expect(formatMoneyForDisplay({})).toBe('');
	});
});
