import { describe, expect, it } from 'vitest';
import {
	assignIfChanged,
	diffNewIds,
	isDashboardRefPath,
	isDashboardShape,
	stableEqual,
	stripDisplayFields
} from './state-utils';

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

	it('returns prev for equal id/updated_at arrays without JSON stringify', () => {
		const prev = [{ id: 'a', updated_at: '1' }];
		const next = [{ id: 'a', updated_at: '1' }];
		expect(assignIfChanged(prev, next)).toBe(prev);
	});

	it('handles null and empty array', () => {
		expect(assignIfChanged(null, null)).toBe(null);
		expect(assignIfChanged([], [])).toEqual([]);
	});

	it('treats dashboard display-only diffs as unchanged', () => {
		const prev = {
			total_balance: 100,
			total_forecast: 100,
			accounts: [{ id: 'a1', balance: 100, balance_display: '100,00 ₽' }],
			debts_summary: { i_owe: 0 },
			recent_transactions: []
		};
		const next = {
			total_balance: 100,
			total_forecast: 100,
			accounts: [{ id: 'a1', balance: 100, balance_display: '100.00 RUB' }],
			debts_summary: { i_owe: 0 },
			recent_transactions: []
		};
		expect(assignIfChanged(prev, next)).toBe(prev);
	});

	it('updates dashboard when balance changes', () => {
		const prev = {
			total_balance: 100,
			accounts: [],
			debts_summary: {}
		};
		const next = {
			total_balance: 200,
			accounts: [],
			debts_summary: {}
		};
		expect(assignIfChanged(prev, next)).toBe(next);
	});
});

describe('stableEqual / stripDisplayFields', () => {
	it('strips *_display recursively', () => {
		expect(
			stripDisplayFields({
				balance: 1,
				balance_display: '1',
				nested: { x: 2, x_display: '2' }
			})
		).toEqual({ balance: 1, nested: { x: 2 } });
	});

	it('stableEqual ignores display fields', () => {
		expect(stableEqual({ n: 1, n_display: 'a' }, { n: 1, n_display: 'b' })).toBe(true);
		expect(stableEqual({ n: 1 }, { n: 2 })).toBe(false);
	});
});

describe('isDashboardRefPath / isDashboardShape', () => {
	it('matches dashboard path and shape', () => {
		expect(isDashboardRefPath('/api/v1/dashboard')).toBe(true);
		expect(isDashboardRefPath('/api/v1/dashboard?x=1')).toBe(true);
		expect(isDashboardRefPath('/api/v1/accounts')).toBe(false);
		expect(isDashboardShape({ total_balance: 0, accounts: [], debts_summary: {} })).toBe(true);
		expect(isDashboardShape({ total: 1 })).toBe(false);
	});
});

describe('diffNewIds', () => {
	it('finds new row ids', () => {
		const prev = [{ id: 'a' }, { id: 'b' }];
		const next = [{ id: 'b' }, { id: 'c' }];
		expect([...diffNewIds(prev, next)]).toEqual(['c']);
	});
});
