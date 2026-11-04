import { test, expect } from '@playwright/test';
import { apiJSON, waitAppReady } from './helpers/auth';
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

test('credit card: pay more than remaining limit', async ({ page }) => {
	await page.setViewportSize({ width: 1280, height: 720 });
	const unique = Date.now();
	const debit = await apiJSON<{ id: string }>(page, 'POST', '/api/v1/accounts', {
		name: `E2E Debit overpay ${unique}`,
		type: 'cash',
		initial_balance: '200000.00'
	});
	const banks = await apiJSON<{ id: string; name: string }[]>(page, 'GET', '/api/v1/banks');
	const bank =
		banks.find((b) => b.name.includes('Тинькофф') || b.name.includes('Т-Банк')) ?? banks[0];
	const card = await apiJSON<{ id: string; name: string }>(page, 'POST', '/api/v1/accounts', {
		name: `E2E Credit overpay ${unique}`,
		type: 'credit_card',
		bank_id: bank.id,
		credit_limit: '130000.00',
		initial_balance: '0.00',
		payment_account_id: debit.id
	});

	await page.goto('/accounts');
	await waitAppReady(page);
	const cardEl = page.locator('.card').filter({ hasText: card.name }).first();
	await rowMenuAction(page, cardEl, 'Оплатить');

	const payDialog = page.getByRole('dialog');
	await expect(payDialog).toBeVisible();
	await payDialog.locator('#tr-amount').fill('140000.00');
	await expect(payDialog.getByRole('button', { name: 'Сохранить' })).toBeEnabled();
	await payDialog.getByRole('button', { name: 'Сохранить' }).click();
	await expect(payDialog).toBeHidden({ timeout: 10_000 });

	const after = await apiJSON<{ balance: number }>(page, 'GET', `/api/v1/accounts/${card.id}`);
	expect(after.balance).toBe(14_000_000);
	await expect(cardEl).toContainText(/140[\s\u00a0\u202f]?000/);
});
