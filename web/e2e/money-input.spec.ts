import { test, expect } from '@playwright/test';
import { waitAppReady } from './helpers/auth';
import { createCashAccount } from './helpers/setup-data';

async function expectEmptyMoneyInput(
	locator: ReturnType<typeof import('@playwright/test').Page.prototype.locator>
) {
	await expect(locator).toHaveValue('');
	await expect(locator).toHaveAttribute('placeholder', '0.00');
}

test('new account balance field shows placeholder instead of 0.00', async ({ page }) => {
	await page.goto('/accounts/new');
	await waitAppReady(page);
	await expectEmptyMoneyInput(page.locator('#balance'));
});

test('recurring operation form amount field starts empty', async ({ page }) => {
	await createCashAccount(page);
	await page.goto('/settings/recurring-operations');
	await waitAppReady(page);
	await page.getByRole('button', { name: 'Добавить' }).click();
	await expectEmptyMoneyInput(page.locator('#recurring-amount-create'));
});

test('expense dialog amount field starts empty', async ({ page }) => {
	await createCashAccount(page);
	await page.goto('/');
	await waitAppReady(page);
	await page.getByRole('button', { name: 'Расход' }).click();
	const dialog = page.getByRole('dialog');
	await expect(dialog).toBeVisible();
	await expectEmptyMoneyInput(dialog.locator('#tx-amount'));
});

test('transfer dialog amount and commission fields start empty', async ({ page }) => {
	await createCashAccount(page);
	await page.goto('/');
	await waitAppReady(page);
	await page.getByRole('button', { name: 'Перевод' }).click();
	const dialog = page.getByRole('dialog');
	await expect(dialog).toBeVisible();
	await expectEmptyMoneyInput(dialog.locator('#tr-amount'));
	await expectEmptyMoneyInput(dialog.locator('#tr-commission'));
});

test('typing zero in money field clears on blur', async ({ page }) => {
	await page.goto('/accounts/new');
	await waitAppReady(page);
	const balance = page.locator('#balance');
	await balance.fill('0');
	await balance.blur();
	await expect(balance).toHaveValue('');
});

test('create account with empty balance defaults to zero', async ({ page }) => {
	const name = `E2E Zero Balance ${Date.now()}`;
	await page.goto('/accounts/new');
	await waitAppReady(page);
	await page.locator('#name').fill(name);
	await page.getByRole('button', { name: 'Создать' }).click();
	await expect(page).toHaveURL(/\/accounts\//, { timeout: 15_000 });
	await expect(page.getByText('0.00').first()).toBeVisible();
});
