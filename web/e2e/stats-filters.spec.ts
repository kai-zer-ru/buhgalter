import { test, expect } from '@playwright/test';
import { waitAppReady } from './helpers/auth';
import { createCashAccount, createExpense } from './helpers/setup-data';
import { selectCombobox } from './helpers/transactions';

test('stats filters by type expense', async ({ page }) => {
	const account = await createCashAccount(page);
	await createExpense(page, account.id, '31.00', 'E2E stats expense');

	await page.goto('/stats');
	await waitAppReady(page);

	await selectCombobox(page, 'stats-type', { label: 'Расход' });
	await expect(page.getByRole('heading', { name: 'По категориям' })).toBeVisible();
	await expect(page.getByRole('heading', { name: 'По периодам' })).toBeVisible();
});

test('stats group by day updates period section', async ({ page }) => {
	await page.goto('/stats');
	await waitAppReady(page);

	await selectCombobox(page, 'stats-group-by', { label: 'По дням' });
	await expect(page.getByRole('heading', { name: 'По периодам' })).toBeVisible();
});

test('stats search shows matching operations', async ({ page }) => {
	const account = await createCashAccount(page);
	const unique = `E2E stats search ${Date.now()}`;
	await createExpense(page, account.id, '19.00', unique);

	await page.goto('/stats');
	await waitAppReady(page);

	await page.getByPlaceholder('Комментарий операции').fill(unique);
	await expect(page.getByRole('heading', { name: 'Результаты поиска' })).toBeVisible({
		timeout: 15_000
	});
});

test('stats reset filters button works', async ({ page }) => {
	await page.goto('/stats');
	await waitAppReady(page);

	await selectCombobox(page, 'stats-type', { label: 'Расход' });
	await page.getByRole('button', { name: 'Сбросить' }).click();
	await expect(page.locator('#stats-type')).toContainText('Все');
});
