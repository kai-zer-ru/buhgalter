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
	await page.goto('/settings/password');
	await waitAppReady(page);

	await expect(page.locator('#old')).toBeVisible();
	await expect(page.locator('#new')).toBeVisible();
	await expect(page.locator('#confirm')).toBeVisible();
});

test('tokens: revoke created API token', async ({ page }) => {
	const tokenName = `E2E Revoke ${Date.now()}`;

	await page.goto('/settings/tokens');
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

test('tokens: perpetual token shows risk warning', async ({ page }) => {
	const tokenName = `E2E Perpetual ${Date.now()}`;

	await page.goto('/settings/tokens');
	await waitAppReady(page);
	await page.locator('#token-name').fill(tokenName);
	await page.getByRole('switch', { name: 'Бессрочный' }).click();
	await expect(page.getByText('Этот токен будет бессрочный — это РИСКОВАННО')).toBeVisible();
	await page.getByRole('button', { name: 'Создать' }).click();
	await page.getByRole('button', { name: 'Закрыть' }).click();
	await expect(page.getByText(tokenName).first()).toBeVisible();
	await expect(page.getByText('Бессрочно').first()).toBeVisible();
});

test('categories: delete expense category', async ({ page }) => {
	const name = `E2E Cat Del ${Date.now()}`;

	await page.goto('/settings/categories');
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

	await page.goto('/settings/categories');
	await waitAppReady(page);
	await page.getByRole('button', { name: 'Доходы', exact: true }).click();
	await page.getByPlaceholder('Название категории').fill(name);
	await page.getByRole('button', { name: 'Создать' }).click();
	await expect(page.getByText(name)).toBeVisible({ timeout: 10_000 });
});

test('categories: make primary shows badge', async ({ page }) => {
	const name = `E2E Primary Cat ${Date.now()}`;

	await page.goto('/settings/categories');
	await waitAppReady(page);
	await page.getByPlaceholder('Название категории').fill(name);
	await page.getByRole('button', { name: 'Создать' }).click();
	await expect(page.getByText(name)).toBeVisible({ timeout: 10_000 });

	const row = page.locator('.space-y-2 > .card').filter({ hasText: name }).first();
	await row.getByRole('button', { name: 'Действия' }).click();
	await page.getByRole('menuitem', { name: 'Сделать главной' }).click();
	await expect(row.getByLabel('Главная категория')).toBeVisible({ timeout: 10_000 });
});

test('categories: create and edit subcategory via dialog', async ({ page }) => {
	const catName = `E2E SubCat Parent ${Date.now()}`;
	const subName = `E2E Sub ${Date.now()}`;
	const subRenamed = `${subName} Renamed`;

	await page.goto('/settings/categories');
	await waitAppReady(page);
	await page.getByPlaceholder('Название категории').fill(catName);
	await page.getByRole('button', { name: 'Создать' }).click();
	await expect(page.getByText(catName)).toBeVisible({ timeout: 10_000 });

	const row = page.locator('.space-y-2 > .card').filter({ hasText: catName }).first();
	await row.getByRole('button', { name: catName }).click();
	const addSubBtn = row.getByRole('button', { name: 'Добавить подкатегорию' });
	await expect(addSubBtn).toBeVisible({ timeout: 10_000 });
	await addSubBtn.click();

	const dialog = page.getByRole('dialog', { name: 'Новая подкатегория' });
	await expect(dialog).toBeVisible({ timeout: 10_000 });
	await expect(dialog.getByLabel('Сменить иконку')).toBeVisible();
	await dialog.getByPlaceholder('Название подкатегории').fill(subName);
	await dialog.getByRole('button', { name: 'Сохранить' }).click();
	await expect(row.getByText(subName)).toBeVisible({ timeout: 10_000 });

	const subRow = row.locator('[data-drag-kind="sub"]').filter({ hasText: subName });
	await rowMenuAction(page, subRow, 'Редактировать');
	const editDialog = page.getByRole('dialog', { name: 'Редактировать подкатегорию' });
	await expect(editDialog).toBeVisible({ timeout: 10_000 });
	await editDialog.getByPlaceholder('Название подкатегории').fill(subRenamed);
	await editDialog.getByRole('button', { name: 'Сохранить' }).click();
	await expect(row.getByText(subRenamed)).toBeVisible({ timeout: 10_000 });
});
