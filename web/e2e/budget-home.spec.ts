import { test, expect } from '@playwright/test';
import { waitAppReady } from './helpers/auth';

test('home: budget widget when budget exists', async ({ page }) => {
	await page.goto('/budget');
	await waitAppReady(page);
	await page.getByRole('button', { name: 'Добавить' }).click();
	await page.getByLabel('Название').fill('Home Widget Budget');
	await page.getByLabel('Лимит').fill('1000');
	await page.getByRole('button', { name: 'Сохранить' }).click();
	await expect(page.getByText('Home Widget Budget')).toBeVisible({ timeout: 10_000 });

	await page.goto('/');
	await waitAppReady(page);
	await expect(page.getByText('Бюджет месяца')).toBeVisible({ timeout: 10_000 });
	await expect(page.getByText('Home Widget Budget')).toBeVisible();
});
