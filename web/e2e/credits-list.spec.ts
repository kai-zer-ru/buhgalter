import { expect, test } from '@playwright/test';
import { apiJSON, formatUTCDateTime, waitAppReady } from './helpers/auth';
import { createCashAccount } from './helpers/setup-data';

test('credits list: clicking a row (not the name link) opens the credit', async ({ page }) => {
	const account = await createCashAccount(page);
	const name = `E2E Row Click ${Date.now()}`;
	const credit = await apiJSON<{ id: string }>(page, 'POST', '/api/v1/credits', {
		name,
		principal_amount: '3000.00',
		issue_date: formatUTCDateTime(new Date(Date.now() - 24 * 60 * 60 * 1000)),
		term_months: 3,
		interest_rate: 0,
		payment_interval: 'month',
		debit_account_id: account.id,
		added_retroactively: false,
		create_transactions: true
	});

	await page.goto('/credits');
	await waitAppReady(page);
	const row = page.getByRole('row', { name: new RegExp(name) });
	await expect(row).toBeVisible({ timeout: 10_000 });
	// The payment cell has no link, so this exercises the row handler itself.
	await row.getByRole('cell').nth(2).click();
	await expect(page).toHaveURL(new RegExp(`/credits/${credit.id}$`), { timeout: 10_000 });
});
