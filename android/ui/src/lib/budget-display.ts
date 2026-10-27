import type { BudgetScope, BudgetSummaryItem } from '$lib/api/client';
import { tr } from '$lib/i18n';
import { formatBalance } from './finance';
import { formatMoneyDisplay, fromCents } from './money';

type BudgetStatusFields = Pick<
	BudgetSummaryItem,
	'remaining' | 'remaining_display' | 'percent' | 'spent' | 'planned'
> & { currency?: string };

function moneyWithCurrency(display: string, currency?: string): string {
	return currency ? formatBalance(display, currency) : formatMoneyDisplay(display);
}

export function isBudgetExceeded(item: BudgetStatusFields): boolean {
	return item.remaining < 0 || item.spent > item.planned;
}

function overshootAmount(item: BudgetStatusFields): string {
	return fromCents(Math.max(0, item.spent - item.planned));
}

/** Card / widget line: «Остаток: …» or «Превышение на …». */
export function budgetStatusLine(item: BudgetStatusFields): string {
	if (isBudgetExceeded(item)) {
		return tr('budget.exceeded', {
			values: {
				amount: moneyWithCurrency(overshootAmount(item), item.currency),
				percent: String(item.percent)
			}
		});
	}
	return tr('budget.remaining_line', {
		values: {
			amount: moneyWithCurrency(item.remaining_display, item.currency),
			percent: String(item.percent)
		}
	});
}

/** Table cell for «Остаток» column (stats). */
export function budgetRemainingCell(item: BudgetStatusFields): string {
	if (isBudgetExceeded(item)) {
		return tr('budget.exceeded', {
			values: {
				amount: moneyWithCurrency(overshootAmount(item), item.currency),
				percent: String(item.percent)
			}
		});
	}
	return moneyWithCurrency(item.remaining_display, item.currency);
}

/** True when create/edit form can load spent preview for the selected scope. */
export function budgetSpentPreviewReady(
	scope: BudgetScope,
	categoryId: string,
	subcategoryId: string
): boolean {
	if (scope === 'all_expense') return true;
	if (scope === 'category') return categoryId !== '';
	if (scope === 'subcategory') return subcategoryId !== '';
	return false;
}

/** Show spent hint only for current or past month (YYYY-MM lexicographic compare). */
export function isBudgetMonthSpentPreviewable(month: string, currentMonth: string): boolean {
	return month <= currentMonth;
}
