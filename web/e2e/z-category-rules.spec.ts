import { expect, test } from '@playwright/test';
import { apiJSON, waitAppReady } from './helpers/auth';
import { createCashAccount } from './helpers/setup-data';

test('category rules page loads and can open create dialog', async ({ page }) => {
	await page.goto('/category-rules');
	await expect(
		page.getByRole('heading', { name: /правила категорий|category rules/i })
	).toBeVisible();
	await page.getByRole('button', { name: /добавить правило|add rule/i }).click();
	await expect(page.getByText(/тип правила|rule type/i)).toBeVisible();
});

test('category rule via API appears in list', async ({ page }) => {
	const account = await createCashAccount(page);
	const cats = await apiJSON<{ id: string }[]>(
		page,
		'GET',
		'/api/v1/categories?type=expense&scope=personal'
	);
	await apiJSON(page, 'POST', '/api/v1/category-rules', {
		kind: 'category_account',
		category_id: cats[0].id,
		default_account_id: account.id
	});
	await page.goto('/category-rules');
	await waitAppReady(page);
	await expect(page.getByText(/категория → счёт|category → account/i).first()).toBeVisible();
});
