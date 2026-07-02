import { test, expect } from '@playwright/test';
import { apiJSON, waitAppReady } from './helpers/auth';

test('home: accounts collapsed under spoiler on mobile', async ({ page }) => {
	const unique = Date.now();
	const account = await apiJSON<{ name: string }>(page, 'POST', '/api/v1/accounts', {
		name: `E2E Mobile Accounts ${unique}`,
		type: 'cash',
		initial_balance: '500.00'
	});

	await page.setViewportSize({ width: 390, height: 844 });
	await page.goto('/');
	await waitAppReady(page);

	const summary = page.locator('summary.dashboard-accounts-summary');
	await expect(summary).toBeVisible({ timeout: 10_000 });
	await expect(page.getByRole('link', { name: account.name })).toBeHidden();

	await summary.click();
	await expect(page.getByRole('link', { name: account.name })).toBeVisible();
});
