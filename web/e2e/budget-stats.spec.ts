import { test, expect } from '@playwright/test';
import { waitAppReady } from './helpers/auth';
import { createCashAccount } from './helpers/setup-data';

test('stats: budget columns for expense category', async ({ page }) => {
	const accountName = `Stats Budget ${Date.now()}`;
	await createCashAccount(page, accountName, '10000');

	await page.goto('/budget');
	await waitAppReady(page);
	await page.getByRole('button', { name: 'Добавить' }).click();
	await page.getByLabel('Название').fill('Stats Cat Budget');
	await page.getByLabel('Лимит').fill('5000');
	await page.getByRole('button', { name: 'Сохранить' }).click();
	await expect(page.getByText('Stats Cat Budget')).toBeVisible({ timeout: 10_000 });

	await page.goto('/stats');
	await waitAppReady(page);
	await expect(page.getByRole('columnheader', { name: 'Бюджет' })).toBeVisible({ timeout: 10_000 });
	await expect(page.getByRole('columnheader', { name: 'Остаток' })).toBeVisible();
});
