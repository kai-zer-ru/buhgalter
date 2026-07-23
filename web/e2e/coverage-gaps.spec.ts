import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { test, expect } from '@playwright/test';
import { login, apiJSON, restoreAdminSession, waitAppReady } from './helpers/auth';
import { advanceImportToPreview, commitImportFromPreview } from './helpers/import';
import { createCashAccount, createIncome } from './helpers/setup-data';
import { confirmDialog, expectToast, expandCollapsibleSection, rowMenuAction } from './helpers/ui';
import {
	fillEditTxAmount,
	fillTransactionForm,
	selectCombobox,
	selectLabeledCombobox
} from './helpers/transactions';

test('login with valid credentials', async ({ page }) => {
	await page.context().clearCookies();
	await login(page);
	await expect(page).toHaveURL(/\/(\?.*)?$/);
	await expect(page.locator('header')).toBeVisible();
});

test('setup redirects when already configured', async ({ page }) => {
	await page.context().clearCookies();
	await page.goto('/setup');
	await expect(page.getByText('Загрузка…')).toHaveCount(0, { timeout: 20_000 });
	await expect(page).not.toHaveURL(/\/setup\/?$/);
});

test('register creates account when registration is enabled', async ({ page }) => {
	const tag = Date.now();
	await restoreAdminSession(page);
	await apiJSON(page, 'PUT', '/api/v1/admin/settings', {
		registration_enabled: true,
		external_url: ''
	});

	await page.context().clearCookies();
	await page.goto('/register');
	await expect(page.getByRole('heading', { name: 'Регистрация' })).toBeVisible();

	const loginName = `e2ereg${tag}`;
	await page.locator('#login').fill(loginName);
	await page.locator('#display').fill(`E2E Reg ${tag}`);
	await page.locator('#password').fill('Regpass1');
	await page.locator('#password-confirm').fill('Regpass1');
	await page.getByRole('button', { name: 'Создать аккаунт' }).click();

	await expect(page).toHaveURL(/\/login/, { timeout: 10_000 });

	await restoreAdminSession(page);
	const users = await apiJSON<{ id: string; login: string; status: string }[]>(
		page,
		'GET',
		'/api/v1/admin/users'
	);
	const created = users.find((u) => u.login === loginName);
	expect(created).toBeTruthy();
	await apiJSON(page, 'PUT', `/api/v1/admin/users/${created!.id}/status`, { status: 'active' });

	await page.context().clearCookies();
	await page.goto('/login');
	await page.locator('#login').fill(loginName);
	await page.locator('#password').fill('Regpass1');
	await page.getByRole('button', { name: 'Войти' }).click();

	await waitAppReady(page);
	await expect(page).toHaveURL(/\/(\?.*)?$/);
	await expect(page.getByRole('button', { name: 'Выйти' })).toBeVisible({ timeout: 10_000 });
	await page.goto('/settings');
	await waitAppReady(page);
	await expect(page.locator('#display')).toHaveValue(`E2E Reg ${tag}`);

	await restoreAdminSession(page);
	await apiJSON(page, 'PUT', '/api/v1/admin/settings', {
		registration_enabled: false,
		external_url: ''
	});
});

test('admin root saves system settings', async ({ page }) => {
	await page.goto('/admin');
	await waitAppReady(page);
	await expect(page.getByText('Открытая регистрация')).toBeVisible();

	await page.getByRole('switch', { name: 'Открытая регистрация' }).click();
	await page.locator('#external').fill('https://e2e.example.com');
	await page
		.locator('form.card.max-w-lg')
		.first()
		.getByRole('button', { name: 'Сохранить', exact: true })
		.click();
	await expectToast(page, 'success', 'Сохранено');

	await page.getByRole('switch', { name: 'Открытая регистрация' }).click();
	await page.locator('#external').fill('');
	await page
		.locator('form.card.max-w-lg')
		.first()
		.getByRole('button', { name: 'Сохранить', exact: true })
		.click();
	await expectToast(page, 'success', 'Сохранено');
});

test('create credit from credits list UI', async ({ page }) => {
	const tag = Date.now();
	const account = await createCashAccount(page, `E2E Credit UI Acc ${tag}`);
	const creditName = `E2E UI Credit ${tag}`;

	await page.goto('/credits');
	await waitAppReady(page);
	await page.getByRole('button', { name: 'Новый кредит' }).click();

	const modal = page.getByRole('dialog');
	await expect(modal.getByRole('heading', { name: 'Новый кредит' })).toBeVisible();
	await modal.getByRole('textbox', { name: 'Название' }).fill(creditName);
	await modal.getByRole('textbox', { name: 'Сумма кредита' }).fill('5000');
	await selectLabeledCombobox(page, 'Счёт списания', { label: account.name });
	await expect(modal.getByText(/Сумма:\s*\d/)).toBeVisible({ timeout: 15_000 });
	await modal.getByRole('button', { name: 'Сохранить' }).click();
	await expect(modal).toHaveCount(0, { timeout: 15_000 });

	await expect(page.getByRole('link', { name: creditName })).toBeVisible({ timeout: 15_000 });
});

