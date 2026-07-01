import { test, expect } from '@playwright/test';
import { waitAppReady } from './helpers/auth';
import { createCashAccount, createExpense } from './helpers/setup-data';
import { selectLabeledCombobox } from './helpers/transactions';

test('budget: create limit → expense → progress', async ({ page }) => {
	const accountName = `Budget Acc ${Date.now()}`;
	await createCashAccount(page, accountName, '5000');

	await page.goto('/budget');
	await waitAppReady(page);
	await page.getByRole('button', { name: 'Добавить' }).click();
	await page.getByLabel('Название').fill('Продукты E2E');
	await selectLabeledCombobox(page, 'Область', 'Категория');
	const catSelect = page.locator('label').filter({ hasText: 'Категория' }).locator('..').getByRole('combobox');
	await catSelect.click();
	await page.getByRole('option').first().click();
	await page.getByLabel('Лимит').fill('1000');
	await page.getByRole('button', { name: 'Сохранить' }).click();
	await expect(page.getByText('Продукты E2E')).toBeVisible({ timeout: 10_000 });

	await createExpense(page, { amount: '200', account: accountName });

	await page.goto('/budget');
	await waitAppReady(page);
	await expect(page.getByText(/200\.00.*1[\s\u00a0]?000\.00/)).toBeVisible({ timeout: 10_000 });
});
