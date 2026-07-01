import { test, expect } from '@playwright/test';
import { apiJSON, waitAppReady } from './helpers/auth';
import { confirmDialog } from './helpers/ui';

test.describe.configure({ mode: 'serial' });

test('admin: reset password modal closes on cancel from notification link', async ({ page }) => {
	await page.goto('/admin/users');
	await waitAppReady(page);

	await page.getByLabel('Логин').fill('e2eresetcancel');
	await page.getByLabel('Пароль', { exact: true }).fill('userpass1');
	await page.getByLabel('Подтверждение пароля').fill('userpass1');
	await page.getByRole('button', { name: 'Создать' }).click();
	await expect(page.getByRole('cell', { name: 'e2eresetcancel' }).first()).toBeVisible({
		timeout: 10_000
	});

	const users = await apiJSON<{ id: string; login: string }[]>(page, 'GET', '/api/v1/admin/users');
	const target = users.find((u) => u.login === 'e2eresetcancel');
	expect(target).toBeTruthy();

	await page.goto(`/admin/users?reset=${target!.id}`);
	await waitAppReady(page);

	const modal = page.getByRole('dialog');
	await expect(modal).toBeVisible();
	await modal.getByRole('button', { name: 'Отмена' }).click();
	await expect(modal).toHaveCount(0, { timeout: 10_000 });
	await expect(page).not.toHaveURL(/reset=/);

	const row = page.getByRole('row', { name: /e2eresetcancel/ });
	await row.getByRole('button', { name: 'Удалить' }).click();
	await confirmDialog(page);
});

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

test('admin system page loads for admin user', async ({ page }) => {
	await page.goto('/admin');
	await waitAppReady(page);
	await expect(page.getByRole('heading', { name: 'Система', level: 1 })).toBeVisible();
});

test('admin: support links visible on /admin', async ({ page }) => {
	await page.goto('/admin');
	await waitAppReady(page);

	const support = page.getByRole('link', { name: 'Поддержать проект' });
	const repository = page.getByRole('link', { name: 'Репозиторий' });
	await expect(support).toBeVisible();
	await expect(repository).toBeVisible();
	await expect(support).toHaveAttribute('href', /tbank\.ru/);
	await expect(repository).toHaveAttribute('href', 'https://github.com/kai-zer-ru/buhgalter');
	await expect(support).toHaveAttribute('target', '_blank');
	await expect(repository).toHaveAttribute('target', '_blank');
});

test('admin: support links visible on subpages', async ({ page }) => {
	await page.goto('/admin/users');
	await waitAppReady(page);

	await expect(page.getByRole('link', { name: 'Поддержать проект' })).toBeVisible();
	await expect(page.getByRole('link', { name: 'Репозиторий' })).toBeVisible();
});
