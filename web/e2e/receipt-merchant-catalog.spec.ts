import { test, expect } from '@playwright/test';
import { apiJSON, formatUTCDateTime, waitAppReady } from './helpers/auth';
import { enableReceiptScannerFeatures } from './helpers/receipt-scanner';
import { createCashAccount } from './helpers/setup-data';
import { rowMenuAction } from './helpers/ui';
import { selectCombobox } from './helpers/transactions';

test('expense form has one merchant field and screens have no receipt-store filter', async ({
	page
}) => {
	const tag = Date.now();
	const merchant = `E2E Каталог ${tag}`;
	const other = `E2E Другой ${tag}`;
	const account = await createCashAccount(page, `E2E Catalog ${tag}`);
	const spentAt = formatUTCDateTime(new Date(Date.now() - 3_600_000));
	await apiJSON(page, 'POST', '/api/v1/transactions', {
		account_id: account.id,
		type: 'expense',
		amount: '12.50',
		merchant_name: merchant,
		transaction_date: spentAt
	});
	await apiJSON(page, 'POST', '/api/v1/transactions', {
		account_id: account.id,
		type: 'expense',
		amount: '40.00',
		merchant_name: other,
		transaction_date: spentAt
	});

	await page.goto('/transactions');
	await waitAppReady(page);
	await expect(page.getByText('Точка из чека')).toHaveCount(0);

	await page.getByRole('button', { name: 'Расход' }).click();
	const dialog = page.getByRole('dialog');
	await expect(dialog).toBeVisible();
	await expect(dialog.locator('#tx-store')).toHaveCount(0);
	await expect(dialog.locator('#tx-category')).toBeVisible();
	const details = dialog.locator('details');
	await details.locator('summary', { hasText: 'Время, комментарий, магазин, теги' }).click();
	await expect(details.locator('#tx-merchant')).toHaveCount(1);
	await expect(dialog.locator('#tx-merchant')).toHaveCount(1);
	await page.keyboard.press('Escape');
	await expect(dialog).toHaveCount(0);

	await page.goto(`/accounts/${account.id}`);
	await waitAppReady(page);
	await expect(page.getByText('Точка из чека')).toHaveCount(0);

	await page.goto('/stats');
	await waitAppReady(page);
	await expect(page.getByText('Точка из чека')).toHaveCount(0);
	await expect(page.locator('#stats-merchant')).toBeVisible();
	await expect(page.getByRole('heading', { name: 'По магазинам' })).toHaveCount(1);
	await expect(page.getByRole('heading', { name: 'По магазинам' })).toBeVisible();
	const byStore = page.locator('.card', {
		has: page.getByRole('heading', { name: 'По магазинам' })
	});
	await expect(byStore.getByRole('cell', { name: merchant })).toBeVisible();
	await expect(byStore.getByRole('cell', { name: other })).toBeVisible();

	await page.locator('#stats-merchant').click();
	const merchantList = page.locator('#stats-merchant-list');
	await page.locator('#stats-merchant-search').fill(merchant);
	const merchantOption = merchantList.getByRole('button', { name: merchant, exact: true });
	await expect(merchantOption).toBeVisible();
	await expect(merchantList.getByRole('button')).toHaveCount(1);
	await merchantOption.evaluate((el) => (el as HTMLButtonElement).click());

	await expect(byStore.locator('tbody tr')).toHaveCount(1);
	await expect(byStore.getByRole('cell', { name: merchant })).toBeVisible();
	await expect(byStore.getByRole('cell', { name: other })).toHaveCount(0);
	const expenseCard = page
		.locator('.card')
		.filter({ has: page.getByText('Расходы', { exact: true }) })
		.first();
	const countCard = page
		.locator('.card')
		.filter({ has: page.getByText('Операций', { exact: true }) })
		.first();
	await expect(expenseCard).toContainText('12.50');
	await expect(expenseCard).not.toContainText('40.00');
	await expect(countCard.locator('.tabular-nums')).toHaveText('1');
});

test('repeat after editing a receipt expense does not send that store id', async ({ page }) => {
	const tag = Date.now();
	const account = await createCashAccount(page, `E2E Repeat ${tag}`);
	const cats = await apiJSON<{ id: string; is_system?: boolean }[]>(
		page,
		'GET',
		'/api/v1/categories?type=expense'
	);
	const cat = cats.find((row) => !row.is_system) ?? cats[0];
	expect(cat).toBeTruthy();

	await enableReceiptScannerFeatures(page);
	const fiscalDoc = String(tag).slice(-6);
	await apiJSON(page, 'POST', '/api/v1/receipts/import', {
		account_id: account.id,
		fiscal_drive_number: '7288940700999001',
		fiscal_document_number: fiscalDoc,
		fiscal_sign: String(tag),
		receipt: {
			fiscal_drive_number: '7288940700999001',
			fiscal_document_number: fiscalDoc,
			fiscal_sign: String(tag),
			date_time: '2026-01-15T10:30:00Z',
			total_sum: 15000,
			store_name: `E2E Точка ${tag}`,
			items: [{ name: `E2E Позиция ${tag}`, quantity: 1, price: 15000, sum: 15000 }],
			fns_check_url: 'https://proverkacheka.nalog.ru/?fn=7288940700999001'
		},
		items: [{ index: 0, category_id: cat.id, include: true }]
	});
	await apiJSON(page, 'POST', '/api/v1/transactions', {
		account_id: account.id,
		type: 'expense',
		amount: '9.00',
		category_id: cat.id,
		description: `E2E без точки ${tag}`,
		transaction_date: formatUTCDateTime(new Date(Date.now() - 3_600_000))
	});

	const posts: Array<Record<string, unknown>> = [];
	await page.route('**/api/v1/transactions', async (route) => {
		if (route.request().method() === 'POST') {
			posts.push(route.request().postDataJSON() as Record<string, unknown>);
		}
		await route.continue();
	});

	await page.goto('/transactions');
	await waitAppReady(page);
	await selectCombobox(page, 'tx-filter-account', { label: account.name });

	const receiptRow = page.getByRole('row', { name: new RegExp(`E2E Позиция ${tag}`) });
	await expect(receiptRow).toBeVisible();
	await rowMenuAction(page, receiptRow, 'Изменить');
	const dialog = page.getByRole('dialog');
	await expect(dialog).toBeVisible();
	await expect(dialog.locator('#tx-store')).toHaveCount(0);
	await page.keyboard.press('Escape');
	await expect(dialog).toHaveCount(0);

	const plainRow = page.getByRole('row', { name: new RegExp(`E2E без точки ${tag}`) });
	await rowMenuAction(page, plainRow, 'Повторить');
	await expect(dialog).toBeVisible();
	await dialog.getByRole('button', { name: 'Сохранить' }).click();
	await expect(dialog).toHaveCount(0, { timeout: 15_000 });

	expect(posts).toHaveLength(1);
	expect(posts[0]).not.toHaveProperty('store_id');

	await rowMenuAction(page, receiptRow, 'Повторить');
	await expect(dialog).toBeVisible();
	await dialog.getByRole('button', { name: 'Сохранить' }).click();
	await expect(dialog).toHaveCount(0, { timeout: 15_000 });

	expect(posts).toHaveLength(2);
	expect(posts[1]).not.toHaveProperty('store_id');
});
