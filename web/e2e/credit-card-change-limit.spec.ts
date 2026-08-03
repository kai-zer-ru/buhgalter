import { test, expect } from '@playwright/test';
import { apiJSON, waitAppReady } from './helpers/auth';
import { createCreditCardAccount } from './helpers/setup-data';
import { rowMenuAction } from './helpers/ui';

test.describe('credit card — change limit', () => {
	test('increase limit via menu and keep debt unchanged', async ({ page }) => {
		await page.setViewportSize({ width: 1280, height: 720 });
		const card = await createCreditCardAccount(page);
		const before = await apiJSON<{
			id: string;
			balance: number;
			credit_limit: number;
		}>(page, 'GET', `/api/v1/accounts/${card.id}`);
		const debtBefore = before.credit_limit - before.balance;

		await page.goto('/accounts');
		await waitAppReady(page);
		const cardEl = page.locator('.card').filter({ hasText: card.name }).first();
		await rowMenuAction(page, cardEl, 'Изменить лимит');

		const dialog = page.getByRole('dialog');
		await expect(dialog).toBeVisible();
		await expect(dialog.getByText('Текущий лимит')).toBeVisible();

		await dialog.locator('#cc-new-limit').fill('70000.00');
		await dialog.getByRole('button', { name: 'Сохранить' }).click();
		await expect(dialog).toBeHidden({ timeout: 10_000 });

		const after = await apiJSON<{
			balance: number;
			credit_limit: number;
		}>(page, 'GET', `/api/v1/accounts/${card.id}`);
		expect(after.credit_limit).toBe(7_000_000);
		expect(after.credit_limit - after.balance).toBe(debtBefore);
	});

	test('decrease blocked when card is not fully paid', async ({ page }) => {
		await page.setViewportSize({ width: 1280, height: 720 });
		const card = await createCreditCardAccount(page);

		await page.goto('/accounts');
		await waitAppReady(page);
		const cardEl = page.locator('.card').filter({ hasText: card.name }).first();
		await rowMenuAction(page, cardEl, 'Изменить лимит');

		const dialog = page.getByRole('dialog');
		await expect(dialog).toBeVisible();
		await dialog.locator('#cc-new-limit').fill('100.00');
		await expect(
			dialog.getByText('Уменьшить лимит можно только при полном погашении карты')
		).toBeVisible();
		await expect(dialog.getByRole('button', { name: 'Сохранить' })).toBeDisabled();
	});
});
