import type { Page } from '@playwright/test';
import { apiJSON, formatUTCDateTime } from './auth';

export async function createCashAccount(page: Page, name?: string) {
	const unique = Date.now();
	return apiJSON<{ id: string; name: string }>(page, 'POST', '/api/v1/accounts', {
		name: name ?? `E2E Account ${unique}`,
		type: 'cash',
		initial_balance: '1000.00'
	});
}

export async function createBankAccount(page: Page, name?: string) {
	const unique = Date.now();
	const banks = await apiJSON<{ id: string; name: string }[]>(page, 'GET', '/api/v1/banks');
	const bank = banks.find((b) => b.name.includes('Сбер')) ?? banks[0];
	return apiJSON<{ id: string; name: string }>(page, 'POST', '/api/v1/accounts', {
		name: name ?? `E2E Bank ${unique}`,
		type: 'bank',
		bank_id: bank.id,
		initial_balance: '500.00'
	});
}

export async function createCreditCardAccount(page: Page, name?: string) {
	const unique = Date.now();
	const banks = await apiJSON<{ id: string; name: string }[]>(page, 'GET', '/api/v1/banks');
	const bank =
		banks.find((b) => b.name.includes('Тинькофф') || b.name.includes('Т-Банк')) ?? banks[0];
	const debit = await createCashAccount(page, `E2E Debit for CC ${unique}`);
	return apiJSON<{ id: string; name: string }>(page, 'POST', '/api/v1/accounts', {
		name: name ?? `E2E Credit Card ${unique}`,
		type: 'credit_card',
		bank_id: bank.id,
		credit_limit: '65000.00',
		initial_balance: '1000.00',
		payment_account_id: debit.id
	});
}

export async function createExpense(
	page: Page,
	accountId: string,
	amount: string,
	description?: string
) {
	const desc = description ?? `E2E expense ${Date.now()}`;
	return apiJSON<{ id: string; amount_display: string; description?: string }>(
		page,
		'POST',
		'/api/v1/transactions',
		{
			account_id: accountId,
			type: 'expense',
			amount,
			description: desc,
			transaction_date: formatUTCDateTime(new Date())
		}
	);
}

export async function createPlannedExpense(
	page: Page,
	accountId: string,
	amount: string,
	description?: string
) {
	const desc = description ?? `E2E planned ${Date.now()}`;
	const future = new Date(Date.now() + 72 * 60 * 60 * 1000);
	return apiJSON<{ id: string; amount_display: string; description?: string }>(
		page,
		'POST',
		'/api/v1/transactions',
		{
			account_id: accountId,
			type: 'expense',
			amount,
			description: desc,
			transaction_date: formatUTCDateTime(future)
		}
	);
}

export async function createIncome(
	page: Page,
	accountId: string,
	amount: string,
	description?: string
) {
	const desc = description ?? `E2E income ${Date.now()}`;
	return apiJSON<{ id: string }>(page, 'POST', '/api/v1/transactions', {
		account_id: accountId,
		type: 'income',
		amount,
		description: desc,
		transaction_date: formatUTCDateTime(new Date())
	});
}

export async function createTransfer(page: Page, fromId: string, toId: string, amount: string) {
	return apiJSON(page, 'POST', '/api/v1/transfers', {
		from_account_id: fromId,
		to_account_id: toId,
		amount,
		transaction_date: formatUTCDateTime(new Date())
	});
}

export async function createCredit(page: Page, accountId: string, name?: string) {
	const unique = Date.now();
	const now = new Date();
	return apiJSON<{ id: string; name: string }>(page, 'POST', '/api/v1/credits', {
		name: name ?? `E2E Credit ${unique}`,
		principal_amount: '3000.00',
		issue_date: formatUTCDateTime(new Date(now.getTime() - 24 * 60 * 60 * 1000)),
		term_months: 3,
		interest_rate: 0,
		payment_interval: 'month',
		debit_account_id: accountId,
		added_retroactively: false,
		create_transactions: true
	});
}
