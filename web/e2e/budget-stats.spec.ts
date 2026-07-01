import { test, expect } from '@playwright/test';
import { apiJSON, waitAppReady } from './helpers/auth';
import { createCashAccount, createExpense } from './helpers/setup-data';
import { selectLabeledCombobox } from './helpers/combobox';

test('stats: budget columns for expense category', async ({ page }) => {
	const accountName = `Stats Budget ${Date.now()}`;
	const account = await createCashAccount(page, accountName);
	const meta = await apiJSON<{
		expense_categories: { id: string; name: string; is_system: boolean }[];
	}>(page, 'GET', '/api/v1/ui/meta');
	const category = meta.expense_categories.find((c) => !c.is_system)!;

	await page.goto('/budget');
	await waitAppReady(page);
	await page.getByRole('button', { name: 'Добавить' }).click();
	await page.getByLabel('Название').fill('Stats Cat Budget');
	await selectLabeledCombobox(page, 'Категория', { label: category.name });
	await page.getByLabel('Лимит').fill('5000');
	await page.getByRole('button', { name: 'Создать' }).click();
	await expect(page.getByText('Stats Cat Budget')).toBeVisible({ timeout: 10_000 });

	await createExpense(page, account.id, '50.00', 'E2E stats budget', category.id);

	await page.goto('/stats');
	await waitAppReady(page);
	await expect(page.getByRole('columnheader', { name: 'Бюджет' })).toBeVisible({ timeout: 10_000 });
	await expect(page.getByRole('columnheader', { name: 'Остаток' })).toBeVisible();
});
