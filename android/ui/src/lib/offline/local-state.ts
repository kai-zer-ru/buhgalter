import type {
	Account,
	AccountBalanceSummary,
	CreditCardsSummary,
	Dashboard
} from '$lib/api/client';
import { fromCents } from '$lib/money';
import { isCreditCard } from '$lib/credit-card';
import { applyAccountDeltas, computeOutboxAccountDeltas } from '$lib/offline/balance-overlay';

function adjustBalanceSummary(
	acc: AccountBalanceSummary,
	deltas: ReturnType<typeof computeOutboxAccountDeltas>
): AccountBalanceSummary {
	const next = applyAccountDeltas(acc.balance, acc.forecast_balance, acc.id, deltas);
	return {
		...acc,
		balance: next.balance,
		balance_display: fromCents(next.balance),
		forecast_balance: next.forecast,
		forecast_display: fromCents(next.forecast)
	};
}

function creditCardsSummaryFromAccounts(
	accounts: AccountBalanceSummary[]
): CreditCardsSummary | null {
	let count = 0;
	let totalBal = 0;
	let totalForecast = 0;
	let totalLimit = 0;
	for (const acc of accounts) {
		if (!isCreditCard(acc)) continue;
		count++;
		totalBal += acc.balance;
		totalForecast += acc.forecast_balance;
		if (acc.credit_limit != null) totalLimit += acc.credit_limit;
	}
	if (count === 0) return null;
	return {
		count,
		total_balance: totalBal,
		total_forecast: totalForecast,
		total_limit: totalLimit,
		total_balance_display: fromCents(totalBal),
		total_forecast_display: fromCents(totalForecast),
		total_limit_display: fromCents(totalLimit)
	};
}

export function applyOutboxToDashboard(dashboard: Dashboard, tz: string): Dashboard {
	const deltas = computeOutboxAccountDeltas(tz);
	const accounts = dashboard.accounts.map((acc) => adjustBalanceSummary(acc, deltas));
	let totalBalance = 0;
	let totalForecast = 0;
	for (const acc of accounts) {
		if (isCreditCard(acc)) continue;
		totalBalance += acc.balance;
		totalForecast += acc.forecast_balance;
	}
	return {
		...dashboard,
		accounts,
		total_balance: totalBalance,
		total_forecast: totalForecast,
		credit_cards_summary: creditCardsSummaryFromAccounts(accounts)
	};
}

export function applyOutboxToAccountBalance(
	summary: AccountBalanceSummary,
	tz: string
): AccountBalanceSummary {
	return adjustBalanceSummary(summary, computeOutboxAccountDeltas(tz));
}

export function applyOutboxToAccount(account: Account, tz: string): Account {
	const deltas = computeOutboxAccountDeltas(tz);
	const next = applyAccountDeltas(account.balance, account.balance, account.id, deltas);
	return {
		...account,
		balance: next.balance,
		balance_display: fromCents(next.balance)
	};
}

export function applyOutboxToAccounts(accounts: Account[], tz: string): Account[] {
	const deltas = computeOutboxAccountDeltas(tz);
	return accounts.map((acc) => {
		const next = applyAccountDeltas(acc.balance, acc.balance, acc.id, deltas);
		return {
			...acc,
			balance: next.balance,
			balance_display: fromCents(next.balance)
		};
	});
}
