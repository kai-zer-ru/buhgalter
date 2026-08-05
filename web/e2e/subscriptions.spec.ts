import { test, expect } from '@playwright/test';
import { waitAppReady } from './helpers/auth';

test('subscriptions page loads', async ({ page }) => {
	await page.goto('/subscriptions');
	await waitAppReady(page);
	await expect(page.getByRole('heading', { level: 1 })).toBeVisible();
	await expect(page.getByRole('button', { name: 'Добавить' })).toBeVisible();
});
