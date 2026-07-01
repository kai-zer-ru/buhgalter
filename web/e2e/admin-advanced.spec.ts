import { test, expect } from '@playwright/test';
import { apiJSON, deleteAdminUserByLogin, restoreAdminSession, waitAppReady } from './helpers/auth';
import { confirmDialog, rowMenuAction } from './helpers/ui';

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
	await rowMenuAction(page, row, 'Удалить');
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
	await rowMenuAction(page, row, 'Сбросить пароль');
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
	await rowMenuAction(page, row, 'Удалить');
	await confirmDialog(page);
	await expect(page.getByRole('cell', { name: 'e2eadminreset' })).toHaveCount(0, {
		timeout: 10_000
	});
});

test('registration: pending user redirected to login', async ({ page }) => {
	await apiJSON(page, 'PUT', '/api/v1/admin/settings', {
		registration_enabled: true,
		external_url: ''
	});

	await deleteAdminUserByLogin(page, 'e2epending');
	await page.context().clearCookies();
	await page.goto('/register');
	await waitAppReady(page);
	await page.getByLabel('Логин').fill('e2epending');
	await page.getByLabel('Пароль', { exact: true }).fill('userpass1');
	await page.getByLabel('Подтверждение пароля').fill('userpass1');
	await page.getByRole('button', { name: 'Создать аккаунт' }).click();

	await expect(page).toHaveURL(/\/login/, { timeout: 10_000 });
	await expect(page.getByText('Регистрация принята')).toBeVisible();

	await restoreAdminSession(page);
});

test('admin: pending user banner opens moderation modal', async ({ page }) => {
	await restoreAdminSession(page);
	await apiJSON(page, 'PUT', '/api/v1/admin/settings', {
		registration_enabled: true,
		external_url: ''
	});

	await deleteAdminUserByLogin(page, 'e2emoderate');

	const reg = await page.request.post('/api/v1/auth/register', {
		data: {
			login: 'e2emoderate',
			password: 'userpass1',
			password_confirm: 'userpass1',
			display_name: 'E2E Moderate'
		}
	});
	expect(reg.ok(), `register e2emoderate failed: ${reg.status()}`).toBeTruthy();

	const users = await apiJSON<{ id: string; login: string; status: string }[]>(
		page,
		'GET',
		'/api/v1/admin/users'
	);
	expect(users.some((u) => u.login === 'e2emoderate' && u.status === 'pending')).toBeTruthy();

	await page.goto('/');
	await page.reload();
	await waitAppReady(page);

	await expect(page.getByText('E2E Moderate')).toBeVisible({ timeout: 15_000 });
	const banner = page.locator('div.rounded-xl.border').filter({ hasText: 'E2E Moderate' });
	await expect(banner).toBeVisible({ timeout: 5_000 });
	await banner.getByRole('button', { name: 'Модерировать' }).click();

	const modal = page.getByRole('dialog');
	await expect(modal).toBeVisible();
	await expect(page).toHaveURL(/moderate=/);
	await modal.getByRole('button', { name: 'Активировать' }).click();
	await expect(modal).toHaveCount(0, { timeout: 10_000 });
	await expect(
		page.locator('div.rounded-xl.border').filter({ hasText: 'E2E Moderate' })
	).toHaveCount(0, { timeout: 10_000 });
});

test('admin: activate pending user and login', async ({ page }) => {
	await page.goto('/admin/users');
	await waitAppReady(page);

	const row = page.getByRole('row', { name: /e2epending/ });
	await expect(row).toBeVisible({ timeout: 10_000 });
	await rowMenuAction(page, row, 'Активировать');
	await expect(row.getByRole('cell', { name: 'Активен' })).toBeVisible();

	await page.goto('/login');
	await waitAppReady(page);
	await page.getByLabel('Логин').fill('e2epending');
	await page.getByLabel('Пароль', { exact: true }).fill('userpass1');
	await page.getByRole('button', { name: 'Войти' }).click();
	await expect(page).toHaveURL('/', { timeout: 15_000 });
});

test('login: banned user sees human-readable error', async ({ page }) => {
	await page.goto('/admin/users');
	await waitAppReady(page);

	const row = page.getByRole('row', { name: /e2epending/ });
	if ((await row.count()) === 0) return;
	await rowMenuAction(page, row, 'Заблокировать');
	await confirmDialog(page, 'Заблокировать');

	await page.goto('/login');
	await waitAppReady(page);
	await page.getByLabel('Логин').fill('e2epending');
	await page.getByLabel('Пароль', { exact: true }).fill('userpass1');
	await page.getByRole('button', { name: 'Войти' }).click();

	await expect(page.getByText('заблокирована', { exact: false })).toBeVisible({ timeout: 10_000 });
	await expect(page.getByText('USER_BANNED')).toHaveCount(0);
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

test('admin: support links visible in settings tab', async ({ page }) => {
	await page.goto('/settings?tab=admin&admin_tab=system');
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

test('admin: support links visible on /admin routes', async ({ page }) => {
	await page.goto('/admin');
	await waitAppReady(page);

	await expect(page.getByRole('link', { name: 'Поддержать проект' })).toBeVisible();
	await expect(page.getByRole('link', { name: 'Репозиторий' })).toBeVisible();
});
