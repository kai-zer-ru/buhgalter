import { describe, expect, it } from 'vitest';
import { assignIfChanged, diffNewIds } from './state-utils';

describe('assignIfChanged', () => {
	it('returns prev when deeply equal objects', () => {
		const prev = { a: 1, b: [2] };
		const next = { a: 1, b: [2] };
		expect(assignIfChanged(prev, next)).toBe(prev);
	});

	it('returns next when a field changes', () => {
		const prev = { balance: 100 };
		const next = { balance: 200 };
		expect(assignIfChanged(prev, next)).toBe(next);
	});

	it('returns prev for equal primitives', () => {
		expect(assignIfChanged(5, 5)).toBe(5);
	});

	it('handles null and empty array', () => {
		expect(assignIfChanged(null, null)).toBe(null);
		expect(assignIfChanged([], [])).toEqual([]);
	});
});

describe('diffNewIds', () => {
	it('finds new row ids', () => {
		const prev = [{ id: 'a' }, { id: 'b' }];
		const next = [{ id: 'b' }, { id: 'c' }];
		expect([...diffNewIds(prev, next)]).toEqual(['c']);
	});
});
