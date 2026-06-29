import { test, expect } from '@playwright/test';
import { waitAppReady } from './helpers/auth';
import {
	createCashAccount,
	createExpense,
	createIncome,
	createTransfer
} from './helpers/setup-data';
import { confirmDialog, rowMenuAction } from './helpers/ui';
import { fillEditTxAmount, selectCombobox } from './helpers/transactions';

test('edit expense on /transactions', async ({ page }) => {
	const tag = Date.now();
	const account = await createCashAccount(page, `E2E Edit Exp ${tag}`);
	await createExpense(page, account.id, '111.00', `E2E edit target ${tag}`);

	await page.goto('/transactions');
	await waitAppReady(page);
	await selectCombobox(page, 'tx-filter-account', { label: account.name });

	const row = page.getByRole('row', { name: new RegExp(`${account.name}.*111\\.00`) });
	await rowMenuAction(page, row, 'Изменить');

	const dialog = page.getByRole('dialog');
	await fillEditTxAmount(dialog, '222', /222(\.00)?/);
	await dialog.getByRole('button', { name: 'Сохранить' }).click();
	await expect(dialog).toHaveCount(0, { timeout: 15_000 });
	await expect(page.getByRole('row', { name: /222\.00/ })).toBeVisible({ timeout: 10_000 });
});

test('delete expense on /transactions', async ({ page }) => {
	const tag = Date.now();
	const account = await createCashAccount(page, `E2E Del Exp ${tag}`);
	await createExpense(page, account.id, '333.00', `E2E delete target ${tag}`);

	await page.goto('/transactions');
	await waitAppReady(page);
	await selectCombobox(page, 'tx-filter-account', { label: account.name });

	const row = page.getByRole('row', { name: new RegExp(`${account.name}.*333\\.00`) });
	await rowMenuAction(page, row, 'Удалить');
	await confirmDialog(page);

	await expect(page.getByRole('row', { name: /333\.00/ })).toHaveCount(0, { timeout: 10_000 });
});

test('edit transfer on /transactions', async ({ page }) => {
	const tag = Date.now();
	const from = await createCashAccount(page, `E2E Tr From ${tag}`);
	const to = await createCashAccount(page, `E2E Tr To ${tag}`);
	await createTransfer(page, from.id, to.id, '58.50');

	await page.goto('/transactions');
	await waitAppReady(page);
	await selectCombobox(page, 'tx-filter-account', { label: from.name });

	const transferRow = page.getByRole('row', {
		name: new RegExp(`${from.name}.*${to.name}.*58\\.50`)
	});
	await expect(transferRow).toBeVisible({ timeout: 10_000 });
	await rowMenuAction(page, transferRow, 'Изменить');

	const dialog = page.getByRole('dialog');
	const amountInput = dialog.locator('#tr-amount');
	await amountInput.click();
	await amountInput.fill('91.25');
	await expect(amountInput).toHaveValue(/91\.25/);
	await dialog.getByRole('button', { name: 'Сохранить' }).click();
	await expect(dialog).toHaveCount(0, { timeout: 15_000 });

	await expect(
		page.getByRole('row', { name: new RegExp(`${from.name}.*${to.name}.*91\\.25`) })
	).toBeVisible({ timeout: 15_000 });
});

test('delete transfer on /transactions', async ({ page }) => {
	const tag = Date.now();
	const from = await createCashAccount(page, `E2E Tr Del From ${tag}`);
	const to = await createCashAccount(page, `E2E Tr Del To ${tag}`);
	await createTransfer(page, from.id, to.id, '44.40');

	await page.goto('/transactions');
	await waitAppReady(page);
	await selectCombobox(page, 'tx-filter-account', { label: from.name });

	const transferRow = page.getByRole('row', {
		name: new RegExp(`${from.name}.*${to.name}.*44\\.40`)
	});
	await rowMenuAction(page, transferRow, 'Удалить');
	await confirmDialog(page);

	await expect(transferRow).toHaveCount(0, { timeout: 10_000 });
});

test('repeat expense opens prefilled create form', async ({ page }) => {
	const tag = Date.now();
	const account = await createCashAccount(page, `E2E Repeat Exp ${tag}`);
	const description = `E2E repeat source ${tag}`;
	await createExpense(page, account.id, '55.25', description);

	await page.goto('/transactions');
	await waitAppReady(page);
	await selectCombobox(page, 'tx-filter-account', { label: account.name });

	const row = page.getByRole('row', { name: new RegExp(`${account.name}.*55\\.25`) });
	await rowMenuAction(page, row, 'Повторить');

	const dialog = page.getByRole('dialog');
	await expect(dialog.getByRole('heading', { name: 'Расход' })).toBeVisible();
	await expect(dialog.locator('#tx-amount')).toHaveValue(/55\.25/);
	await expect(dialog.locator('#tx-desc')).toHaveValue(description);
	await dialog.getByRole('button', { name: 'Сохранить' }).click();
	await expect(dialog).toHaveCount(0, { timeout: 15_000 });

	await expect(page.getByRole('row', { name: /55\.25/ })).toHaveCount(2, { timeout: 10_000 });
});

