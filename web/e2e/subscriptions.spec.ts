import { test, expect } from '@playwright/test';
import { apiJSON, waitAppReady } from './helpers/auth';
import { createCashAccount } from './helpers/setup-data';

test('subscription notify deep link opens inline edit', async ({ page }) => {
	const account = await createCashAccount(page);
	const name = `E2E Notify Sub ${Date.now()}`;
	const sub = await apiJSON<{ id: string; name: string }>(page, 'POST', '/api/v1/subscriptions', {
		name,
		amount: '99.00',
		account_id: account.id,
		period: 'month',
		start_date: '2026-05-04 08:00:00',
		active: true
	});

	let deepLinkLoads = 0;
	page.on('request', (req) => {
		const path = new URL(req.url()).pathname;
		if (path === `/subscriptions/${sub.id}`) deepLinkLoads += 1;
	});

	await page.goto(`/subscriptions/${sub.id}`);
	await waitAppReady(page);
	await expect(page).toHaveURL(/\/subscriptions$/);
	await expect(page.getByRole('textbox', { name: 'Название' })).toHaveValue(name);
	expect(deepLinkLoads).toBeLessThanOrEqual(2);
});

test('subscriptions page loads', async ({ page }) => {
	await page.goto('/subscriptions');
	await waitAppReady(page);
	await expect(page.getByRole('heading', { level: 1 })).toBeVisible();
	await expect(page.getByRole('button', { name: 'Добавить' })).toBeVisible();
});

test('subscription form shows three upcoming charge dates', async ({ page }) => {
	await page.goto('/subscriptions');
	await waitAppReady(page);
	await page.getByRole('button', { name: 'Добавить' }).click();
	await expect(page.getByText('Ближайшие списания')).toBeVisible();
	await expect(page.getByRole('button', { name: 'Сбросить по периоду' })).toBeVisible();
	await expect(page.getByText('Списание 1')).toBeVisible();
	await expect(page.getByText('Списание 2')).toBeVisible();
	await expect(page.getByText('Списание 3')).toBeVisible();
});

test('subscription date picker can navigate months', async ({ page }) => {
	await page.goto('/subscriptions');
	await waitAppReady(page);
	await page.getByRole('button', { name: 'Добавить' }).click();

	const startDatePicker = page.locator('#subscription-start-date-create');
	await startDatePicker.click();

	const panel = page.locator('.popover-panel').last();
	const monthHeading = panel.locator('button.text-sm.font-medium.capitalize').first();
	const initialMonth = (await monthHeading.textContent())?.trim();
	expect(initialMonth).toBeTruthy();

	await panel.locator('button').filter({ hasText: '‹' }).first().click();
	await expect(monthHeading).not.toHaveText(initialMonth ?? '');
});
