import { describe, expect, it } from 'vitest';
import { formatMoneyLive, mapMoneyInputCursor } from './money';

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
