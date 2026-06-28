import { test, expect } from '@playwright/test';
import { waitAppReady } from './helpers/auth';
import { confirmDialog } from './helpers/ui';

test.describe.configure({ mode: 'serial' });

test('admin: reset user password', async ({ page }) => {
	await page.goto('/admin/users');
	await waitAppReady(page);

	await page.getByLabel('Логин').fill('e2eadminreset');
	await page.getByLabel('Пароль', { exact: true }).fill('userpass1');
	await page.getByLabel('Подтверждение пароля').fill('userpass1');
	await page.getByRole('button', { name: 'Создать' }).click();
	await expect(page.getByRole('cell', { name: 'e2eadminreset' }).first()).toBeVisible({
		timeout: 10_000
	});

	const row = page.getByRole('row', { name: /e2eadminreset/ });
	await row.getByRole('button', { name: 'Сбросить пароль' }).click();
	const modal = page.getByRole('dialog');
	await modal.locator('input[type="password"]').first().fill('newpass99');
	await modal.locator('input[type="password"]').nth(1).fill('newpass99');
	await modal.getByRole('button', { name: 'Сохранить' }).click();
	await expect(modal).toHaveCount(0, { timeout: 10_000 });
});

test('admin: delete test user', async ({ page }) => {
	await page.goto('/admin/users');
	await waitAppReady(page);

	const row = page.getByRole('row', { name: /e2eadminreset/ });
	if ((await row.count()) === 0) return;
	await row.getByRole('button', { name: 'Удалить' }).click();
	await confirmDialog(page);
	await expect(page.getByRole('cell', { name: 'e2eadminreset' })).toHaveCount(0, {
		timeout: 10_000
	});
});

test('admin: save backup schedule settings', async ({ page }) => {
	await page.goto('/admin/backups');
	await waitAppReady(page);

	await page.locator('#retention').fill('14');
	await page.getByRole('button', { name: 'Сохранить' }).click();
	await expect(page.getByText('Настройки сохранены').first()).toBeVisible({ timeout: 10_000 });
});

test('admin: diagnostics copy button', async ({ page }) => {
	await page.goto('/admin/diagnostics');
	await waitAppReady(page);

	const copyBtn = page.getByRole('button', { name: 'Скопировать для отчёта' });
	await expect(copyBtn).toBeVisible();
	await copyBtn.click();
});

test('admin system tab loads for admin user', async ({ page }) => {
	await page.goto('/settings?tab=admin&admin_tab=system');
	await waitAppReady(page);
	await expect(page.getByRole('tab', { name: 'Система', exact: true })).toHaveAttribute(
		'aria-selected',
		'true'
	);
});
