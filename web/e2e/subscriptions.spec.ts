import { test, expect } from '@playwright/test';
import { waitAppReady } from './helpers/auth';

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
