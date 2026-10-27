import { beforeEach, describe, expect, it } from 'vitest';
import { formatBalance, formatTransferRate, setCurrencyCatalog } from './finance';

beforeEach(() => {
	setCurrencyCatalog([
		{ code: 'RUB', symbol: '₽' },
		{ code: 'USD', symbol: '$' }
	]);
});

describe('formatBalance', () => {
	it('uses the catalog symbol for RUB', () => {
		expect(formatBalance('1234.56', 'RUB')).toBe('1 234.56 ₽');
	});

	it('uses the catalog symbol for USD', () => {
		expect(formatBalance('10.00', 'USD')).toBe('10.00 $');
	});

	it('prints an unknown code as-is', () => {
		expect(formatBalance('5.00', 'XYZ')).toBe('5.00 XYZ');
	});

	it('prints the code before the catalog is loaded', () => {
		setCurrencyCatalog([]);
		expect(formatBalance('5.00', 'RUB')).toBe('5.00 RUB');
	});
});

describe('formatTransferRate', () => {
	it('divides to-cents by from-cents with 6 decimal places', () => {
		expect(formatTransferRate(900_000, 10_000)).toBe('0.011111');
	});

	it('returns null when either side is not positive', () => {
		expect(formatTransferRate(0, 100)).toBeNull();
		expect(formatTransferRate(100, 0)).toBeNull();
	});
});
