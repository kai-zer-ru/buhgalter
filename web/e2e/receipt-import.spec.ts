import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { test, expect } from '@playwright/test';
import { apiJSON, waitAppReady } from './helpers/auth';
import { enableReceiptScannerFeatures, mockReceiptParseRoute } from './helpers/receipt-scanner';
import { createCashAccount } from './helpers/setup-data';
import { selectCombobox } from './helpers/transactions';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const qrFixture = path.join(__dirname, 'fixtures', 'receipt-qr.png');

const mockReceipt = {
	fiscal_drive_number: '7288940700123456',
	fiscal_document_number: '12345',
	fiscal_sign: '9876543210',
	date_time: '2026-01-15T10:30:00Z',
	total_sum: 15000,
	store_name: 'E2E Магазин',
	items: [{ name: 'E2E Товар', quantity: 1, price: 15000, sum: 15000 }],
	fns_check_url: 'https://proverkacheka.nalog.ru/?fn=7288940700123456'
};

async function getExpenseCategories(page: import('@playwright/test').Page) {
	return apiJSON<{ id: string; name: string; is_system?: boolean }[]>(
		page,
		'GET',
		'/api/v1/categories?type=expense'
	);
}

async function getExpenseCategoryId(page: import('@playwright/test').Page) {
	const cats = await getExpenseCategories(page);
	return cats[0].id;
}

test('receipt import form creates expense from photo scan job', async ({ page }) => {
	await enableReceiptScannerFeatures(page);
	const tag = Date.now();
	const fiscalDoc = String(tag).slice(-5);
	const account = await createCashAccount(page, `E2E Receipt ${tag}`);
	const scannedReceipt = {
		...mockReceipt,
		fiscal_document_number: fiscalDoc
	};

	await mockReceiptParseRoute(page, scannedReceipt);

	await page.goto('/transactions');
	await waitAppReady(page);
	await page.getByRole('button', { name: 'Чек', exact: true }).click();

	const dialog = page.getByRole('dialog');
	await dialog.locator('input[type="file"]').setInputFiles(qrFixture);
	await expect(dialog.getByText('E2E Магазин')).toBeVisible({ timeout: 20_000 });

	await selectCombobox(page, 'receipt-account', { label: account.name });
	await selectCombobox(page, 'receipt-cat-0', { index: 1 });

	await dialog.getByRole('button', { name: /Создать 1 операций/ }).click();
	await expect(dialog).toHaveCount(0, { timeout: 15_000 });

	await selectCombobox(page, 'tx-filter-account', { label: account.name });
	await expect(page.getByRole('row', { name: /E2E Товар.*150\.00/ })).toBeVisible({
		timeout: 10_000
	});
});

test('receipt bulk category applies only to unassigned included rows', async ({ page }) => {
	await enableReceiptScannerFeatures(page);
	const tag = Date.now();
	const fiscalDoc = String(tag).slice(-5);
	const account = await createCashAccount(page, `E2E Receipt Bulk ${tag}`);
	const userCats = (await getExpenseCategories(page)).filter((c) => !c.is_system);
	expect(userCats.length).toBeGreaterThanOrEqual(2);
	const [catManual, catBulk] = userCats;

	const scannedReceipt = {
		...mockReceipt,
		fiscal_document_number: fiscalDoc,
		total_sum: 25000,
		items: [
			{ name: 'E2E Bulk A', quantity: 1, price: 10000, sum: 10000 },
			{ name: 'E2E Bulk B', quantity: 1, price: 15000, sum: 15000 }
		]
	};

	await mockReceiptParseRoute(page, scannedReceipt);

	let importBody: {
		items: { index: number; category_id?: string; include: boolean }[];
	} | null = null;
	await page.route('**/api/v1/receipts/import', async (route) => {
		importBody = route.request().postDataJSON() as typeof importBody;
		await route.continue();
	});

	await page.goto('/transactions');
	await waitAppReady(page);
	await page.getByRole('button', { name: 'Чек', exact: true }).click();

	const dialog = page.getByRole('dialog');
	await dialog.locator('input[type="file"]').setInputFiles(qrFixture);
	await expect(dialog.getByText('E2E Магазин')).toBeVisible({ timeout: 20_000 });

	await selectCombobox(page, 'receipt-account', { label: account.name });
	await selectCombobox(page, 'receipt-cat-0', { label: catManual.name });
	await selectCombobox(page, 'receipt-bulk-category', { label: catBulk.name });
	await dialog.getByRole('button', { name: 'Проставить без категории' }).click();

	await dialog.getByRole('button', { name: /Создать 2 операций/ }).click();
	await expect(dialog).toHaveCount(0, { timeout: 15_000 });

	expect(importBody).not.toBeNull();
	const items = importBody!.items;
	expect(items.find((i) => i.index === 0)?.category_id).toBe(catManual.id);
	expect(items.find((i) => i.index === 1)?.category_id).toBe(catBulk.id);

	await selectCombobox(page, 'tx-filter-account', { label: account.name });
	await expect(
		page.getByRole('row').filter({ hasText: 'E2E Bulk A' }).filter({ hasText: '100.00' })
	).toBeVisible({ timeout: 10_000 });
	await expect(
		page.getByRole('row').filter({ hasText: 'E2E Bulk B' }).filter({ hasText: '150.00' })
	).toBeVisible({ timeout: 10_000 });
});

