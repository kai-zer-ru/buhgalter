import { test, expect } from '@playwright/test';
import { waitAppReady } from './helpers/auth';
import { expandCollapsibleSection } from './helpers/ui';

test('mobile menu drill-down opens settings submenu', async ({ page }) => {
	await page.setViewportSize({ width: 390, height: 844 });
	await page.goto('/');
	await waitAppReady(page);

	await page.getByRole('button', { name: 'Меню' }).click();
	await expect(page.locator('.nav-mobile-panel')).toBeVisible();
	await page.locator('.nav-mobile-panel').getByRole('menuitem', { name: 'Настройки' }).click();
	await expect(
		page.locator('.nav-mobile-panel').getByRole('menuitem', { name: 'Операции' })
	).toBeHidden();
	await expect(
		page.locator('.nav-mobile-panel').getByRole('menuitem', { name: 'Пароль' })
	).toBeVisible();

	await page.locator('.nav-mobile-panel').getByRole('button', { name: 'Назад' }).click();
	await expect(
		page.locator('.nav-mobile-panel').getByRole('menuitem', { name: 'Операции' })
	).toBeVisible();
});

test('desktop nav starts with home and opens dashboard', async ({ page }) => {
	await page.setViewportSize({ width: 1280, height: 720 });
	await page.goto('/accounts');
	await waitAppReady(page);

	const desktopNav = page.locator('header nav.hidden');
	const firstLink = desktopNav.getByRole('link').first();
	await expect(firstLink).toHaveText('Главная');
	await firstLink.click();
	await waitAppReady(page);

	await expect(page).toHaveURL('/');
	await expect(firstLink).toHaveClass(/nav-link-active/);
});

test('mobile menu starts with home and opens dashboard', async ({ page }) => {
	await page.setViewportSize({ width: 390, height: 844 });
	await page.goto('/accounts');
	await waitAppReady(page);

	await page.getByRole('button', { name: 'Меню' }).click();
	const panel = page.locator('.nav-mobile-panel');
	await expect(panel).toBeVisible();

	const firstItem = panel.getByRole('menuitem').first();
	await expect(firstItem).toHaveText('Главная');
	await firstItem.click();
	await waitAppReady(page);

	await expect(page).toHaveURL('/');
});

test('mobile menu navigates to stats', async ({ page }) => {
	await page.setViewportSize({ width: 390, height: 844 });
	await page.goto('/');
	await waitAppReady(page);

	await page.getByRole('button', { name: 'Меню' }).click();
	await expect(page.locator('.nav-mobile-panel')).toBeVisible();
	await page.locator('.nav-mobile-panel').getByRole('menuitem', { name: 'Статистика' }).click();
	await waitAppReady(page);

	await expect(page).toHaveURL(/\/stats/);
	await expect(page.getByRole('heading', { name: 'Статистика', level: 1 })).toBeVisible();
});

test('mobile menu navigates to transactions', async ({ page }) => {
	await page.setViewportSize({ width: 390, height: 844 });
	await page.goto('/');
	await waitAppReady(page);

	await page.getByRole('button', { name: 'Меню' }).click();
	await expect(page.locator('.nav-mobile-panel')).toBeVisible();
	await page.locator('.nav-mobile-panel').getByRole('menuitem', { name: 'Операции' }).click();
	await waitAppReady(page);

	await expect(page).toHaveURL(/\/transactions/);
	await expect(page.getByRole('heading', { name: 'Все операции', level: 1 })).toBeVisible();
});

test('dashboard link opens all transactions', async ({ page }) => {
	await page.goto('/');
	await waitAppReady(page);
	await expandCollapsibleSection(page, 'Последние операции');
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
	await page.goto('/settings/password');
	await waitAppReady(page);

	const crumbs = page.locator('.breadcrumbs');
	await expect(crumbs.getByRole('link', { name: 'Главная' })).toBeVisible();
	await expect(crumbs.getByRole('link', { name: 'Настройки' })).toBeVisible();
	await expect(crumbs.getByText('Пароль', { exact: true })).toBeVisible();
});

test('settings dropdown navigates to password', async ({ page }) => {
	await page.setViewportSize({ width: 1280, height: 720 });
	await page.goto('/');
	await waitAppReady(page);

	await page.getByRole('button', { name: 'Настройки' }).click();
	await page.getByRole('menuitem', { name: 'Пароль' }).click();
	await waitAppReady(page);

	await expect(page).toHaveURL(/\/settings\/password/);
	await expect(page.getByRole('heading', { name: 'Пароль', level: 1 })).toBeVisible();
});

