import { describe, expect, it } from 'vitest';
import type { Account, Credit, Dashboard, Debt, Transaction } from '$lib/api/client';
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

const account = (partial: Partial<Account> = {}): Account =>
	({
		id: 'a1',
		name: 'Cash',
		type: 'cash',
		bank_id: null,
		initial_balance: 0,
		balance: 10050,
		balance_display: '100.50',
		status: 'active',
		is_primary: true,
		created_at: '',
		updated_at: '',
		...partial
	}) as Account;

describe('buildUpcomingItems', () => {
	it('merges and sorts by date', () => {
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
				amount_display: '200.00',
				is_settled: false
			} as Debt
		];
		const future = [
			{
				id: 't1',
				description: 'Rent',
				transaction_date: '2026-08-05',
				amount_display: '300.00',
				account_name: 'Main'
			} as Transaction
		];
		const items = buildUpcomingItems(credits, debts, future, 'RUB', 5);
		expect(items.map((i) => i.id)).toEqual(['d1', 't1', 'c1']);
		expect(items[0].route).toBe('/debtors/p1');
		expect(items[2].route).toBe('/credits/c1');
	});
});

describe('buildWidgetSnapshot', () => {
	it('formats balance and picks budget', () => {
		const snap = buildWidgetSnapshot({
			dashboard: dash(),
			accounts: [account()],
			budgetItems: [
				{
					id: 'b1',
					name: 'All',
					scope: 'all_expense',
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
		expect(snap.total_balance_display).toContain('RUB');
		expect(snap.show_forecast).toBe(true);
		expect(snap.budget?.name).toBe('All');
		expect(snap.accounts[0].is_primary).toBe(true);
	});
});
