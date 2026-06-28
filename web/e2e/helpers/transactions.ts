import { expect, type Page } from '@playwright/test';
import { selectCombobox, selectLabeledCombobox } from './combobox';

export async function fillTransactionForm(
	page: Page,
	opts: {
		amount: string;
		account: string;
		categoryIndex?: number;
	}
) {
	const dialog = page.getByRole('dialog');
	await expect(dialog).toBeVisible();
	await selectCombobox(page, 'tx-account', { label: opts.account });
	await selectCombobox(page, 'tx-category', { index: opts.categoryIndex ?? 0 });
	const amountInput = dialog.locator('#tx-amount');
	await amountInput.click();
	await amountInput.fill(opts.amount);
	await expect(amountInput).not.toHaveValue('');
	await dialog.getByRole('button', { name: 'Сохранить' }).click();
	await expect(dialog).toHaveCount(0, { timeout: 15_000 });
}

export { selectCombobox, selectLabeledCombobox };
