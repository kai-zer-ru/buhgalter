import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { test, expect } from '@playwright/test';
import { apiJSON, waitAppReady } from './helpers/auth';
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

async function getExpenseCategoryId(page: import('@playwright/test').Page) {
	const cats = await apiJSON<{ id: string }[]>(page, 'GET', '/api/v1/categories?type=expense');
	return cats[0].id;
}

test('receipt import form creates expense from photo scan job', async ({ page }) => {
	const tag = Date.now();
	const account = await createCashAccount(page, `E2E Receipt ${tag}`);
	const categoryId = await getExpenseCategoryId(page);

	await page.route('**/api/v1/receipts/scan-jobs', async (route) => {
		if (route.request().method() === 'POST') {
			await route.fulfill({
				status: 202,
				contentType: 'application/json',
				body: JSON.stringify({ id: 'scan-job-1', status: 'queued' })
			});
			return;
		}
		await route.continue();
	});

	await page.route('**/api/v1/receipts/scan-jobs/scan-job-1', async (route) => {
		await route.fulfill({
			status: 200,
			contentType: 'application/json',
			body: JSON.stringify({ id: 'scan-job-1', status: 'done', result: mockReceipt })
		});
	});

	await page.goto('/transactions');
	await waitAppReady(page);
	await page.getByRole('button', { name: 'Чек', exact: true }).click();

	const dialog = page.getByRole('dialog');
	await dialog.locator('input[type="file"]').setInputFiles(qrFixture);
	await expect(dialog.getByText('E2E Магазин')).toBeVisible({ timeout: 20_000 });

	await selectCombobox(page, 'receipt-cat-0', { index: 1 });

	await page.route('**/api/v1/receipts/import', async (route) => {
		await route.fulfill({
			status: 200,
			contentType: 'application/json',
			body: JSON.stringify({
				receipt_import_id: 'import-1',
				transactions: [
					{
						id: 'tx-1',
						description: `E2E Товар. Чек №${mockReceipt.fiscal_document_number}, сумма 150.00`
					}
				]
			})
		});
	});

	await dialog.getByRole('button', { name: /Создать 1 операций/ }).click();
	await expect(dialog).toHaveCount(0, { timeout: 15_000 });

	await selectCombobox(page, 'tx-filter-account', { label: account.name });
	await expect(page.getByRole('row', { name: /E2E Товар.*150\.00/ })).toBeVisible({ timeout: 10_000 });
});

test('receipt icon opens detail dialog with FNS link', async ({ page }) => {
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
