import type { BudgetSummaryItem } from '$lib/api/client';
import { tr } from '$lib/i18n';
import { fromCents } from '$lib/money';

type BudgetStatusFields = Pick<
	BudgetSummaryItem,
	'remaining' | 'remaining_display' | 'percent' | 'spent' | 'planned'
>;

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
			values: { amount: overshootAmount(item), percent: String(item.percent) }
		});
	}
	return tr('budget.remaining_line', {
		values: { amount: item.remaining_display, percent: String(item.percent) }
	});
}

/** Table cell for «Остаток» column (stats). */
export function budgetRemainingCell(item: BudgetStatusFields): string {
	if (isBudgetExceeded(item)) {
		return tr('budget.exceeded', {
			values: { amount: overshootAmount(item), percent: String(item.percent) }
		});
	}
	return item.remaining_display;
}
