import { test, expect } from '@playwright/test';
import { apiJSON, formatUTCDateTime } from './helpers/auth';
import { createCashAccount } from './helpers/setup-data';

function localTimeFromUTC(txDate: string, timezone: string): string {
	const [datePart, timePart] = txDate.split(' ');
	const [y, m, d] = datePart.split('-').map(Number);
	const [hh, mm] = timePart.split(':').map(Number);
	const utc = new Date(Date.UTC(y, m - 1, d, hh, mm));
	return utc.toLocaleTimeString('en-GB', {
		timeZone: timezone,
		hour: '2-digit',
		minute: '2-digit',
		hour12: false
	});
}

test('credit auto-debit stores transaction with debit_time_local', async ({ page }) => {
	const account = await createCashAccount(page);
	const me = await apiJSON<{ timezone: string }>(page, 'GET', '/api/v1/auth/me');
	const creditName = `E2E Auto Debit ${Date.now()}`;
	const now = new Date();
	const credit = await apiJSON<{ id: string }>(page, 'POST', '/api/v1/credits', {
		name: creditName,
		principal_amount: '3000.00',
		issue_date: formatUTCDateTime(new Date(now.getTime() - 24 * 60 * 60 * 1000)),
		term_months: 3,
		interest_rate: 0,
		payment_interval: 'month',
		debit_account_id: account.id,
		debit_time_local: '10:00',
		added_retroactively: false,
		create_transactions: false
	});

	await apiJSON<{ applied: number }>(page, 'POST', `/api/v1/test/credits/${credit.id}/apply-due`);

	const txs = await apiJSON<{ data: { transaction_date: string }[] }>(
		page,
		'GET',
		`/api/v1/transactions?search=${encodeURIComponent(creditName)}`
	);
	expect(txs.data.length).toBeGreaterThan(0);
	const txDate = txs.data[0].transaction_date;
	expect(txDate).not.toMatch(/ 00:00:00$/);
	expect(localTimeFromUTC(txDate, me.timezone || 'Europe/Moscow')).toBe('10:00');
});
