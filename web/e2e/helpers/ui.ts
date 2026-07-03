import { expect, type Locator, type Page } from '@playwright/test';

export type ToastType = 'success' | 'error' | 'warning' | 'info';

function toastRole(type: ToastType): 'alert' | 'status' {
	return type === 'error' || type === 'warning' ? 'alert' : 'status';
}

export function toastLocator(page: Page, type: ToastType, text?: string): Locator {
	let loc = page.getByRole(toastRole(type));
	if (text) loc = loc.filter({ hasText: text });
	return loc;
}

export async function expectToast(page: Page, type: ToastType, text: string, timeout = 10_000) {
	await expect(toastLocator(page, type, text)).toBeVisible({ timeout });
}

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

/** Expand a top-level `CollapsibleSection` / `AccountGroupPanel` by summary label. */
export async function expandCollapsibleSection(page: Page, label: string | RegExp) {
	const panel = page.locator('details.account-group-panel').filter({ hasText: label });
	const summary = panel.locator('summary.account-group-summary');
	await expect(summary).toBeVisible();
	if (!(await panel.evaluate((el) => (el as HTMLDetailsElement).open))) {
		await summary.click();
	}
	return panel;
}