test('edit income on /transactions', async ({ page }) => {
	const tag = Date.now();
	const account = await createCashAccount(page, `E2E Inc Edit ${tag}`);
	const description = `E2E income edit ${tag}`;
	await createIncome(page, account.id, '145.50', description);

	await page.goto('/transactions');
	await waitAppReady(page);
	await selectCombobox(page, 'tx-filter-account', { label: account.name });

	const row = page.getByRole('row', { name: new RegExp(`${account.name}.*145\\.50`) });
	await rowMenuAction(page, row, 'Изменить');

	const dialog = page.getByRole('dialog');
	await fillEditTxAmount(dialog, '199.75');
	await dialog.getByRole('button', { name: 'Сохранить' }).click();
	await expect(dialog).toHaveCount(0, { timeout: 15_000 });
	await expect(page.getByRole('row', { name: /199\.75/ })).toBeVisible({ timeout: 10_000 });
});

test('delete income on /transactions', async ({ page }) => {
	const tag = Date.now();
	const account = await createCashAccount(page, `E2E Inc Del ${tag}`);
	await createIncome(page, account.id, '167.25', `E2E income delete ${tag}`);

	await page.goto('/transactions');
	await waitAppReady(page);
	await selectCombobox(page, 'tx-filter-account', { label: account.name });

	const row = page.getByRole('row', { name: new RegExp(`${account.name}.*167\\.25`) });
	await rowMenuAction(page, row, 'Удалить');
	await confirmDialog(page);

	await expect(row).toHaveCount(0, { timeout: 10_000 });
});

test('edit expense from dashboard recent list', async ({ page }) => {
	const tag = Date.now();
	const account = await createCashAccount(page, `E2E Dash Edit ${tag}`);
	const description = `E2E dash edit ${tag}`;
	await apiJSON(page, 'POST', '/api/v1/transactions', {
		account_id: account.id,
		type: 'expense',
		amount: '55.40',
		description,
		transaction_date: new Date().toISOString().slice(0, 19).replace('T', ' ')
	});

	await page.goto('/');
	await waitAppReady(page);
	await expandCollapsibleSection(page, 'Последние операции');

	const row = page.getByRole('row', {
		name: new RegExp(`${description}.*55\\.40|55\\.40.*${description}`)
	});
	await rowMenuAction(page, row, 'Изменить');

	const dialog = page.getByRole('dialog');
	await fillEditTxAmount(dialog, '66.80');
	await dialog.getByRole('button', { name: 'Сохранить' }).click();
	await expect(dialog).toHaveCount(0, { timeout: 15_000 });
	await expandCollapsibleSection(page, 'Последние операции');
	await expect(
		page.getByRole('row', {
			name: new RegExp(`${description}.*66\\.80|66\\.80.*${description}`)
		})
	).toBeVisible({ timeout: 10_000 });
});

test('account detail desktop: create income', async ({ page }) => {
	const account = await createCashAccount(page, `E2E Acc Income ${Date.now()}`);

	await page.setViewportSize({ width: 1280, height: 720 });
	await page.goto(`/accounts/${account.id}`);
	await waitAppReady(page);

	await page.getByRole('button', { name: 'Доход', exact: true }).click();
	await fillTransactionForm(page, { amount: '120', account: account.name });
	await expect(page.getByText('120.00').first()).toBeVisible({ timeout: 10_000 });
});

test('import CSV commits transactions', async ({ page }) => {
	const tag = Date.now();
	const accountName = `E2E Import ${tag}`;
	const description = `E2E import row ${tag}`;
	await createCashAccount(page, accountName);

	const today = new Date();
	const dateStr = `${String(today.getDate()).padStart(2, '0')}.${String(today.getMonth() + 1).padStart(2, '0')}.${today.getFullYear()}`;
	const csv = [
		'Тип,Дата,Сумма списания,Валюта списания,Счет списания,Сумма пополнения,Валюта назначения,Счет пополнения,Категория,Subcategory,Описание,Проект,Пользователь',
		`Расходы,${dateStr},75.00,RUB,${accountName},,,,Продукты,Молоко,${description},,Tester`
	].join('\n');
	const csvPath = path.join(os.tmpdir(), `e2e-import-${tag}.csv`);
	fs.writeFileSync(csvPath, csv, 'utf8');

	try {
		await page.goto('/settings/import');
		await waitAppReady(page);
		await page.locator('input[type="file"]').setInputFiles(csvPath);
		await expect(page.getByText(path.basename(csvPath))).toBeVisible({ timeout: 10_000 });

		await advanceImportToPreview(page);
		await commitImportFromPreview(page);
		await expect(page.getByText(/Создано операций: 1/)).toBeVisible({ timeout: 10_000 });

		const listed = await apiJSON<{ data: unknown[] }>(
			page,
			'GET',
			`/api/v1/transactions?search=${encodeURIComponent(description)}`
		);
		expect(listed.data.length).toBeGreaterThan(0);
	} finally {
		fs.unlinkSync(csvPath);
	}
});

test('login page shows register link when registration enabled', async ({ page }) => {
	await apiJSON(page, 'PUT', '/api/v1/admin/settings', {
		registration_enabled: true,
		external_url: ''
	});

	const status = await page.request.get('/api/v1/setup/status');
	expect((await status.json()).registration_enabled).toBe(true);

	await page.context().clearCookies();
	await page.goto('/login');
	await waitAppReady(page);
	await expect(page.getByRole('link', { name: 'Зарегистрироваться' })).toBeVisible({
		timeout: 10_000
	});

	await login(page);
	await restoreAdminSession(page);
	await apiJSON(page, 'PUT', '/api/v1/admin/settings', {
		registration_enabled: false,
		external_url: ''
	});
});