test('other dropdown navigates to accounts, debts, credits, recurring and subscriptions', async ({
	page
}) => {
	await page.setViewportSize({ width: 1280, height: 720 });
	await page.goto('/');
	await waitAppReady(page);

	await page.getByRole('button', { name: 'Прочее' }).click();
	await page.getByRole('menuitem', { name: 'Счета' }).click();
	await waitAppReady(page);
	await expect(page).toHaveURL(/\/accounts/);

	await page.getByRole('button', { name: 'Прочее' }).click();
	await page.getByRole('menuitem', { name: 'Долги' }).click();
	await waitAppReady(page);
	await expect(page).toHaveURL(/\/debts/);

	await page.getByRole('button', { name: 'Прочее' }).click();
	await page.getByRole('menuitem', { name: 'Кредиты' }).click();
	await waitAppReady(page);
	await expect(page).toHaveURL(/\/credits/);

	await page.getByRole('button', { name: 'Прочее' }).click();
	await page.getByRole('menuitem', { name: 'Периодические операции' }).click();
	await waitAppReady(page);
	await expect(page).toHaveURL(/\/recurring-operations/);

	await page.getByRole('button', { name: 'Прочее' }).click();
	await page.getByRole('menuitem', { name: 'Подписки' }).click();
	await waitAppReady(page);
	await expect(page).toHaveURL(/\/subscriptions/);

	await page.getByRole('button', { name: 'Прочее' }).click();
	await page.getByRole('menuitem', { name: 'Шаблоны операций' }).click();
	await waitAppReady(page);
	await expect(page).toHaveURL(/\/transaction-templates/);
});

test('categories is under other menu with home breadcrumb', async ({ page }) => {
	await page.setViewportSize({ width: 1280, height: 720 });
	await page.goto('/');
	await waitAppReady(page);

	await page.getByRole('button', { name: 'Прочее' }).click();
	await page.getByRole('menuitem', { name: 'Категории' }).click();
	await waitAppReady(page);
	await expect(page).toHaveURL(/\/categories/);

	const crumbs = page.locator('.breadcrumbs');
	await expect(crumbs.getByRole('link', { name: 'Главная' })).toBeVisible();
	await expect(crumbs.getByText('Категории', { exact: true })).toBeVisible();
	await expect(crumbs.getByRole('link', { name: 'Настройки' })).toHaveCount(0);
});

test('transaction templates is under other menu with home breadcrumb', async ({ page }) => {
	await page.setViewportSize({ width: 1280, height: 720 });
	await page.goto('/');
	await waitAppReady(page);

	await page.getByRole('button', { name: 'Прочее' }).click();
	await page.getByRole('menuitem', { name: 'Шаблоны операций' }).click();
	await waitAppReady(page);
	await expect(page).toHaveURL(/\/transaction-templates/);

	const crumbs = page.locator('.breadcrumbs');
	await expect(crumbs.getByRole('link', { name: 'Главная' })).toBeVisible();
	await expect(crumbs.getByText('Шаблоны операций', { exact: true })).toBeVisible();
	await expect(crumbs.getByRole('link', { name: 'Настройки' })).toHaveCount(0);
});

test('mobile menu drill-down opens other submenu', async ({ page }) => {
	await page.setViewportSize({ width: 390, height: 844 });
	await page.goto('/');
	await waitAppReady(page);

	await page.getByRole('button', { name: 'Меню' }).click();
	const panel = page.locator('.nav-mobile-panel');
	await expect(panel).toBeVisible();
	await panel.getByRole('menuitem', { name: 'Прочее' }).click();
	await expect(panel.getByRole('menuitem', { name: 'Счета' })).toBeVisible();
	await expect(panel.getByRole('menuitem', { name: 'Долги' })).toBeVisible();
	await expect(panel.getByRole('menuitem', { name: 'Кредиты' })).toBeVisible();
	await expect(panel.getByRole('menuitem', { name: 'Категории' })).toBeVisible();
	await expect(panel.getByRole('menuitem', { name: 'Подписки' })).toBeVisible();
	await expect(panel.getByRole('menuitem', { name: 'Периодические операции' })).toBeVisible();
	await expect(panel.getByRole('menuitem', { name: 'Шаблоны операций' })).toBeVisible();
	await panel.getByRole('menuitem', { name: 'Периодические операции' }).click();
	await waitAppReady(page);
	await expect(page).toHaveURL(/\/recurring-operations/);
});

test('nav links highlight active section', async ({ page }) => {
	await page.setViewportSize({ width: 1280, height: 720 });
	await page.goto('/accounts');
	await waitAppReady(page);
	await expect(page.getByRole('button', { name: 'Прочее' })).toHaveClass(/nav-link-active/);

	await page.goto('/debts');
	await waitAppReady(page);
	await expect(page.getByRole('button', { name: 'Прочее' })).toHaveClass(/nav-link-active/);

	await page.goto('/transactions');
	await waitAppReady(page);
	await expect(
		page.locator('header nav.hidden').getByRole('link', { name: 'Операции', exact: true })
	).toHaveClass(/nav-link-active/);
});

test('logout returns to login page', async ({ page }) => {
	await page.goto('/');
	await waitAppReady(page);
	await page.getByRole('button', { name: 'Выйти' }).click();
	await expect(page).toHaveURL(/\/login/, { timeout: 15_000 });
});
