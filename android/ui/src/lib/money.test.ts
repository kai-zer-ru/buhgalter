import { describe, expect, it } from 'vitest';
import {
	formatMoneyForInput,
	formatMoneyInput,
	formatMoneyLive,
	mapMoneyInputCursor
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
});
