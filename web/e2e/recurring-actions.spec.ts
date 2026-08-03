import { test, expect } from '@playwright/test';
import { waitAppReady, apiJSON } from './helpers/auth';
import { createCashAccount } from './helpers/setup-data';
import { selectLabeledCombobox } from './helpers/transactions';
import { confirmDialog, rowMenuAction } from './helpers/ui';

test.describe.configure({ mode: 'serial' });

async function createRecurring(page: import('@playwright/test').Page, description: string) {
	const account = await createCashAccount(page);
	await page.goto('/settings/recurring-operations');
	await waitAppReady(page);
	const header = page
		.locator('div')
		.filter({ has: page.getByRole('heading', { name: 'Периодические операции', level: 1 }) })
		.filter({ has: page.getByRole('button', { name: 'Добавить' }) });
	await expect(header.first()).toBeVisible();
	await page.getByRole('button', { name: 'Добавить' }).click();
	await page.locator('#recurring-amount-create').fill('55');
	await page.locator('#recurring-description-create').fill(description);
	await selectLabeledCombobox(page, 'Счёт', { label: account.name });
	await selectLabeledCombobox(page, 'Категория', { index: 0 });
	await page.getByRole('button', { name: 'Создать' }).click();
	await expect(page.getByRole('row', { name: new RegExp(description) })).toBeVisible({
		timeout: 10_000
	});
}

test('create recurring uses 08:00 local time by default', async ({ page }) => {
	const account = await createCashAccount(page);
	const description = `E2E Rec Time ${Date.now()}`;

	await page.goto('/settings/recurring-operations');
	await waitAppReady(page);

	await page.getByRole('button', { name: 'Добавить' }).click();
	await page.locator('#recurring-amount-create').fill('42');
	await page.locator('#recurring-description-create').fill(description);
	await selectLabeledCombobox(page, 'Счёт', { label: account.name });
	await selectLabeledCombobox(page, 'Категория', { index: 0 });

	// waitForRequest is race-free vs page.route + side-effect flag (flaky empty postedTime).
	const postPromise = page.waitForRequest((req) => {
		if (req.method() !== 'POST') return false;
		const path = new URL(req.url()).pathname.replace(/\/$/, '');
		return path.endsWith('/api/v1/recurring-operations');
	});
	await page.getByRole('button', { name: 'Создать' }).click();
	const body = (await postPromise).postDataJSON() as { time_local?: string };
	expect(body.time_local).toBe('08:00');

	await expect(page.getByRole('row', { name: new RegExp(description) })).toBeVisible({
		timeout: 10_000
	});
});

test('edit recurring operation inline', async ({ page }) => {
	const description = `E2E Rec Edit ${Date.now()}`;
	const updated = `${description} updated`;
	await createRecurring(page, description);

	const row = page.getByRole('row', { name: new RegExp(description) });
	await rowMenuAction(page, row, 'Изменить');
	await page.locator('#recurring-description-edit').fill(updated);
	await page.getByRole('button', { name: 'Сохранить' }).click();

	await expect(page.getByRole('row', { name: new RegExp(updated) })).toBeVisible({
		timeout: 10_000
	});
});

test('delete recurring operation', async ({ page }) => {
	const description = `E2E Rec Delete ${Date.now()}`;
	await createRecurring(page, description);

	const row = page.getByRole('row', { name: new RegExp(description) });
	await rowMenuAction(page, row, 'Удалить');
	await confirmDialog(page);

	await expect(page.getByRole('row', { name: new RegExp(description) })).toHaveCount(0, {
		timeout: 10_000
	});
});

test('create weekly recurring with weekday selector', async ({ page }) => {
	const account = await createCashAccount(page);
	const description = `E2E Weekly ${Date.now()}`;

	await page.goto('/settings/recurring-operations');
	await waitAppReady(page);
	await page.getByRole('button', { name: 'Добавить' }).click();
	await page.locator('#recurring-amount-create').fill('33');
	await page.locator('#recurring-description-create').fill(description);
	await selectLabeledCombobox(page, 'Счёт', { label: account.name });
	await selectLabeledCombobox(page, 'Категория', { index: 0 });
	await page.locator('#recurring-period-create').selectOption('week');
	await page.getByRole('button', { name: 'Создать' }).click();

	await expect(page.getByRole('row', { name: new RegExp(description) })).toBeVisible({
		timeout: 10_000
	});
});

test('recurring expense updates account balance when applied', async ({ page }) => {
	const account = await createCashAccount(page);
	const categories = await apiJSON<{ id: string }[]>(
		page,
		'GET',
		'/api/v1/categories?type=expense'
	);
	const description = `E2E Rec Balance ${Date.now()}`;

	const created = await apiJSON<{ id: string }>(page, 'POST', '/api/v1/recurring-operations', {
		type: 'expense',
		amount: '50.00',
		description,
		account_id: account.id,
		category_id: categories[0].id,
		period: 'month',
		day_of_month: 1,
		start_date: '2020-01-01 00:00:00',
		time_local: '08:00'
	});

	await apiJSON<{ applied: number }>(
		page,
		'POST',
		`/api/v1/test/recurring-operations/${created.id}/run-now`
	);

	const bal = await apiJSON<{ balance: number }>(
		page,
		'GET',
		`/api/v1/accounts/${account.id}/balance`
	);
	expect(bal.balance).toBe(95_000);

	const txs = await apiJSON<{ data: { description?: string }[] }>(
		page,
		'GET',
		`/api/v1/transactions?search=${encodeURIComponent(description)}`
	);
	expect(txs.data.some((tx) => tx.description === description)).toBeTruthy();
});
