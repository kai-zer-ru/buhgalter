import { expect, type Page } from '@playwright/test';
import { confirmDialog } from './ui';

export async function advanceImportToPreview(page: Page) {
	const preview = page.getByRole('heading', { name: 'Готово к импорту' });
	for (let i = 0; i < 10; i++) {
		if (await preview.isVisible()) return;
		const next = page.getByRole('button', { name: 'Далее' });
		await expect(next).toBeVisible({ timeout: 20_000 });
		await next.click();
	}
	await expect(preview).toBeVisible({ timeout: 20_000 });
}

export async function commitImportFromPreview(page: Page) {
	const jobCreated = page.waitForResponse(
		(res) =>
			res.url().includes('/api/v1/import/jobs') &&
			!res.url().match(/\/import\/jobs\/[^/]+$/) &&
			res.request().method() === 'POST',
		{ timeout: 20_000 }
	);
	await page.getByRole('button', { name: 'Импортировать' }).click();
	await confirmDialog(page, 'Импортировать');
	const created = await jobCreated;
	expect(created.ok(), `POST /import/jobs → ${created.status()}`).toBeTruthy();

	await expect(page.getByRole('heading', { name: 'Импорт завершён' })).toBeVisible({
		timeout: 30_000
	});
}
