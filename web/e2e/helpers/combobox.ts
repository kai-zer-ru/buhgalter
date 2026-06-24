import { expect, type Page } from '@playwright/test';

/** Custom Select/Combobox (`role="combobox"` + `#id-list` listbox). */
export async function selectCombobox(
	page: Page,
	id: string,
	option: { label?: string; index?: number }
) {
	const trigger = page.locator(`#${id}`);
	await trigger.click();
	const list = page.locator(`#${id}-list`);
	await expect(list).toBeVisible();

	if (option.label !== undefined) {
		await list.getByRole('button', { name: option.label, exact: true }).click();
		return;
	}
	if (option.index !== undefined) {
		await list.getByRole('button').nth(option.index).click();
		return;
	}
	throw new Error('selectCombobox: pass label or index');
}
