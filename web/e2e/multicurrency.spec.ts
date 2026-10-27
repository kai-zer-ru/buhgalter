import { test, expect } from '@playwright/test';
import { waitAppReady } from './helpers/auth';
import { selectCombobox } from './helpers/combobox';
import { createCashAccount } from './helpers/setup-data';

test('create USD account from the currency selector', async ({ page }) => {
	const name = `E2E USD ${Date.now()}`;
	await page.goto('/accounts/new');
	await waitAppReady(page);

	await expect(page.locator('#account-currency')).toBeVisible();
	await page.getByLabel('Название').fill(name);
	await page.getByRole('button', { name: 'Наличные' }).click();
	await selectCombobox(page, 'account-currency', { label: 'USD $' });
	await page.getByLabel('Начальный баланс').fill('100');
	await page.getByRole('button', { name: 'Создать' }).click();
	await expect(page).toHaveURL(/\/accounts\/?$/, { timeout: 15_000 });

	const card = page.locator('.card').filter({ hasText: name }).first();
	await expect(card).toBeVisible({ timeout: 10_000 });
	await expect(card.getByText('100.00 $', { exact: true })).toBeVisible();
});

test('FX transfer requires to_amount and sends it', async ({ page }) => {
	const unique = Date.now();
	const rubName = `E2E RUB FX ${unique}`;
	const usdName = `E2E USD FX ${unique}`;
	await createCashAccount(page, rubName);

	await page.goto('/accounts/new');
	await waitAppReady(page);
	await page.getByLabel('Название').fill(usdName);
	await page.getByRole('button', { name: 'Наличные' }).click();
	await selectCombobox(page, 'account-currency', { label: 'USD $' });
	await page.getByLabel('Начальный баланс').fill('10');
	await page.getByRole('button', { name: 'Создать' }).click();
	await expect(page).toHaveURL(/\/accounts\/?$/, { timeout: 15_000 });

	await page.goto('/');
	await waitAppReady(page);
	await page.getByRole('button', { name: 'Перевод', exact: true }).click();
	await selectCombobox(page, 'from-acc', { label: rubName });
	await selectCombobox(page, 'to-acc', { label: usdName });
	await page.locator('#tr-amount').fill('100');
	await expect(page.locator('#tr-to-amount')).toBeVisible();
	await expect(page.getByRole('button', { name: 'Сохранить' })).toBeDisabled();

	const posted = page.waitForRequest(
		(req) => req.method() === 'POST' && req.url().includes('/api/v1/transfers')
	);
	await page.locator('#tr-to-amount').fill('1');
	await expect(page.getByRole('button', { name: 'Сохранить' })).toBeEnabled();
	await page.getByRole('button', { name: 'Сохранить' }).click();
	const req = await posted;
	expect(JSON.parse(req.postData() ?? '{}')).toMatchObject({
		amount: '100.00',
		to_amount: '1.00'
	});
	await expect(page.getByText('100.00').first()).toBeVisible({ timeout: 10_000 });
});
