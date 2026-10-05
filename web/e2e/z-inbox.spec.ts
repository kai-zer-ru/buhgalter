import { expect, test } from '@playwright/test';
import { apiJSON, formatUTCDateTime, waitAppReady } from './helpers/auth';
import { createCashAccount } from './helpers/setup-data';
import { selectCombobox } from './helpers/transactions';
import { openRowActions } from './helpers/ui';

test('inbox lists uncategorized transaction and clears after category assign', async ({ page }) => {
	const account = await createCashAccount(page);
	const tag = `E2E inbox ${Date.now()}`;
	const amount = '15.00';
	const created = await apiJSON<{ id: string }>(page, 'POST', '/api/v1/transactions', {
		account_id: account.id,
		type: 'expense',
		amount,
		description: tag,
		transaction_date: formatUTCDateTime(new Date())
	});
	const inboxPeek = await apiJSON<{ data: { id: string }[] }>(
		page,
		'GET',
		'/api/v1/transactions/inbox?limit=50'
	);
	expect(inboxPeek.data.some((t) => t.id === created.id)).toBe(true);

	await page.goto('/transactions/inbox');
	await waitAppReady(page);
	const row = page.getByRole('row').filter({ hasText: account.name }).filter({ hasText: amount });
	await expect(row).toBeVisible({ timeout: 15_000 });
	await openRowActions(row);
	await page.getByRole('menuitem', { name: /изменить|edit/i }).click();
	const dialog = page.getByRole('dialog');
	await expect(dialog).toBeVisible();
	await selectCombobox(page, 'tx-category', { index: 0 });
	await dialog.getByRole('button', { name: /сохранить|save/i }).click();
	await expect(dialog).toHaveCount(0, { timeout: 15_000 });

	await expect(row).toHaveCount(0, { timeout: 15_000 });
});

test('nav inbox hidden when queue empty', async ({ page }) => {
	const { count } = await apiJSON<{ count: number }>(
		page,
		'GET',
		'/api/v1/transactions/inbox/count'
	);
	test.skip(count > 0, 'inbox not empty in this environment');

	await page.goto('/');
	await waitAppReady(page);
	const otherBtn = page.locator('header nav').getByRole('button', { name: /прочее|other/i });
	await otherBtn.click();
	await expect(page.getByRole('menuitem', { name: /разобрать|inbox/i })).toHaveCount(0);
});

test('nav inbox badge shows count after uncategorized create', async ({ page }) => {
	const account = await createCashAccount(page);
	await apiJSON(page, 'POST', '/api/v1/transactions', {
		account_id: account.id,
		type: 'expense',
		amount: '1.00',
		description: `E2E badge ${Date.now()}`,
		transaction_date: formatUTCDateTime(new Date())
	});
	await page.goto('/');
	await waitAppReady(page);
	const otherBtn = page.locator('header nav').getByRole('button', { name: /прочее|other/i });
	await expect
		.poll(async () => otherBtn.locator('.nav-badge').isVisible(), { timeout: 15_000 })
		.toBe(true);
	await otherBtn.click();
	await expect(
		page.getByRole('menuitem', { name: /разобрать|inbox/i }).locator('.nav-badge')
	).toBeVisible();
});
