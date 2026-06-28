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
	await page.getByRole('button', { name: 'Импортировать' }).click();
	await confirmDialog(page, 'Импортировать');
	await expect(page.getByRole('heading', { name: 'Импорт завершён' })).toBeVisible({
		timeout: 30_000
	});
}
