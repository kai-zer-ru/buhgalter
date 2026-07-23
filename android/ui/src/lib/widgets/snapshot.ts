import type {
	Account,
	BudgetSummaryItem,
	Credit,
	Dashboard,
	Debt,
	Transaction
} from '$lib/api/client';

export type WidgetUpcomingKind = 'credit' | 'debt' | 'future';

export type WidgetUpcomingItem = {
	kind: WidgetUpcomingKind;
	id: string;
	title: string;
	subtitle: string;
	date: string;
	amount_display: string;
	route: string;
};

export type WidgetAccountItem = {
	id: string;
	name: string;
	balance_display: string;
	is_primary: boolean;
};

export type WidgetBudgetItem = {
	name: string;
	spent_display: string;
	planned_display: string;
	remaining_display: string;
	percent: number;
	status: string;
};

export type WidgetSnapshot = {
	updated_at: string;
	currency: string;
	language: string;
	total_balance_display: string;
	total_forecast_display: string;
	show_forecast: boolean;
	credit_cards_display: string | null;
	budget: WidgetBudgetItem | null;
	upcoming: WidgetUpcomingItem[];
	accounts: WidgetAccountItem[];
};

export type BuildWidgetSnapshotInput = {
	dashboard: Dashboard;
	accounts: Account[];
	budgetItems: BudgetSummaryItem[];
	credits: Credit[];
	debts: Debt[];
	futureTx: Transaction[];
	currency: string;
	language: string;
	now?: Date;
};

function formatCentsDisplay(cents: number, currency: string): string {
	const value = (cents / 100).toLocaleString(undefined, {
		minimumFractionDigits: 2,
		maximumFractionDigits: 2
	});
	return `${value} ${currency}`;
}

function pickBudget(items: BudgetSummaryItem[]): WidgetBudgetItem | null {
	if (items.length === 0) return null;
	const all = items.find((b) => b.scope === 'all_expense');
	const pick =
		all ??
		[...items].filter((b) => b.scope !== 'all_expense').sort((a, b) => b.percent - a.percent)[0];
	if (!pick) return null;
	return {
		name: pick.name,
		spent_display: pick.spent_display,
		planned_display: pick.planned_display,
		remaining_display: pick.remaining_display,
		percent: pick.percent,
		status: pick.status
	};
}

function parseSortDate(raw: string): number {
	const t = Date.parse(raw);
	return Number.isFinite(t) ? t : Number.POSITIVE_INFINITY;
}

/** Merge credits / unsettled debts / future txs into a dated list (nearest first). */
export function buildUpcomingItems(
	credits: Credit[],
	debts: Debt[],
	futureTx: Transaction[],
	currency = 'RUB',
	limit = 5
): WidgetUpcomingItem[] {
	const items: WidgetUpcomingItem[] = [];

	for (const c of credits) {
		if (c.status !== 'active' || !c.next_payment_date) continue;
		items.push({
			kind: 'credit',
			id: c.id,
			title: c.name?.trim() || 'Credit',
			subtitle: c.debit_account_name || '',
			date: c.next_payment_date,
			amount_display:
				c.next_payment_amount != null
					? formatCentsDisplay(c.next_payment_amount, currency)
					: c.monthly_payment_display,
			route: `/credits/${c.id}`
		});
	}

	for (const d of debts) {
		if (d.is_settled || !d.due_date) continue;
		items.push({
			kind: 'debt',
			id: d.id,
			title: d.debtor_name,
			subtitle: d.direction === 'borrowed' ? 'i_owe' : 'owed_to_me',
			date: d.due_date,
			amount_display: d.amount_display,
			route: `/debtors/${d.debtor_id}`
		});
	}

	for (const tx of futureTx) {
		items.push({
			kind: 'future',
			id: tx.id,
			title: tx.description?.trim() || tx.category_name || 'Payment',
			subtitle: tx.account_name || '',
			date: tx.transaction_date,
			amount_display: tx.amount_display,
			route: '/transactions'
		});
	}

	items.sort((a, b) => parseSortDate(a.date) - parseSortDate(b.date));
	return items.slice(0, limit).map((item) => ({
		...item,
		amount_display: item.amount_display.trim()
	}));
}

export function buildWidgetSnapshot(input: BuildWidgetSnapshotInput): WidgetSnapshot {
	const { dashboard, currency } = input;
	const cards = dashboard.credit_cards_summary;
	return {
		updated_at: (input.now ?? new Date()).toISOString(),
		currency,
		language: input.language || 'ru',
		total_balance_display: formatCentsDisplay(dashboard.total_balance, currency),
		total_forecast_display: formatCentsDisplay(dashboard.total_forecast, currency),
		show_forecast: dashboard.total_forecast !== dashboard.total_balance,
		credit_cards_display: cards ? formatCentsDisplay(cards.total_balance, currency) : null,
		budget: pickBudget(input.budgetItems),
		upcoming: buildUpcomingItems(input.credits, input.debts, input.futureTx, currency),
		accounts: input.accounts
			.filter((a) => a.status === 'active')
			.map((a) => ({
				id: a.id,
				name: a.name,
				balance_display: a.balance_display,
				is_primary: a.is_primary
			}))
	};
}
