import { test, expect } from '@playwright/test';
import { waitAppReady } from './helpers/auth';
import { createCashAccount, createCredit } from './helpers/setup-data';
import { rowMenuAction } from './helpers/ui';

test('change credit name via header menu', async ({ page }) => {
	const account = await createCashAccount(page);
	const credit = await createCredit(page, account.id, 'E2E Old Credit Name');
	const newName = `E2E New Credit ${Date.now()}`;

	await page.goto(`/credits/${credit.id}`);
	await waitAppReady(page);

	await page.locator('main').getByRole('button', { name: 'Действия' }).first().click();
	await page.getByRole('menuitem', { name: 'Изменить название' }).click();

	const modal = page.getByRole('dialog');
	await modal.getByRole('textbox', { name: 'Название' }).fill(newName);
	await modal.getByRole('button', { name: 'Сохранить' }).click();

	await expect(page.getByRole('heading', { name: newName })).toBeVisible({ timeout: 10_000 });
});

test('complete credit closes it', async ({ page }) => {
	const account = await createCashAccount(page);
	const credit = await createCredit(page, account.id, `E2E Complete ${Date.now()}`);

	await page.goto(`/credits/${credit.id}`);
	await waitAppReady(page);

	await page.locator('main').getByRole('button', { name: 'Действия' }).first().click();
	await page.getByRole('menuitem', { name: 'Завершить' }).click();

	const modal = page.getByRole('dialog');
	await modal.getByRole('button', { name: 'Завершить' }).click();

	await page.goto('/credits');
	await waitAppReady(page);
	await page.getByRole('tab', { name: 'Завершённые', exact: true }).click();
	await expect(page.getByRole('link', { name: credit.name })).toBeVisible({ timeout: 15_000 });
});

test('credits list: active and closed tabs', async ({ page }) => {
	await page.goto('/credits');
	await waitAppReady(page);

	await expect(page.getByRole('tab', { name: 'Активные', exact: true })).toHaveAttribute(
		'aria-selected',
		'true'
	);
	await page.getByRole('tab', { name: 'Завершённые', exact: true }).click();
	await expect(page.getByRole('tab', { name: 'Завершённые', exact: true })).toHaveAttribute(
		'aria-selected',
		'true'
	);
});

test('change debit account on credit detail', async ({ page }) => {
	const accountA = await createCashAccount(page, 'E2E Credit Acc A');
	const accountB = await createCashAccount(page, 'E2E Credit Acc B');
	const credit = await createCredit(page, accountA.id, `E2E Switch Acc ${Date.now()}`);

	await page.goto(`/credits/${credit.id}`);
	await waitAppReady(page);

	await page.locator('main').getByRole('button', { name: 'Действия' }).first().click();
	await page.getByRole('menuitem', { name: 'Сменить счёт' }).click();

	const modal = page.getByRole('dialog');
	await modal.getByRole('combobox').click();
	await page.getByRole('button', { name: accountB.name, exact: true }).click();
	await modal.getByRole('button', { name: 'Сохранить' }).click();

	await expect(page.getByText(accountB.name).first()).toBeVisible({ timeout: 10_000 });
});

test('pay credit from schedule row menu', async ({ page }) => {
	const account = await createCashAccount(page);
	const credit = await createCredit(page, account.id, `E2E Pay Row ${Date.now()}`);

	await page.goto(`/credits/${credit.id}`);
	await waitAppReady(page);

	const pendingRow = page.locator('table tbody tr').first();
	await rowMenuAction(page, pendingRow, 'Оплатить');
	const modal = page.getByRole('dialog');
	await modal.getByRole('button', { name: 'Оплатить' }).click();

	await expect(page.getByText('Списан').first()).toBeVisible({ timeout: 15_000 });
});
