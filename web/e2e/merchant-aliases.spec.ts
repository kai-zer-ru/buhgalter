import { test, expect } from '@playwright/test';
import { apiJSON, formatUTCDateTime, waitAppReady } from './helpers/auth';
import { createCashAccount } from './helpers/setup-data';
import { confirmDialog, rowMenuAction } from './helpers/ui';
import { selectCombobox } from './helpers/transactions';

test('merchant alias is found in the form and merge keeps one journal name', async ({ page }) => {
	const tag = Date.now();
	const primary = `Пятёрочка ${tag}`;
	const alias = `Pyaterochka ${tag}`;
	const alias2 = `Pyat2 ${tag}`;
	const other = `Магнит ${tag}`;
	const account = await createCashAccount(page, `E2E Merch ${tag}`);

	await apiJSON(page, 'POST', '/api/v1/transactions', {
		account_id: account.id,
		type: 'expense',
		amount: '18.40',
		description: `merge source ${tag}`,
		merchant_name: other,
		transaction_date: formatUTCDateTime(new Date())
	});

	await page.goto('/merchants');
	await waitAppReady(page);
	await page.getByPlaceholder('Название магазина').fill(primary);
	await page.locator('#merchant-new-aliases').fill(alias);
	await page.getByRole('button', { name: 'Создать' }).click();
	await expect(page.locator('.font-medium', { hasText: primary })).toBeVisible();

	await page.getByPlaceholder('Поиск…').fill(alias);
	await expect(page.locator('.font-medium', { hasText: primary })).toBeVisible();
	await expect(page.locator('.font-medium', { hasText: other })).toHaveCount(0);
	await page.getByPlaceholder('Поиск…').fill('');

	const targetCard = page.locator('.card').filter({
		has: page.locator('.font-medium', { hasText: primary })
	});
	await rowMenuAction(page, targetCard, 'Изменить');
	// Edit mode drops `.font-medium`, so `targetCard` no longer matches this row.
	const aliasInput = page.locator('input[aria-label="Алиасы"]:not(#merchant-new-aliases)');
	await expect(aliasInput).toHaveValue(alias);
	await aliasInput.fill(alias2);
	await page.getByRole('button', { name: 'Сохранить', exact: true }).click();
	await expect(targetCard).toContainText(alias2);

	await page.getByPlaceholder('Название магазина').fill(other);
	await page.getByRole('button', { name: 'Создать' }).click();
	await expect(page.locator('.font-medium', { hasText: other })).toBeVisible();

	await page.goto('/transactions');
	await waitAppReady(page);
	await page.getByRole('button', { name: 'Расход' }).click();
	const dialog = page.getByRole('dialog');
	await expect(dialog).toBeVisible();
	await selectCombobox(page, 'tx-account', { label: account.name });
	await selectCombobox(page, 'tx-category', { index: 0 });
	const amountInput = dialog.locator('#tx-amount');
	await amountInput.click();
	await amountInput.fill('17.40');
	await dialog.locator('summary', { hasText: 'Время, комментарий, магазин, теги' }).click();
	const merchant = dialog.locator('#tx-merchant');
	const list = page.locator('#tx-merchant-list');
	await merchant.fill(alias);
	await expect(list.getByRole('button', { name: /Создать/ })).toBeVisible();
	await merchant.fill(alias2);
	await expect(list.getByRole('button', { name: primary, exact: true })).toBeVisible();
	await expect(list.getByRole('button', { name: /Создать/ })).toHaveCount(0);
	await list.getByRole('button', { name: primary, exact: true }).click();
	await expect(merchant).toHaveValue(primary);
	await dialog.getByRole('button', { name: 'Сохранить' }).click();
	await expect(dialog).toHaveCount(0, { timeout: 15_000 });

	await selectCombobox(page, 'tx-filter-account', { label: account.name });
	const aliasRow = page.getByRole('row', { name: new RegExp(`17\\.40`) });
	await expect(aliasRow).toContainText(primary);
	await expect(aliasRow).not.toContainText(alias);
	await expect(aliasRow).not.toContainText(alias2);

	await page.goto('/merchants');
	await waitAppReady(page);
	await rowMenuAction(page, targetCard, 'Объединить');
	const mergeDialog = page.getByRole('dialog');
	await mergeDialog.getByRole('checkbox', { name: other }).check();
	await mergeDialog.getByRole('button', { name: 'Объединить', exact: true }).click();
	await confirmDialog(page, 'Объединить');
	await expect(page.locator('.font-medium', { hasText: other })).toHaveCount(0);
	await expect(targetCard).toContainText(other);

	await page.goto('/transactions');
	await waitAppReady(page);
	await selectCombobox(page, 'tx-filter-account', { label: account.name });
	const mergedRow = page.getByRole('row', { name: new RegExp(`18\\.40`) });
	await expect(mergedRow).toContainText(primary);
	await expect(mergedRow).not.toContainText(other);
	await expect(page.getByRole('row', { name: new RegExp(`17\\.40`) })).toContainText(primary);
});
