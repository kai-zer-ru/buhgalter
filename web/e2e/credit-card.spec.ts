import { test, expect } from '@playwright/test';
import { waitAppReady } from './helpers/auth';
import { createCreditCardAccount } from './helpers/setup-data';
import { rowMenuAction } from './helpers/ui';

test('credit card: dashboard block, pay and charge fee', async ({ page }) => {
	const card = await createCreditCardAccount(page);

	await page.goto('/');
	await waitAppReady(page);
	await expect(page.getByText('Кредитный баланс')).toBeVisible({ timeout: 10_000 });

	await page.goto('/accounts');
	await waitAppReady(page);
	const cardEl = page.locator('.card').filter({ hasText: card.name }).first();
	await rowMenuAction(page, cardEl, 'Оплатить');

	const payDialog = page.getByRole('dialog');
	await expect(payDialog).toBeVisible();
	await payDialog.locator('#tr-amount').fill('100.00');
	await payDialog.getByRole('button', { name: 'Сохранить' }).click();
	await expect(payDialog).toBeHidden({ timeout: 10_000 });

	await rowMenuAction(page, cardEl, 'Списать комиссию');
	const feeDialog = page.getByRole('dialog');
	await expect(feeDialog).toBeVisible();
	await feeDialog.locator('#cc-fee-amount').fill('50.00');
	await feeDialog.getByRole('button', { name: 'Сохранить' }).click();
	await expect(feeDialog).toBeHidden({ timeout: 10_000 });
});
