import type {
	Account,
	BudgetSummaryItem,
	Credit,
	Dashboard,
	Debt,
	RecurringOperation,
	Subscription,
	Transaction
} from '$lib/api/client';
import { formatBalance } from '$lib/finance';
import { fromCents } from '$lib/money';

export type WidgetUpcomingKind = 'credit' | 'debt' | 'future' | 'subscription' | 'recurring';

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
	/** @deprecated kept for older native builds; prefer cash/bank/credit_funds */
	total_balance_display: string;
	total_forecast_display: string;
	show_forecast: boolean;
	credit_cards_display: string | null;
	cash_display: string;
	bank_display: string;
	credit_funds_display: string;
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
	subscriptions?: Subscription[];
	recurring?: RecurringOperation[];
	currency: string;
	language: string;
	now?: Date;
};

/** Same money formatting as in-app UI (`10 000.00 ₽`). */
function formatWidgetCents(cents: number, currency: string): string {
	return formatBalance(fromCents(cents), currency);
}

function formatWidgetRaw(raw: string | null | undefined, currency: string): string {
	if (!raw?.trim()) return '';
	return formatBalance(raw, currency);
}

function pickBudget(items: BudgetSummaryItem[], currency: string): WidgetBudgetItem | null {
	if (items.length === 0) return null;
	const all = items.find((b) => b.scope === 'all_expense');
	const pick =
		all ??
		[...items].filter((b) => b.scope !== 'all_expense').sort((a, b) => b.percent - a.percent)[0];
	if (!pick) return null;
	return {
		name: pick.name,
		spent_display: formatWidgetCents(pick.spent, currency),
		planned_display: formatWidgetCents(pick.planned, currency),
		remaining_display: formatWidgetCents(pick.remaining, currency),
		percent: pick.percent,
		status: pick.status
	};
}

function parseSortDate(raw: string): number {
	const t = Date.parse(raw);
	return Number.isFinite(t) ? t : Number.POSITIVE_INFINITY;
}

/** Merge credits / debts / future / subscriptions / recurring into a dated list (nearest first). */
export function buildUpcomingItems(
	credits: Credit[],
	debts: Debt[],
	futureTx: Transaction[],
	currency = 'RUB',
	limit = 5,
	subscriptions: Subscription[] = [],
	recurring: RecurringOperation[] = []
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
					? formatWidgetCents(c.next_payment_amount, currency)
					: formatWidgetRaw(c.monthly_payment_display, currency),
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
			amount_display: formatWidgetCents(d.amount, currency),
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
			amount_display: formatWidgetCents(tx.amount, currency),
			route: '/transactions'
		});
	}

	for (const s of subscriptions) {
		if (!s.active || !s.next_run_at) continue;
		items.push({
			kind: 'subscription',
			id: s.id,
			title: s.name?.trim() || 'Subscription',
			subtitle: s.account_name || '',
			date: s.next_run_at,
			amount_display: formatWidgetCents(s.amount, currency),
			route: '/subscriptions'
		});
	}

	for (const r of recurring) {
		if (!r.active || !r.next_run_at) continue;
		items.push({
			kind: 'recurring',
			id: r.id,
			title: r.description?.trim() || r.category_name || 'Recurring',
			subtitle: r.account_name || '',
			date: r.next_run_at,
			amount_display: formatWidgetCents(r.amount, currency),
			route: '/recurring-operations'
		});
	}

	items.sort((a, b) => parseSortDate(a.date) - parseSortDate(b.date));
	return items.slice(0, limit).map((item) => ({
		...item,
		amount_display: item.amount_display.trim()
	}));
}

function sumActiveBalanceByType(accounts: Account[], type: Account['type']): number {
	let sum = 0;
	for (const a of accounts) {
		if (a.status !== 'active' || a.type !== type) continue;
		sum += a.balance;
	}
	return sum;
}

export function buildWidgetSnapshot(input: BuildWidgetSnapshotInput): WidgetSnapshot {
	const { dashboard, currency, accounts } = input;
	const cards = dashboard.credit_cards_summary;
	const cashCents = sumActiveBalanceByType(accounts, 'cash');
	const bankCents = sumActiveBalanceByType(accounts, 'bank');
	const creditCents = cards?.total_balance ?? sumActiveBalanceByType(accounts, 'credit_card');
	const cashDisplay = formatWidgetCents(cashCents, currency);
	const bankDisplay = formatWidgetCents(bankCents, currency);
	const creditDisplay = formatWidgetCents(creditCents, currency);
	return {
		updated_at: (input.now ?? new Date()).toISOString(),
		currency,
		language: input.language || 'ru',
		total_balance_display: formatWidgetCents(dashboard.total_balance, currency),
		total_forecast_display: formatWidgetCents(dashboard.total_forecast, currency),
		show_forecast: dashboard.total_forecast !== dashboard.total_balance,
		credit_cards_display: cards ? creditDisplay : creditCents !== 0 ? creditDisplay : null,
		cash_display: cashDisplay,
		bank_display: bankDisplay,
		credit_funds_display: creditDisplay,
		budget: pickBudget(input.budgetItems, currency),
		upcoming: buildUpcomingItems(
			input.credits,
			input.debts,
			input.futureTx,
			currency,
			5,
			input.subscriptions ?? [],
			input.recurring ?? []
		),
		accounts: accounts
			.filter((a) => a.status === 'active')
			.map((a) => ({
				id: a.id,
				name: a.name,
				balance_display: formatWidgetCents(a.balance, currency),
				is_primary: a.is_primary
			}))
	};
}
