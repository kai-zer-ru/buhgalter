import { test, expect } from '@playwright/test';
import { waitAppReady } from './helpers/auth';
import { createCashAccount, createExpense, createIncome } from './helpers/setup-data';
import { selectCombobox } from './helpers/transactions';

test('filter transactions by type expense', async ({ page }) => {
	const account = await createCashAccount(page);
	await createExpense(page, account.id, '10.00', 'E2E filter expense');
	await createIncome(page, account.id, '20.00', 'E2E filter income');

	await page.goto('/transactions');
	await waitAppReady(page);

	await selectCombobox(page, 'tx-filter-type', { label: 'Расход' });
	await expect(page).toHaveURL(/type=expense/, { timeout: 10_000 });
	await expect(page.getByRole('row', { name: /10\.00/ })).toBeVisible({ timeout: 10_000 });
	await expect(page.getByRole('row', { name: /20\.00/ })).toHaveCount(0, { timeout: 10_000 });
});

test('search by description narrows results', async ({ page }) => {
	const account = await createCashAccount(page);
	const unique = `E2E search ${Date.now()}`;
	await createExpense(page, account.id, '15.00', unique);
	await createExpense(page, account.id, '16.00', 'E2E other row');

	await page.goto('/transactions');
	await waitAppReady(page);

	await page.locator('.filter-panel-body input.input.w-full').last().fill(unique);
	await expect(page).toHaveURL(/search=/, { timeout: 10_000 });
	await expect(page.getByRole('row', { name: /15\.00/ })).toBeVisible({ timeout: 10_000 });
	await expect(page.getByRole('row', { name: /16\.00/ })).toHaveCount(0, { timeout: 10_000 });
});

test('reset filters clears URL params', async ({ page }) => {
	await page.goto('/transactions?type=expense&search=test');
	await waitAppReady(page);

	await page.getByRole('button', { name: 'Сбросить' }).click();
	await expect(page).not.toHaveURL(/type=expense/, { timeout: 10_000 });
	await expect(page).not.toHaveURL(/search=/, { timeout: 10_000 });
});

test('filter by account shows only matching rows', async ({ page }) => {
	const accA = await createCashAccount(page, 'E2E Filter A');
	const accB = await createCashAccount(page, 'E2E Filter B');
	await createExpense(page, accA.id, '21.00', 'E2E on A');
	await createExpense(page, accB.id, '22.00', 'E2E on B');

	await page.goto('/transactions');
	await waitAppReady(page);

	await selectCombobox(page, 'tx-filter-account', { label: accA.name });
	await expect(page).toHaveURL(/account_id=/, { timeout: 10_000 });
	await expect(page.getByRole('row', { name: /21\.00/ })).toBeVisible({ timeout: 10_000 });
	await expect(page.getByRole('row', { name: /22\.00/ })).toHaveCount(0, { timeout: 10_000 });
});

test('mobile filters spoiler expands fields', async ({ page }) => {
	await page.setViewportSize({ width: 390, height: 844 });
	await page.goto('/transactions');
	await waitAppReady(page);

	const summary = page.locator('.filter-panel-summary');
	await expect(summary).toBeVisible();
	await summary.click();
	await expect(page.locator('#tx-filter-type')).toBeVisible();
});
