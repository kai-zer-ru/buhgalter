import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { test, expect } from '@playwright/test';
import { selectCombobox, selectLabeledCombobox, fillTransactionForm } from './helpers/transactions';
import { waitAppReady, apiJSON, formatUTCDateTime } from './helpers/auth';
import { createCashAccount, createExpense } from './helpers/setup-data';

const __dirname = path.dirname(fileURLToPath(import.meta.url));

test.describe.configure({ mode: 'serial' });

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
	await page.getByRole('button', { name: 'Расход', exact: true }).click();
	await fillTransactionForm(page, { amount: '250', account: 'E2E Cash' });
	await expect(page.getByText('250.00').first()).toBeVisible({ timeout: 10_000 });

	await page.goto('/accounts');
	await waitAppReady(page);
	await expect(page.getByRole('link', { name: /E2E Cash.*750/ })).toBeVisible({ timeout: 10_000 });
});

test('create income on dashboard', async ({ page }) => {
	await page.goto('/');
	await waitAppReady(page);
	await page.getByRole('button', { name: 'Доход', exact: true }).click();
	await fillTransactionForm(page, { amount: '100', account: 'E2E Cash' });
	await expect(page.getByText('100.00').first()).toBeVisible({ timeout: 10_000 });
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
	await page.getByRole('button', { name: 'Перевод', exact: true }).click();
	await selectCombobox(page, 'from-acc', { label: 'E2E Cash' });
	await selectCombobox(page, 'to-acc', { label: 'E2E Bank' });
	await page.locator('#tr-amount').fill('100');
	await page.getByRole('button', { name: 'Сохранить' }).click();
	await expect(page.getByText('100.00').first()).toBeVisible({ timeout: 10_000 });
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

test('export tab opens download link', async ({ page }) => {
	await page.goto('/settings?tab=import');
	await waitAppReady(page);
	await page.getByRole('tab', { name: 'Экспорт', exact: true }).click();
	await expect(page.getByRole('button', { name: 'Скачать CSV' })).toBeVisible();
});

test('settings change theme', async ({ page }) => {
	await page.goto('/settings');
	await waitAppReady(page);
	await selectCombobox(page, 'theme', { label: 'Тёмная' });
	await page.getByRole('button', { name: 'Сохранить' }).click();

	await expect(page.locator('html')).toHaveClass(/dark/, { timeout: 10_000 });
});

test('create API token', async ({ page }) => {
	await page.goto('/settings?tab=tokens');
	await waitAppReady(page);
	await page.locator('#token-name').fill('E2E Token');
	await page.getByRole('button', { name: 'Создать' }).click();
	await expect(page.getByText('API-токен создан')).toBeVisible({ timeout: 10_000 });
	await page.getByRole('button', { name: 'Закрыть' }).click();
	await expect(page.getByText('E2E Token').first()).toBeVisible();
});

test('admin create user and manual backup', async ({ page }) => {
	await page.goto('/admin/users');
	await waitAppReady(page);
	await page.getByLabel('Логин').fill('e2euser');
	await page.getByLabel('Пароль', { exact: true }).fill('userpass1');
	await page.getByLabel('Подтверждение пароля').fill('userpass1');
	await page.getByRole('button', { name: 'Создать' }).click();
	await expect(page.getByRole('cell', { name: 'e2euser' }).first()).toBeVisible({
		timeout: 10_000
	});

	await page.goto('/admin/backups');
	await waitAppReady(page);
	await page.getByRole('button', { name: 'Запустить сейчас' }).click();
	await expect(page.getByText(/\.db/)).toBeVisible({ timeout: 15_000 });
});

test('stats page shows summary and category sections', async ({ page }) => {
	const marker = `E2E stats marker ${Date.now()}`;
	const account = await createCashAccount(page, `E2E Stats ${Date.now()}`);
	await createExpense(page, account.id, '250.00', marker);

	await page.goto('/stats');
	await waitAppReady(page);
	await expect(page.getByRole('heading', { name: 'Статистика', level: 1 })).toBeVisible();
	await expect(page.getByRole('heading', { name: 'По категориям' })).toBeVisible();
	await expect(page.getByRole('heading', { name: 'По периодам' })).toBeVisible();

	await page.getByPlaceholder('Комментарий операции').fill(marker);
	await expect(page.getByRole('heading', { name: 'Результаты поиска' })).toBeVisible({
		timeout: 10_000
	});
	await expect(page.getByRole('row', { name: new RegExp(marker) })).toBeVisible({ timeout: 10_000 });
});

test('add expense category', async ({ page }) => {
	const name = `E2E Cat ${Date.now()}`;
	await page.goto('/settings?tab=categories');
	await waitAppReady(page);
	await page.getByPlaceholder('Название категории').fill(name);
	await page.getByRole('button', { name: 'Создать' }).click();
	await expect(page.getByText(name)).toBeVisible({ timeout: 10_000 });
});

test('create credit and pay first installment', async ({ page }) => {
	const account = await createCashAccount(page);
	const creditName = `E2E UI Credit ${Date.now()}`;
	const now = new Date();
	const credit = await apiJSON<{ id: string }>(page, 'POST', '/api/v1/credits', {
		name: creditName,
		principal_amount: '6000.00',
		issue_date: formatUTCDateTime(new Date(now.getTime() - 24 * 60 * 60 * 1000)),
		term_months: 3,
		interest_rate: 0,
		payment_interval: 'month',
		debit_account_id: account.id,
		added_retroactively: false,
		create_transactions: true
	});

	await page.goto(`/credits/${credit.id}`);
	await waitAppReady(page);
	await page.getByRole('button', { name: 'Оплатить' }).first().click();
	const payModal = page.getByRole('dialog');
	await payModal.getByRole('button', { name: 'Оплатить' }).click();
	await expect(page.getByText('Списан').first()).toBeVisible({ timeout: 15_000 });
});

test('create debt and settle', async ({ page }) => {
	const debtorName = `E2E UI Debtor ${Date.now()}`;
	await page.goto('/debts');
	await waitAppReady(page);
	await expect(page.getByRole('button', { name: 'Дать в долг' })).toBeVisible();
	await page.getByRole('button', { name: 'Дать в долг' }).click();
	const debtModal = page.getByRole('dialog');
	await debtModal.getByRole('textbox', { name: 'Имя должника' }).fill(debtorName);
	await debtModal.getByRole('textbox', { name: 'Сумма' }).fill('500');
	await debtModal.getByRole('button', { name: 'Создать' }).click();
	await expect(page.getByRole('cell', { name: debtorName })).toBeVisible({ timeout: 10_000 });

	const row = page.getByRole('row', { name: new RegExp(debtorName) });
	await row.getByRole('button', { name: 'Действия' }).click();
	await page.getByRole('menuitem', { name: 'Закрыть' }).click();
	const settleModal = page.getByRole('dialog');
	await settleModal.getByRole('button', { name: 'Закрыть' }).click();
	await expect(page.getByRole('cell', { name: debtorName })).toHaveCount(0, { timeout: 10_000 });

	await page.getByRole('tab', { name: 'Закрытые', exact: true }).click();
	await expect(page.getByRole('cell', { name: debtorName })).toBeVisible({ timeout: 10_000 });
});

test('create recurring operation', async ({ page }) => {
	const description = `E2E Recurring ${Date.now()}`;
	await page.goto('/recurring-operations');
	await waitAppReady(page);
	await page.getByRole('button', { name: 'Добавить' }).click();
	await page.locator('#recurring-amount-create').fill('99');
	await page.locator('#recurring-description-create').fill(description);
	await selectLabeledCombobox(page, 'Счёт', { label: 'E2E Cash' });
	await selectLabeledCombobox(page, 'Категория', { index: 0 });
	await page.getByRole('button', { name: 'Создать' }).click();
	await expect(page.getByRole('row', { name: new RegExp(description) })).toBeVisible({
		timeout: 10_000
	});
});

test('notifications settings load', async ({ page }) => {
	await page.goto('/settings?tab=notifications');
	await waitAppReady(page);
	await expect(page.getByRole('tab', { name: 'Уведомления' })).toHaveAttribute(
		'aria-selected',
		'true'
	);
	await expect(
		page
			.getByRole('heading', { name: 'Ключ шифрования не настроен' })
			.or(page.getByRole('heading', { name: 'Telegram', exact: true }))
	).toBeVisible();
});

test('admin diagnostics loads', async ({ page }) => {
	await page.goto('/admin/diagnostics');
	await waitAppReady(page);
	await expect(page.getByRole('heading', { name: 'Диагностика', level: 2 })).toBeVisible();
	await expect(page.getByRole('cell', { name: 'app_version', exact: true })).toBeVisible();
	await expect(page.getByText('1.2.2').first()).toBeVisible();
});

test('password reset request on login page', async ({ page }) => {
	await page.context().clearCookies();
	await page.goto('/login');
	await page.getByRole('button', { name: 'Запросить сброс пароля' }).click();
	await page.getByRole('dialog').locator('input').fill('admin');
	await page.getByRole('button', { name: 'Отправить запрос' }).click();
	await expect(page.getByText('Запрос отправлен')).toBeVisible({ timeout: 10_000 });
});
