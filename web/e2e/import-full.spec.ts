import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { test, expect } from '@playwright/test';
import { waitAppReady } from './helpers/auth';
import { createCashAccount } from './helpers/setup-data';

const __dirname = path.dirname(fileURLToPath(import.meta.url));

async function ensureImportAccount(page: import('@playwright/test').Page) {
	const response = await page.request.get('/api/v1/accounts');
	const accounts = (await response.json()) as { name: string }[];
	if (!accounts.some((account) => account.name === 'E2E Cash')) {
		await createCashAccount(page, 'E2E Cash');
	}
}

test('import CSV wizard reaches mapping step', async ({ page }) => {
	await ensureImportAccount(page);
	await page.goto('/settings?tab=import');
	await waitAppReady(page);

	const fileInput = page.locator('input[type="file"]');
	await fileInput.setInputFiles(path.join(__dirname, 'fixtures', 'sample.csv'));
	await expect(page.getByText('sample.csv')).toBeVisible({ timeout: 10_000 });
	await page.getByRole('button', { name: 'Далее' }).click();

	await expect(
		page.getByText(/Сопоставление счетов|Сопоставление колонок|Готово к импорту/)
	).toBeVisible({
		timeout: 20_000
	});
});

test('export CSV download button is enabled with filters', async ({ page }) => {
	await page.goto('/settings?tab=import');
	await waitAppReady(page);
	await page.getByRole('tab', { name: 'Экспорт', exact: true }).click();

	const downloadBtn = page.getByRole('button', { name: 'Скачать CSV' });
	await expect(downloadBtn).toBeVisible();
	await expect(downloadBtn).toBeEnabled();
});
