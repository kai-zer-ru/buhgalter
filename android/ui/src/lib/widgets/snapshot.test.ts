import { describe, expect, it } from 'vitest';
import type {
	Credit,
	Dashboard,
	Debt,
	RecurringOperation,
	Subscription,
	Transaction
} from '$lib/api/client';
import { buildUpcomingItems, buildWidgetSnapshot } from './snapshot';

const dash = (partial: Partial<Dashboard> = {}): Dashboard => ({
	total_balance: 10050,
	total_forecast: 9050,
	accounts: [],
	recent_transactions: [],
	debts_summary: {
		i_owe: 0,
		owed_to_me: 0,
		overdue_i_owe: 0,
		overdue_owed_to_me: 0,
		active_count: 0
	},
	...partial
});

describe('buildUpcomingItems', () => {
	it('merges and sorts by date with formatted amounts', () => {
		const credits = [
			{
				id: 'c1',
				name: 'Bank',
				status: 'active',
				next_payment_date: '2026-08-10',
				next_payment_amount: 50000,
				debit_account_name: 'Main',
				monthly_payment_display: '500.00'
			} as Credit
		];
		const debts = [
			{
				id: 'd1',
				debtor_id: 'p1',
				debtor_name: 'Ivan',
				direction: 'borrowed',
				due_date: '2026-08-01',
				amount: 20000,
				amount_display: '200.00',
				is_settled: false
			} as Debt
		];
		const future = [
			{
				id: 't1',
				description: 'Rent',
				transaction_date: '2026-08-05',
				amount: 30000,
				amount_display: '300.00',
				account_name: 'Main'
			} as Transaction
		];
		const items = buildUpcomingItems(credits, debts, future, 'RUB', 5);
		expect(items.map((i) => i.id)).toEqual(['d1', 't1', 'c1']);
		expect(items[0].amount_display).toBe('200.00 ₽');
		expect(items[2].amount_display).toBe('500.00 ₽');
		expect(items[0].route).toBe('/debtors/p1');
		expect(items[2].route).toBe('/credits/c1');
	});

	it('includes subscriptions and recurring operations', () => {
		const subscriptions = [
			{
				id: 's1',
				name: 'Netflix',
				active: true,
				next_run_at: '2026-08-03T12:00:00Z',
				amount: 1500,
				amount_display: '15.00',
				account_name: 'Card'
			} as Subscription
		];
		const recurring = [
			{
				id: 'r1',
				description: 'Salary',
				active: true,
				next_run_at: '2026-08-02T09:00:00Z',
				amount: 100000,
				amount_display: '1000.00',
				account_name: 'Main',
				category_name: 'Income'
			} as RecurringOperation
		];
		const items = buildUpcomingItems([], [], [], 'RUB', 5, subscriptions, recurring);
		expect(items.map((i) => i.id)).toEqual(['r1', 's1']);
		expect(items[0].amount_display).toBe('1 000.00 ₽');
		expect(items[1].amount_display).toBe('15.00 ₽');
	});
});

describe('buildWidgetSnapshot', () => {
	it('formats funds by account type', () => {
		const snap = buildWidgetSnapshot({
			dashboard: dash({
				total_balance: 100050,
				total_forecast: 90500,
				credit_cards_summary: {
					total_balance: 1001400,
					total_forecast: 1001400,
					total_limit: 0,
					count: 1,
					total_balance_display: '10014.00',
					total_forecast_display: '10014.00',
					total_limit_display: '0'
				},
				accounts: [
					{
						id: 'c1',
						name: 'Cash',
						type: 'cash',
						balance: 50000,
						balance_display: '500.00',
						forecast_balance: 50000,
						forecast_display: '500.00',
						has_future_this_month: false,
						is_primary: false
					},
					{
						id: 'b1',
						name: 'Bank',
						type: 'bank',
						balance: 4099253,
						balance_display: '40 992.53',
						forecast_balance: 4099253,
						forecast_display: '40 992.53',
						has_future_this_month: false,
						is_primary: true
					},
					{
						id: 'cc1',
						name: 'Card',
						type: 'credit_card',
						balance: 1001400,
						balance_display: '10 014.00',
						forecast_balance: 1001400,
						forecast_display: '10 014.00',
						has_future_this_month: false,
						is_primary: false
					}
				]
			}),
			budgetItems: [
				{
					id: 'b1',
					name: 'All',
					scope: 'all_expense',
					spent: 4000,
					planned: 10000,
					remaining: 6000,
					spent_display: '40.00',
					planned_display: '100.00',
					remaining_display: '60.00',
					percent: 40,
					status: 'ok'
				} as never
			],
			credits: [],
			debts: [],
			futureTx: [],
			currency: 'RUB',
			language: 'ru',
			now: new Date('2026-07-15T12:00:00Z')
		});
		expect(snap.cash_display).toBe('500.00 ₽');
		expect(snap.bank_display).toBe('40 992.53 ₽');
		expect(snap.credit_funds_display).toBe('10 014.00 ₽');
		expect(snap.budget?.spent_display).toBe('40.00 ₽');
	});
});
