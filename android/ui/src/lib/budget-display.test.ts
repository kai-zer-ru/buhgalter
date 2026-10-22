import { describe, expect, it, vi } from 'vitest';
import {
	budgetRemainingCell,
	budgetSpentPreviewReady,
	budgetStatusLine,
	isBudgetExceeded,
	isBudgetMonthSpentPreviewable
} from './budget-display';

vi.mock('$lib/i18n', () => ({
	tr: (key: string, options?: { values?: Record<string, string> }) => {
		const v = options?.values ?? {};
		if (key === 'budget.exceeded') return `exceeded:${v.amount}:${v.percent}`;
		if (key === 'budget.remaining_line') return `remaining:${v.amount}:${v.percent}`;
		return key;
	}
}));

const base = {
	remaining: 12_000,
	remaining_display: '120.00',
	percent: 60,
	spent: 18_000,
	planned: 30_000
};

describe('budget-display', () => {
	it('detects exceeded budget', () => {
		expect(isBudgetExceeded(base)).toBe(false);
		expect(isBudgetExceeded({ ...base, remaining: -380, spent: 33_800, percent: 107 })).toBe(true);
	});

	it('formats exceeded line with positive overshoot', () => {
		expect(
			budgetStatusLine({
				...base,
				remaining: -380,
				remaining_display: '-380.00',
				spent: 33_800,
				planned: 30_000,
				percent: 107
			})
		).toBe('exceeded:38.00:107');
	});

	it('formats remaining line when under budget', () => {
		expect(budgetStatusLine(base)).toBe('remaining:120.00:60');
	});

	it('formats stats cell for exceeded', () => {
		expect(
			budgetRemainingCell({
				...base,
				remaining: -380,
				spent: 33_800,
				percent: 107
			})
		).toBe('exceeded:38.00:107');
	});

	it('checks spent preview readiness by scope', () => {
		expect(budgetSpentPreviewReady('all_expense', '', '')).toBe(true);
		expect(budgetSpentPreviewReady('category', '', '')).toBe(false);
		expect(budgetSpentPreviewReady('category', 'c1', '')).toBe(true);
		expect(budgetSpentPreviewReady('subcategory', 'c1', '')).toBe(false);
		expect(budgetSpentPreviewReady('subcategory', 'c1', 's1')).toBe(true);
	});

	it('allows spent preview for current and past months only', () => {
		expect(isBudgetMonthSpentPreviewable('2026-07', '2026-07')).toBe(true);
		expect(isBudgetMonthSpentPreviewable('2026-06', '2026-07')).toBe(true);
		expect(isBudgetMonthSpentPreviewable('2026-08', '2026-07')).toBe(false);
	});
});