test('repeat income opens prefilled create form', async ({ page }) => {
	const tag = Date.now();
	const account = await createCashAccount(page, `E2E Repeat Inc ${tag}`);
	const description = `E2E repeat income ${tag}`;
	await createIncome(page, account.id, '77.75', description);

	await page.goto('/transactions');
	await waitAppReady(page);
	await selectCombobox(page, 'tx-filter-account', { label: account.name });

	const row = page.getByRole('row', { name: new RegExp(`${account.name}.*77\\.75`) });
	await rowMenuAction(page, row, 'Повторить');

	const dialog = page.getByRole('dialog');
	await expect(dialog.getByRole('heading', { name: 'Доход' })).toBeVisible();
	await expect(dialog.locator('#tx-amount')).toHaveValue(/77\.75/);
	await expect(dialog.locator('#tx-desc')).toHaveValue(description);
	await dialog.getByRole('button', { name: 'Сохранить' }).click();
	await expect(dialog).toHaveCount(0, { timeout: 15_000 });

	await expect(page.getByRole('row', { name: /77\.75/ })).toHaveCount(2, { timeout: 10_000 });
});

test('repeat transfer opens prefilled create form', async ({ page }) => {
	const tag = Date.now();
	const from = await createCashAccount(page, `E2E Repeat From ${tag}`);
	const to = await createCashAccount(page, `E2E Repeat To ${tag}`);
	await createTransfer(page, from.id, to.id, '42.00');

	await page.goto('/transactions');
	await waitAppReady(page);
	await selectCombobox(page, 'tx-filter-account', { label: from.name });

	const transferRow = page.getByRole('row', {
		name: new RegExp(`${from.name}.*${to.name}.*42\\.00`)
	});
	await rowMenuAction(page, transferRow, 'Повторить');

	const dialog = page.getByRole('dialog');
	await expect(dialog.getByRole('heading', { name: 'Перевод' })).toBeVisible();
	await expect(dialog.locator('#tr-amount')).toHaveValue(/42\.00/);
	await dialog.getByRole('button', { name: 'Сохранить' }).click();
	await expect(dialog).toHaveCount(0, { timeout: 15_000 });

	await expect(
		page.getByRole('row', { name: new RegExp(`${from.name}.*${to.name}.*42\\.00`) })
	).toHaveCount(2, { timeout: 15_000 });
});

test('make transaction recurring opens prefilled form', async ({ page }) => {
	const tag = Date.now();
	const account = await createCashAccount(page, `E2E Rec Acc ${tag}`);
	const description = `E2E recurring source ${tag}`;
	await createExpense(page, account.id, '88.50', description);

	await page.goto('/transactions');
	await waitAppReady(page);
	await selectCombobox(page, 'tx-filter-account', { label: account.name });

	const row = page.getByRole('row', { name: new RegExp(`${account.name}.*88\\.50`) });
	await rowMenuAction(page, row, 'Сделать периодической');

	await waitAppReady(page);
	await expect(page).toHaveURL(/\/recurring-operations\/?$/);
	await expect(page.locator('#recurring-amount-create')).toHaveValue(/88\.50/);
	await expect(page.locator('#recurring-description-create')).toHaveValue(description);
	await expect(page.getByText('Форма заполнена по операции')).toBeVisible();
});

test('delete expense from dashboard recent list', async ({ page }) => {
	const account = await createCashAccount(page);
	await createExpense(page, account.id, '77.00', 'E2E dashboard delete');

	await page.goto('/');
	await waitAppReady(page);

	const row = page.getByRole('row', { name: /77\.00/ });
	await rowMenuAction(page, row, 'Удалить');
	await confirmDialog(page);

	await expect(page.getByRole('row', { name: /77\.00/ })).toHaveCount(0, { timeout: 10_000 });
});

test('create transfer with commission', async ({ page }) => {
	const from = await createCashAccount(page, 'E2E Comm From');
	const to = await createCashAccount(page, 'E2E Comm To');

	await page.goto('/');
	await waitAppReady(page);
	await page.getByRole('button', { name: 'Перевод', exact: true }).click();

	await selectCombobox(page, 'from-acc', { label: from.name });
	await selectCombobox(page, 'to-acc', { label: to.name });
	const dialog = page.getByRole('dialog');
	await dialog.locator('#tr-amount').fill('100');
	await dialog.locator('#tr-commission').fill('5');
	await dialog.getByRole('button', { name: 'Сохранить' }).click();
	await expect(dialog).toHaveCount(0, { timeout: 15_000 });

	await page.goto('/transactions');
	await waitAppReady(page);
	await expect(page.getByText('5.00').first()).toBeVisible({ timeout: 10_000 });
});
