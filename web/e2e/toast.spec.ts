import { test, expect } from '@playwright/test';
import { waitAppReady } from './helpers/auth';
import { createCashAccount } from './helpers/setup-data';
import { expectToast, rowMenuAction } from './helpers/ui';

test('toast success after inline account save', async ({ page }) => {
	const unique = Date.now();
	const account = await createCashAccount(page, `Toast Cash ${unique}`);
	const newName = `Toast Cash updated ${unique}`;

	await page.goto('/accounts');
	await waitAppReady(page);

	const card = page.locator('.card').filter({ hasText: account.name }).first();
	await rowMenuAction(page, card, 'Редактировать');
	await page.locator(`#edit-name-${account.id}`).fill(newName);
	await page
		.locator(`#edit-name-${account.id}`)
		.locator('xpath=ancestor::form[1]')
		.getByRole('button', { name: 'Сохранить' })
		.click();

	await expectToast(page, 'success', 'Сохранено');
});

test('toast error on invalid login', async ({ page }) => {
	await page.goto('/login');
	await expect(page.getByText('Загрузка…')).toHaveCount(0, { timeout: 20_000 });
	await expect(page.getByLabel('Логин')).toBeVisible();
	await page.getByLabel('Логин').fill('invalid-user-e2e');
	await page.getByLabel('Пароль').fill('wrong-password');
	await page.getByRole('button', { name: 'Войти' }).click();

	await expectToast(page, 'error', 'Неверный логин или пароль');
});
