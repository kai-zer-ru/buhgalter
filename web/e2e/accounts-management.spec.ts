import { test, expect } from '@playwright/test';
import { waitAppReady } from './helpers/auth';
import { createCashAccount, createTransfer } from './helpers/setup-data';
import { confirmDialog, rowMenuAction } from './helpers/ui';

test('accounts list: edit name inline', async ({ page }) => {
	const unique = Date.now();
	const account = await createCashAccount(page, `E2E Rename Me ${unique}`);
	const newName = `E2E Renamed ${unique}`;

	await page.goto('/accounts');
	await waitAppReady(page);

	const card = page.locator('.card').filter({ hasText: account.name }).first();
	await rowMenuAction(page, card, 'Редактировать');
	const nameInput = page.locator(`#edit-name-${account.id}`);
	await nameInput.fill(newName);
	await nameInput
		.locator('xpath=ancestor::form[1]')
		.getByRole('button', { name: 'Сохранить' })
		.click();

	await expect(page.getByText(newName)).toBeVisible({ timeout: 10_000 });
});

test('accounts list: make primary account', async ({ page }) => {
	const unique = Date.now();
	const primary = await createCashAccount(page, `E2E Primary Target ${unique}`);
	await createCashAccount(page, `E2E Non Primary ${unique}`);

	await page.goto('/accounts');
	await waitAppReady(page);

	const card = page.locator('.card').filter({ hasText: primary.name });
	await rowMenuAction(page, card, 'Сделать основным');
	await expect(card.getByLabel('Основной счёт')).toBeVisible({ timeout: 10_000 });
});

test('accounts list: archive removes card', async ({ page }) => {
	const account = await createCashAccount(page, `E2E Archive Me ${Date.now()}`);

	await page.goto('/accounts');
	await waitAppReady(page);

	const card = page.locator('.card').filter({ hasText: account.name });
	await rowMenuAction(page, card, 'Архивировать');
	await expect(page.getByText(account.name)).toHaveCount(0, { timeout: 10_000 });
});

test('account detail: edit via header menu', async ({ page }) => {
	const unique = Date.now();
	const account = await createCashAccount(page, `E2E Detail Edit ${unique}`);
	const newName = `E2E Detail Renamed ${unique}`;

	await page.goto(`/accounts/${account.id}`);
	await waitAppReady(page);

	const header = page.locator('.card').first();
	await header.getByRole('button', { name: 'Действия' }).click();
	await page.getByRole('menuitem', { name: 'Редактировать' }).click();
	await header.locator('input').first().fill(newName);
	await header.getByRole('button', { name: 'Сохранить' }).click();

	await expect(page.getByRole('heading', { name: newName })).toBeVisible({ timeout: 10_000 });
});

test('account detail: delete account redirects to list', async ({ page }) => {
	const account = await createCashAccount(page, `E2E Delete Me ${Date.now()}`);

	await page.goto(`/accounts/${account.id}`);
	await waitAppReady(page);

	const header = page.locator('.card').first();
	await header.getByRole('button', { name: 'Действия' }).click();
	await page.getByRole('menuitem', { name: 'Удалить' }).click();
	await confirmDialog(page);

	await expect(page).toHaveURL(/\/accounts\/?$/, { timeout: 15_000 });
	await expect(page.getByText(account.name)).toHaveCount(0);
});

test('dashboard: click account card opens account page', async ({ page }) => {
	const account = await createCashAccount(page, `E2E Dashboard Nav ${Date.now()}`);

	await page.goto('/');
	await waitAppReady(page);
	await page.getByRole('link', { name: new RegExp(account.name) }).click();
	await waitAppReady(page);

	await expect(page).toHaveURL(new RegExp(`/accounts/${account.id}`));
	await expect(page.getByRole('heading', { name: account.name })).toBeVisible();
});

test('account detail: transfer counterparty links open correct account', async ({ page }) => {
	const from = await createCashAccount(page, `E2E Xfer From ${Date.now()}`);
	const to = await createCashAccount(page, `E2E Xfer To ${Date.now()}`);
	const amount = '75.00';
	await createTransfer(page, from.id, to.id, amount);

	await page.goto(`/accounts/${to.id}`);
	await waitAppReady(page);
	const incomingRow = page.getByRole('row').filter({ hasText: amount });
	await incomingRow.getByRole('link', { name: from.name, exact: true }).click();
	await waitAppReady(page);
	await expect(page).toHaveURL(new RegExp(`/accounts/${from.id}(?:\\?|$)`));
	await expect(page.getByRole('heading', { name: from.name })).toBeVisible();

	await page.goto(`/accounts/${from.id}`);
	await waitAppReady(page);
	const outgoingRow = page.getByRole('row').filter({ hasText: amount });
	await outgoingRow.getByRole('link', { name: to.name, exact: true }).click();
	await waitAppReady(page);
	await expect(page).toHaveURL(new RegExp(`/accounts/${to.id}(?:\\?|$)`));
	await expect(page.getByRole('heading', { name: to.name })).toBeVisible();
});
