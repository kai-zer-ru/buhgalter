import { test, expect } from '@playwright/test';
import { waitAppReady, formatUTCDateTime } from './helpers/auth';
import { createCashAccount } from './helpers/setup-data';
import { confirmDialog, rowMenuAction } from './helpers/ui';

test('create borrowed debt (Взять в долг)', async ({ page }) => {
	const debtorName = `E2E Borrowed ${Date.now()}`;
	await page.goto('/debts');
	await waitAppReady(page);

	await page.getByRole('button', { name: 'Взять в долг' }).click();
	const modal = page.getByRole('dialog');
	await modal.getByRole('textbox', { name: 'Имя должника' }).fill(debtorName);
	await modal.getByRole('textbox', { name: 'Сумма' }).fill('300');
	await modal.getByRole('button', { name: 'Создать' }).click();

	await expect(page.getByRole('cell', { name: debtorName })).toBeVisible({ timeout: 10_000 });
	await expect(page.getByRole('cell', { name: 'Взял в долг' })).toBeVisible();
});

test('partial debt settlement', async ({ page }) => {
	const account = await createCashAccount(page);
	const debtorName = `E2E Partial ${Date.now()}`;
	const now = new Date();

	await page.request.post('/api/v1/debts', {
		data: {
			debtor_name: debtorName,
			direction: 'lent',
			amount: '1000.00',
			debt_date: formatUTCDateTime(now),
			due_date: formatUTCDateTime(new Date(now.getTime() + 7 * 24 * 60 * 60 * 1000)),
			affects_balance: false,
			account_id: account.id
		}
	});

	await page.goto('/debts');
	await waitAppReady(page);

	const row = page.getByRole('row', { name: new RegExp(debtorName) });
	await rowMenuAction(page, row, 'Закрыть');
	const modal = page.getByRole('dialog');
	await modal.getByRole('textbox', { name: 'Сумма погашения' }).fill('400');
	await modal.getByRole('button', { name: 'Закрыть' }).click();

	await expect(page.getByRole('cell', { name: debtorName })).toBeVisible({ timeout: 10_000 });
	await expect(page.getByRole('cell', { name: '600.00 ₽' }).first()).toBeVisible();
});

test('delete active debt from row menu', async ({ page }) => {
	const account = await createCashAccount(page);
	const debtorName = `E2E Debt Delete ${Date.now()}`;
	const now = new Date();

	await page.request.post('/api/v1/debts', {
		data: {
			debtor_name: debtorName,
			direction: 'lent',
			amount: '200.00',
			debt_date: formatUTCDateTime(now),
			due_date: formatUTCDateTime(new Date(now.getTime() + 7 * 24 * 60 * 60 * 1000)),
			affects_balance: false,
			account_id: account.id
		}
	});

	await page.goto('/debts');
	await waitAppReady(page);

	const row = page.getByRole('row', { name: new RegExp(debtorName) });
	await rowMenuAction(page, row, 'Удалить');
	await confirmDialog(page);

	await expect(page.getByRole('cell', { name: debtorName })).toHaveCount(0, { timeout: 10_000 });
});

test('navigate from debts list to debtor detail', async ({ page }) => {
	const account = await createCashAccount(page);
	const debtorName = `E2E Debtor Page ${Date.now()}`;
	const now = new Date();
	const debt = await page.request.post('/api/v1/debts', {
		data: {
			debtor_name: debtorName,
			direction: 'lent',
			amount: '150.00',
			debt_date: formatUTCDateTime(now),
			due_date: formatUTCDateTime(new Date(now.getTime() + 7 * 24 * 60 * 60 * 1000)),
			affects_balance: false,
			account_id: account.id
		}
	});
	const body = (await debt.json()) as { debtor_id: string };

	await page.goto('/debts');
	await waitAppReady(page);
	await page.getByRole('link', { name: debtorName }).click();
	await waitAppReady(page);

	await expect(page).toHaveURL(new RegExp(`/debtors/${body.debtor_id}`));
	await expect(page.getByRole('heading', { name: debtorName })).toBeVisible();
});

test('debtor page: lend more debt', async ({ page }) => {
	const account = await createCashAccount(page);
	const debtorName = `E2E Lend More ${Date.now()}`;
	const now = new Date();
	const debt = await page.request.post('/api/v1/debts', {
		data: {
			debtor_name: debtorName,
			direction: 'lent',
			amount: '100.00',
			debt_date: formatUTCDateTime(now),
			due_date: formatUTCDateTime(new Date(now.getTime() + 7 * 24 * 60 * 60 * 1000)),
			affects_balance: false,
			account_id: account.id
		}
	});
	const body = (await debt.json()) as { debtor_id: string };

	await page.goto(`/debtors/${body.debtor_id}`);
	await waitAppReady(page);
	await page.getByRole('button', { name: 'Дать в долг ещё' }).click();
	const modal = page.getByRole('dialog');
	await modal.getByRole('textbox', { name: 'Сумма' }).fill('50');
	await modal.getByRole('button', { name: 'Создать' }).click();

	await expect(page.getByText(/150|50/).first()).toBeVisible({ timeout: 10_000 });
});
