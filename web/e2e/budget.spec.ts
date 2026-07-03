import { test, expect } from '@playwright/test';
import { apiJSON, waitAppReady } from './helpers/auth';
import { createCashAccount, createExpense } from './helpers/setup-data';
import { selectLabeledCombobox } from './helpers/combobox';

test('budget: create limit → expense → progress', async ({ page }) => {
	const accountName = `Budget Acc ${Date.now()}`;
	const account = await createCashAccount(page, accountName);
	const meta = await apiJSON<{
		expense_categories: { id: string; name: string; is_system: boolean }[];
	}>(page, 'GET', '/api/v1/ui/meta');
	const summary = await apiJSON<{
		items: { scope: string; category_id?: string }[];
	}>(page, 'GET', '/api/v1/budgets/summary');
	const usedCategoryIds = new Set(
		summary.items
			.filter((i) => i.scope === 'category' && i.category_id)
			.map((i) => i.category_id as string)
	);
	const category = meta.expense_categories.find((c) => !c.is_system && !usedCategoryIds.has(c.id))!;

	await page.goto('/budget');
	await waitAppReady(page);
	await page.getByRole('button', { name: 'Добавить' }).click();
	await page.getByLabel('Название').fill('Продукты E2E');
	await selectLabeledCombobox(page, 'Область', { label: 'Категория' });
	await selectLabeledCombobox(page, 'Категория', { label: category.name });
	await page.getByLabel('Лимит').fill('1000');
	await page.getByRole('button', { name: 'Создать' }).click();
	await expect(page.getByText('Продукты E2E')).toBeVisible({ timeout: 10_000 });

	await createExpense(page, account.id, '200.00', 'E2E budget progress', category.id);

	await page.goto('/budget');
	await waitAppReady(page);
	await expect(page.getByText(/200\.00.*1[\s\u00a0]?000\.00/)).toBeVisible({ timeout: 10_000 });
});

test('budget form: scope field visibility', async ({ page }) => {
	const summary = await apiJSON<{
		items: { scope: string }[];
	}>(page, 'GET', '/api/v1/budgets/summary');
	const hasAllExpense = summary.items.some((i) => i.scope === 'all_expense');

	const meta = await apiJSON<{
		expense_categories: { id: string; name: string; is_system: boolean }[];
	}>(page, 'GET', '/api/v1/ui/meta');
	const category = meta.expense_categories.find((c) => !c.is_system)!;
	await apiJSON(page, 'POST', `/api/v1/categories/${category.id}/subcategories`, {
		name: `E2E Sub ${Date.now()}`
	});

	await page.goto('/budget');
	await waitAppReady(page);
	await page.getByRole('button', { name: 'Добавить' }).click();

	const categoryField = page.getByLabel('Категория', { exact: true });
	const subcategoryField = page.getByLabel('Подкатегория', { exact: true });
	const amountField = page.getByLabel('Лимит');

	if (!hasAllExpense) {
		await selectLabeledCombobox(page, 'Область', { label: 'Все расходы' });
		await expect(categoryField).not.toBeVisible();
		await expect(subcategoryField).not.toBeVisible();
		await expect(amountField).toBeVisible();
	}

	await selectLabeledCombobox(page, 'Область', { label: 'Категория' });
	await expect(categoryField).toBeVisible();
	await expect(subcategoryField).not.toBeVisible();
	await expect(amountField).toBeVisible();

	await selectLabeledCombobox(page, 'Область', { label: 'Подкатегория' });
	await expect(categoryField).toBeVisible();
	await expect(subcategoryField).toBeVisible();
	await expect(amountField).toBeVisible();
});
