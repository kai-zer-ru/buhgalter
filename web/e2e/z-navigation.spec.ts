import { test, expect } from '@playwright/test';
import { waitAppReady } from './helpers/auth';

test('mobile menu navigates to stats', async ({ page }) => {
	await page.setViewportSize({ width: 390, height: 844 });
	await page.goto('/');
	await waitAppReady(page);

	await page.getByRole('button', { name: 'Меню' }).click();
	await page.locator('.nav-mobile-panel').getByRole('link', { name: 'Статистика' }).click();
	await waitAppReady(page);

	await expect(page).toHaveURL(/\/stats/);
	await expect(page.getByRole('heading', { name: 'Статистика', level: 1 })).toBeVisible();
});

test('dashboard link opens all transactions', async ({ page }) => {
	await page.goto('/');
	await waitAppReady(page);
	await page.getByRole('link', { name: 'Все операции', exact: true }).click();
	await waitAppReady(page);

	await expect(page).toHaveURL(/\/transactions/);
	await expect(page.getByRole('heading', { name: 'Все операции', level: 1 })).toBeVisible();
});

test('transactions breadcrumb returns to home', async ({ page }) => {
	await page.goto('/transactions');
	await waitAppReady(page);
	await page.locator('.breadcrumbs').getByRole('link', { name: 'Главная' }).click();
	await waitAppReady(page);

	await expect(page).toHaveURL('/');
});

test('accounts breadcrumb shows section trail', async ({ page }) => {
	await page.goto('/accounts');
	await waitAppReady(page);

	const crumbs = page.locator('.breadcrumbs');
	await expect(crumbs.getByRole('link', { name: 'Главная' })).toBeVisible();
	await expect(crumbs.getByText('Счета', { exact: true })).toBeVisible();
});

test('settings breadcrumb reflects active tab', async ({ page }) => {
	await page.goto('/settings?tab=password');
	await waitAppReady(page);

	const crumbs = page.locator('.breadcrumbs');
	await expect(crumbs.getByRole('link', { name: 'Главная' })).toBeVisible();
	await expect(crumbs.getByRole('link', { name: 'Настройки' })).toBeVisible();
	await expect(crumbs.getByText('Пароль', { exact: true })).toBeVisible();
});

test('nav links highlight active section', async ({ page }) => {
	await page.setViewportSize({ width: 1280, height: 720 });
	await page.goto('/accounts');
	await waitAppReady(page);
	await expect(
		page.locator('header nav.hidden').getByRole('link', { name: 'Счета', exact: true })
	).toHaveClass(/nav-link-active/);

	await page.goto('/debts');
	await waitAppReady(page);
	await expect(
		page.locator('header nav.hidden').getByRole('link', { name: 'Долги', exact: true })
	).toHaveClass(/nav-link-active/);
});

test('logout returns to login page', async ({ page }) => {
	await page.goto('/');
	await waitAppReady(page);
	await page.getByRole('button', { name: 'Выйти' }).click();
	await expect(page).toHaveURL(/\/login/, { timeout: 15_000 });
});