test('receipt icon opens detail dialog with FNS link', async ({ page }) => {
	await enableReceiptScannerFeatures(page);
	const tag = Date.now();
	const account = await createCashAccount(page, `E2E Receipt View ${tag}`);
	const categoryId = await getExpenseCategoryId(page);
	const fiscalDoc = String(tag).slice(-5);

	const imported = await apiJSON<{
		receipt_import_id: string;
		transactions: { id: string; description: string }[];
	}>(page, 'POST', '/api/v1/receipts/import', {
		account_id: account.id,
		fiscal_drive_number: '7288940700123456',
		fiscal_document_number: fiscalDoc,
		fiscal_sign: '111222333',
		receipt: {
			...mockReceipt,
			fiscal_document_number: fiscalDoc,
			items: [{ name: 'E2E Dialog Item', quantity: 1, price: 9900, sum: 9900 }],
			total_sum: 9900
		},
		items: [{ index: 0, category_id: categoryId, include: true }]
	});

	await page.goto('/transactions');
	await waitAppReady(page);
	await selectCombobox(page, 'tx-filter-account', { label: account.name });

	const row = page.getByRole('row', { name: /E2E Dialog Item.*99\.00/ });
	await expect(row).toBeVisible({ timeout: 10_000 });
	await row.getByRole('button', { name: 'Данные чека' }).click();

	const dialog = page.getByRole('dialog').filter({ hasText: 'E2E Магазин' });
	await expect(dialog.getByText('E2E Dialog Item')).toBeVisible();
	await expect(dialog.getByRole('link', { name: 'Открыть на сайте ФНС' })).toHaveAttribute(
		'href',
		/proverkacheka\.nalog\.ru/
	);

	await page.keyboard.press('Escape');
	await expect(dialog).toHaveCount(0);

	expect(imported.receipt_import_id).toBeTruthy();
	expect(imported.transactions[0]?.description).toContain(`Чек №${fiscalDoc}`);
});

test('attach receipt to existing expense with sum mismatch confirm', async ({ page }) => {
	await enableReceiptScannerFeatures(page);
	const tag = Date.now();
	const account = await createCashAccount(page, `E2E Attach ${tag}`);
	const categoryId = await getExpenseCategoryId(page);

	const tx = await apiJSON<{ id: string }>(page, 'POST', '/api/v1/transactions', {
		account_id: account.id,
		type: 'expense',
		amount: '100.00',
		category_id: categoryId,
		description: `E2E attach ${tag}`,
		transaction_date: '2026-03-01 12:00:00'
	});

	await mockReceiptParseRoute(page, {
		...mockReceipt,
		total_sum: 15000,
		fiscal_document_number: 'attach-doc'
	});

	let attachBody: { acknowledge_sum_mismatch?: boolean } | null = null;
	await page.route('**/api/v1/receipts/attach', async (route) => {
		attachBody = route.request().postDataJSON() as { acknowledge_sum_mismatch?: boolean };
		await route.fulfill({
			status: 200,
			contentType: 'application/json',
			body: JSON.stringify({
				receipt_import_id: 'attach-import-1',
				transaction: { ...tx, receipt_import_id: 'attach-import-1' }
			})
		});
	});

	await page.goto('/transactions');
	await waitAppReady(page);
	await selectCombobox(page, 'tx-filter-account', { label: account.name });

	const row = page.getByRole('row', { name: new RegExp(`E2E attach ${tag}`) });
	await expect(row).toBeVisible({ timeout: 10_000 });
	await row.getByRole('button', { name: 'Действия' }).click();
	await page.getByRole('menuitem', { name: 'Прикрепить чек' }).click();

	const dialog = page.getByRole('dialog').filter({ hasText: 'Прикрепить чек' });
	await dialog.locator('input[type="file"]').setInputFiles(qrFixture);
	await expect(dialog.getByText('E2E Магазин')).toBeVisible({ timeout: 20_000 });
	// Category pre-filled from transaction; confirm attach.
	await dialog.getByRole('button', { name: 'Прикрепить', exact: true }).click();

	const confirmDialog = page.getByRole('alertdialog');
	await expect(confirmDialog.getByText(/100\.00/)).toBeVisible();
	await expect(confirmDialog.getByText(/150\.00/)).toBeVisible();
	await confirmDialog.getByRole('button', { name: 'Добавить всё равно' }).click();

	await expect(dialog).toHaveCount(0, { timeout: 15_000 });
	expect(attachBody?.acknowledge_sum_mismatch).toBe(true);
});
