import fs from 'node:fs';
import { test, expect, type Page } from '@playwright/test';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { selectCombobox } from './helpers/combobox';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const authFile = path.join(__dirname, '.auth', 'admin.json');

const ADMIN = {
	login: 'admin',
	password: 'secret123',
	displayName: 'E2E Admin'
};

test.describe.configure({ mode: 'serial' });

async function waitAppReady(page: Page) {
	await expect(page.getByText('Загрузка…')).toHaveCount(0, { timeout: 20_000 });
	await expect(page.locator('header')).toBeVisible({ timeout: 20_000 });
}

async function login(page: Page) {
	await page.goto('/login');
	await page.locator('#login').fill(ADMIN.login);
	await page.locator('#password').fill(ADMIN.password);
	await page.getByRole('button', { name: 'Войти' }).click();
	await waitAppReady(page);
}

async function completeSetupIfNeeded(page: Page) {
	await page.goto('/setup');
	await expect(page.getByText('Загрузка…')).toHaveCount(0, { timeout: 20_000 });

	const displayName = page.locator('#display-name');
	if (!(await displayName.isVisible())) {
		return;
	}

	await displayName.fill(ADMIN.displayName);
	await page.locator('#login').fill(ADMIN.login);
	await page.locator('#password').fill(ADMIN.password);
	await page.locator('#password-confirm').fill(ADMIN.password);
	await page.getByRole('button', { name: 'Завершить настройку' }).click();
	await page.waitForURL('**/login**', { timeout: 15_000 });
}

test('setup → login', async ({ page }) => {
	await completeSetupIfNeeded(page);
	await login(page);
	fs.mkdirSync(path.dirname(authFile), { recursive: true });
	await page.context().storageState({ path: authFile });
});

test('cold start at / redirects to login', async ({ page }) => {
	await page.context().clearCookies();
	await page.goto('/');
	await expect(page.getByText('Загрузка…')).toHaveCount(0, { timeout: 20_000 });
	await expect(page).toHaveURL(/\/login/);
	await expect(page.getByRole('button', { name: 'Войти' })).toBeVisible();
});

test.describe('authenticated', () => {
	test.use({ storageState: authFile });

	test('create account → add expense → see balance', async ({ page }) => {
		await page.goto('/accounts/new');
		await waitAppReady(page);
		await page.getByLabel('Название').fill('E2E Cash');
		await page.getByRole('button', { name: 'Наличные' }).click();
		await page.getByLabel('Начальный баланс').fill('1000');
		await page.getByRole('button', { name: 'Создать' }).click();
		await expect(page).toHaveURL(/\/accounts\//);

		await page.goto('/transactions');
		await waitAppReady(page);
		await page.getByRole('button', { name: /Операция/ }).click();
		await page.getByRole('button', { name: 'Расход' }).click();
		await page.getByLabel('Сумма').fill('250');
		await selectCombobox(page, 'tx-account', { label: 'E2E Cash' });
		await selectCombobox(page, 'tx-category', { index: 0 });
		await page.getByRole('button', { name: 'Сохранить' }).click();
		await expect(page.getByRole('cell', { name: '250.00' })).toBeVisible({ timeout: 10_000 });

		await page.goto('/settings?tab=accounts');
		await waitAppReady(page);
		await expect(page.getByText(/750/)).toBeVisible({ timeout: 10_000 });
	});

	test('create transfer', async ({ page }) => {
		await page.goto('/accounts/new');
		await waitAppReady(page);
		await page.getByLabel('Название').fill('E2E Bank');
		await page.getByRole('button', { name: 'Банковский' }).click();
		await page.getByPlaceholder('Поиск банка…').fill('Сбер');
		await page
			.getByRole('button', { name: /Сбербанк/ })
			.first()
			.click();
		await page.getByLabel('Начальный баланс').fill('500');
		await page.getByRole('button', { name: 'Создать' }).click();
		await expect(page).toHaveURL(/\/accounts\//);

		await page.goto('/');
		await waitAppReady(page);
		await page.getByRole('button', { name: 'Перевод' }).click();
		await selectCombobox(page, 'from-acc', { label: 'E2E Cash' });
		await selectCombobox(page, 'to-acc', { label: 'E2E Bank' });
		await page.locator('#tr-amount').fill('100');
		await page.getByRole('button', { name: 'Сохранить' }).click();
		await expect(page.getByRole('cell', { name: '100.00' })).toBeVisible({ timeout: 10_000 });
	});

	test('import csv preview', async ({ page }) => {
		await page.goto('/settings?tab=import');
		await waitAppReady(page);
		const fileInput = page.locator('input[type="file"]');
		await fileInput.setInputFiles(path.join(__dirname, 'fixtures', 'sample.csv'));

		await expect(page.getByText('sample.csv')).toBeVisible({ timeout: 10_000 });
		await page.getByRole('button', { name: 'Далее' }).click();
		await expect(page.getByText(/Сопоставление счетов|Всего строк|Готово к импорту/)).toBeVisible({
			timeout: 20_000
		});
	});

	test('settings change theme', async ({ page }) => {
		await page.goto('/settings');
		await waitAppReady(page);
		await selectCombobox(page, 'theme', { label: 'Тёмная' });
		await page.getByRole('button', { name: 'Сохранить' }).click();

		await expect(page.locator('html')).toHaveClass(/dark/, { timeout: 10_000 });
	});

	test('admin create user', async ({ page }) => {
		await page.goto('/admin/users');
		await waitAppReady(page);
		await page.getByLabel('Логин').fill('e2euser');
		await page.getByLabel('Пароль', { exact: true }).fill('userpass1');
		await page.getByLabel('Подтверждение пароля').fill('userpass1');
		await page.getByRole('button', { name: 'Создать' }).click();

		await expect(page.getByRole('cell', { name: 'e2euser' }).first()).toBeVisible({
			timeout: 10_000
		});
	});
});
