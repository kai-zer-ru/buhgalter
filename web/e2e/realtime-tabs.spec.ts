import { test, expect } from '@playwright/test';
import { waitAppReady } from './helpers/auth';
import { createCashAccount, createExpense } from './helpers/setup-data';

test('realtime: second tab sees expense without manual reload', async ({ page, context }) => {
	const unique = Date.now();
	const account = await createCashAccount(page, `E2E Realtime ${unique}`);

	const path = `/accounts/${account.id}`;
	await page.goto(path);
	await waitAppReady(page);
	await Promise.race([
		page.waitForEvent('websocket', { timeout: 15_000 }),
		page.waitForFunction(() => true, undefined, { timeout: 1_500 }).catch(() => undefined)
	]);

	const tab2 = await context.newPage();
	await tab2.goto(path);
	await waitAppReady(tab2);
	await Promise.race([
		tab2.waitForEvent('websocket', { timeout: 15_000 }),
		tab2.waitForFunction(() => true, undefined, { timeout: 1_500 }).catch(() => undefined)
	]);

	await expect(tab2.getByRole('heading', { name: account.name })).toBeVisible({ timeout: 10_000 });
	await expect(tab2.getByText(/1[\s\u00a0]?000[.,]00/).first()).toBeVisible({ timeout: 10_000 });

	await createExpense(page, account.id, '25.00', `E2E realtime tx ${unique}`);

	await expect(tab2.getByText(/975[.,]00/).first()).toBeVisible({ timeout: 20_000 });
});
