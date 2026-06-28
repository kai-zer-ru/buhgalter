import { test, expect, type Page } from '@playwright/test';
import { apiJSON, waitAppReady } from './helpers/auth';
import { fillTransactionForm } from './helpers/transactions';

async function openAccountPage(page: Page) {
	const account = await apiJSON<{ id: string; name: string }>(page, 'POST', '/api/v1/accounts', {
		name: `E2E Account Actions ${Date.now()}`,
		type: 'cash',
		initial_balance: '1000.00'
	});
	await page.goto(`/accounts/${account.id}`);
	await waitAppReady(page);
	return { account, header: page.locator('.card').first() };
}

test.describe('account page — operation entry points', () => {
	test('desktop: icon buttons in header, not in menu', async ({ page }) => {
		await page.setViewportSize({ width: 1280, height: 720 });
		const { header } = await openAccountPage(page);

		await expect(header.getByRole('button', { name: 'Доход', exact: true })).toBeVisible();
		await expect(header.getByRole('button', { name: 'Расход', exact: true })).toBeVisible();
		await expect(header.getByRole('button', { name: 'Перевод', exact: true })).toBeVisible();

		await header.getByRole('button', { name: 'Действия' }).click();
		await expect(page.getByRole('menuitem', { name: 'Доход' })).toHaveCount(0);
		await expect(page.getByRole('menuitem', { name: 'Расход' })).toHaveCount(0);
		await expect(page.getByRole('menuitem', { name: 'Перевод' })).toHaveCount(0);
		await expect(page.getByRole('menuitem', { name: 'Редактировать' })).toBeVisible();
	});

	test('mobile: icon buttons hidden, operations in header menu', async ({ page }) => {
		await page.setViewportSize({ width: 390, height: 844 });
		const { account, header } = await openAccountPage(page);

		await expect(header.getByRole('button', { name: 'Доход', exact: true })).toBeHidden();
		await expect(header.getByRole('button', { name: 'Расход', exact: true })).toBeHidden();
		await expect(header.getByRole('button', { name: 'Перевод', exact: true })).toBeHidden();

		await header.getByRole('button', { name: 'Действия' }).click();
		await expect(page.getByRole('menuitem', { name: 'Расход' })).toBeVisible();
		await page.getByRole('menuitem', { name: 'Расход' }).click();

		await fillTransactionForm(page, { amount: '50', account: account.name });
		await expect(page.getByText('50.00').first()).toBeVisible({ timeout: 10_000 });
	});
});
