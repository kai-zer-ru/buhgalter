import { describe, expect, it } from 'vitest';
import {
	evaluateMoneyExpression,
	formatMoneyForInput,
	formatMoneyInput,
	formatMoneyLive,
	mapMoneyInputCursor,
	toCents
} from './money';

describe('formatMoneyForInput', () => {
	it('returns empty for unset and zero values', () => {
		expect(formatMoneyForInput('')).toBe('');
		expect(formatMoneyForInput('0')).toBe('');
		expect(formatMoneyForInput('0.00')).toBe('');
		expect(formatMoneyForInput('0,00')).toBe('');
	});

	it('formats non-zero amounts for input', () => {
		expect(formatMoneyForInput('1000.00')).toBe('1 000.00');
		expect(formatMoneyForInput('50')).toBe('50');
	});
});

describe('formatMoneyInput', () => {
	it('clears zero on blur', () => {
		expect(formatMoneyInput('0')).toBe('');
		expect(formatMoneyInput('0.00')).toBe('');
	});

	it('normalizes non-zero amounts', () => {
		expect(formatMoneyInput('1000')).toBe('1 000.00');
	});

	it('evaluates addition and subtraction on blur', () => {
		expect(formatMoneyInput('7899+500')).toBe('8 399.00');
		expect(formatMoneyInput('100+200-50')).toBe('250.00');
		expect(formatMoneyInput('7 899.50+100')).toBe('7 999.50');
		expect(formatMoneyInput('-100+50')).toBe('-50.00');
	});

	it('keeps incomplete expression without inventing a result', () => {
		expect(formatMoneyInput('7899+')).toBe('7 899+');
		expect(formatMoneyInput('100+-')).toBe('100+-');
	});
});

describe('evaluateMoneyExpression / toCents', () => {
	it('parses +/− chains in kopecks', () => {
		expect(toCents('7899+500')).toBe(839900);
		expect(toCents('100+200-50')).toBe(25000);
		expect(evaluateMoneyExpression('7899+500')).toBe(839900);
	});

	it('returns null for incomplete expressions', () => {
		expect(evaluateMoneyExpression('7899+')).toBeNull();
		expect(evaluateMoneyExpression('100+-')).toBeNull();
		expect(() => toCents('7899+')).toThrow();
	});
});

describe('formatMoneyLive', () => {
	it('preserves +/− operators while typing', () => {
		expect(formatMoneyLive('7899+')).toBe('7 899+');
		expect(formatMoneyLive('7899+500')).toBe('7 899+500');
		expect(formatMoneyLive('100+200-50')).toBe('100+200-50');
	});
});

describe('mapMoneyInputCursor', () => {
	it('does not jump to end when editing grouped integer part', () => {
		const raw = '4 000';
		const formatted = formatMoneyLive(raw);
		expect(formatted).toBe('4 000');
		const cursor = mapMoneyInputCursor(raw, 3, formatted);
		expect(cursor).toBe(3);
		expect(cursor).toBeLessThan(formatted.length);
	});

	it('keeps position in fractional part', () => {
		const raw = '4 000.05';
		const formatted = formatMoneyLive(raw);
		const cursor = mapMoneyInputCursor(raw, 7, formatted);
		expect(formatted.slice(0, cursor)).toBe('4 000.0');
	});

	it('handles backspace in grouped digits', () => {
		const raw = '4 00';
		const formatted = formatMoneyLive(raw);
		expect(formatted).toBe('400');
		const cursor = mapMoneyInputCursor(raw, 2, formatted);
		expect(cursor).toBe(1);
	});

	it('keeps caret after operator while typing expression', () => {
		const raw = '7899+5';
		const formatted = formatMoneyLive(raw);
		expect(formatted).toBe('7 899+5');
		const cursor = mapMoneyInputCursor(raw, raw.length, formatted);
		expect(cursor).toBe(formatted.length);
	});
});
