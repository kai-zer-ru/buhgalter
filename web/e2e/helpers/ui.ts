import { expect, type Locator, type Page } from '@playwright/test';

export async function dismissBlockingModals(page: Page) {
	const update = page.getByRole('dialog').getByRole('button', { name: 'Понятно' });
	if (await update.isVisible().catch(() => false)) {
		await update.click();
		await expect(update).toHaveCount(0, { timeout: 5_000 });
	}
}

export async function confirmDialog(page: Page, confirmLabel = 'Удалить') {
	const dialog = page.getByRole('alertdialog');
	await expect(dialog).toBeVisible();
	await dialog.getByRole('button', { name: confirmLabel, exact: true }).click();
	await expect(dialog).toHaveCount(0, { timeout: 15_000 });
}

export async function openRowActions(row: Locator) {
	await row.getByRole('button', { name: 'Действия' }).click();
}

export async function clickMenuItem(page: Page, name: string) {
	await page.getByRole('menuitem', { name, exact: true }).click();
}

export async function rowMenuAction(page: Page, row: Locator, action: string) {
	await openRowActions(row);
	await clickMenuItem(page, action);
}
