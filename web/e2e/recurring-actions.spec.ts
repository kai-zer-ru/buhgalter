import { test, expect } from '@playwright/test';
import { waitAppReady } from './helpers/auth';
import { createCashAccount } from './helpers/setup-data';
import { selectLabeledCombobox } from './helpers/transactions';
import { confirmDialog, rowMenuAction } from './helpers/ui';

test.describe.configure({ mode: 'serial' });

async function createRecurring(page: import('@playwright/test').Page, description: string) {
	const account = await createCashAccount(page);
	await page.goto('/recurring-operations');
	await waitAppReady(page);
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

	let postedTime = '';
	await page.route('**/api/v1/recurring-operations', async (route) => {
		if (route.request().method() === 'POST') {
			const body = route.request().postDataJSON() as { time_local?: string };
			postedTime = body.time_local ?? '';
		}
		await route.continue();
	});

	await page.goto('/recurring-operations');
	await waitAppReady(page);
	await page.getByRole('button', { name: 'Добавить' }).click();
	await page.locator('#recurring-amount-create').fill('42');
	await page.locator('#recurring-description-create').fill(description);
	await selectLabeledCombobox(page, 'Счёт', { label: account.name });
	await selectLabeledCombobox(page, 'Категория', { index: 0 });
	await page.getByRole('button', { name: 'Создать' }).click();

	await expect(page.getByRole('row', { name: new RegExp(description) })).toBeVisible({
		timeout: 10_000
	});
	expect(postedTime).toBe('08:00');
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

	await page.goto('/recurring-operations');
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
