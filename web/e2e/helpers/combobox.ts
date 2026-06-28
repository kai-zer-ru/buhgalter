import { expect, type Page } from '@playwright/test';

async function pickComboboxOption(
	page: Page,
	trigger: ReturnType<Page['locator']>,
	option: { label?: string; index?: number }
) {
	await trigger.click();
	const id = await trigger.getAttribute('id');
	const list = id ? page.locator(`#${id}-list`) : page.getByRole('listbox').first();
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

/** Custom Select/Combobox (`role="combobox"` + `#id-list` listbox). */
export async function selectCombobox(
	page: Page,
	id: string,
	option: { label?: string; index?: number }
) {
	await pickComboboxOption(page, page.locator(`#${id}`), option);
}

/** Combobox located by its visible label text (for forms with duplicate default ids). */
export async function selectLabeledCombobox(
	page: Page,
	labelText: string,
	option: { label?: string; index?: number }
) {
	const field = page.locator('label', { hasText: labelText }).first().locator('..');
	await pickComboboxOption(page, field.getByRole('combobox'), option);
}
