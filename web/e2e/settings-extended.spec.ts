import { test, expect } from '@playwright/test';
import { ADMIN, waitAppReady } from './helpers/auth';
import { confirmDialog, rowMenuAction } from './helpers/ui';

test('profile: change display name', async ({ page }) => {
	const newName = `E2E Display ${Date.now()}`;

	await page.goto('/settings');
	await waitAppReady(page);
	await page.locator('#display').fill(newName);
	await page.getByRole('button', { name: 'Сохранить' }).click();

	await expect(page.getByText('Сохранено').first()).toBeVisible({
		timeout: 10_000
	});

	await page.locator('#display').fill(ADMIN.displayName);
	await page.getByRole('button', { name: 'Сохранить' }).click();
});

test('password tab loads form fields', async ({ page }) => {
	await page.goto('/settings?tab=password');
	await waitAppReady(page);

	await expect(page.locator('#old')).toBeVisible();
	await expect(page.locator('#new')).toBeVisible();
	await expect(page.locator('#confirm')).toBeVisible();
});

test('tokens: revoke created API token', async ({ page }) => {
	const tokenName = `E2E Revoke ${Date.now()}`;

	await page.goto('/settings?tab=tokens');
	await waitAppReady(page);
	await page.locator('#token-name').fill(tokenName);
	await page.getByRole('button', { name: 'Создать' }).click();
	await page.getByRole('button', { name: 'Закрыть' }).click();
	await expect(page.getByText(tokenName).first()).toBeVisible();

	const row = page.getByRole('row', { name: new RegExp(tokenName) });
	await row.getByRole('button', { name: 'Удалить' }).click();
	await confirmDialog(page, 'Удалить');

	await expect(page.getByText(tokenName)).toHaveCount(0, { timeout: 10_000 });
});

test('categories: delete expense category', async ({ page }) => {
	const name = `E2E Cat Del ${Date.now()}`;

	await page.goto('/settings?tab=categories');
	await waitAppReady(page);
	await page.getByPlaceholder('Название категории').fill(name);
	await page.getByRole('button', { name: 'Создать' }).click();
	await expect(page.getByText(name)).toBeVisible({ timeout: 10_000 });

	const row = page.locator('.space-y-2 > .card').filter({ hasText: name }).first();
	await rowMenuAction(page, row, 'Удалить');
	await confirmDialog(page);
	await expect(page.getByText(name)).toHaveCount(0, { timeout: 10_000 });
});

test('categories: create income category on Доходы tab', async ({ page }) => {
	const name = `E2E Income ${Date.now()}`;

	await page.goto('/settings?tab=categories');
	await waitAppReady(page);
	await page.getByRole('button', { name: 'Доходы', exact: true }).click();
	await page.getByPlaceholder('Название категории').fill(name);
	await page.getByRole('button', { name: 'Создать' }).click();
	await expect(page.getByText(name)).toBeVisible({ timeout: 10_000 });
});

test('categories: make primary shows badge', async ({ page }) => {
	const name = `E2E Primary Cat ${Date.now()}`;

	await page.goto('/settings?tab=categories');
	await waitAppReady(page);
	await page.getByPlaceholder('Название категории').fill(name);
	await page.getByRole('button', { name: 'Создать' }).click();
	await expect(page.getByText(name)).toBeVisible({ timeout: 10_000 });

	const row = page.locator('.space-y-2 > .card').filter({ hasText: name }).first();
	await row.getByRole('button', { name: 'Действия' }).click();
	await page.getByRole('menuitem', { name: 'Сделать главной' }).click();
	await expect(row.getByLabel('Главная категория')).toBeVisible({ timeout: 10_000 });
});
