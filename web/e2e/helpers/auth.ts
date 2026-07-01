import { expect, type Page } from '@playwright/test';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));

export const authFile = path.join(__dirname, '..', '.auth', 'admin.json');

export const ADMIN = {
	login: 'admin',
	password: 'secret123',
	displayName: 'E2E Admin'
};

function isPublicAppPath(pathname: string): boolean {
	return pathname === '/login' || pathname === '/register' || pathname === '/setup';
}

export async function waitAppReady(page: Page) {
	await expect(page.getByText('Загрузка…')).toHaveCount(0, { timeout: 20_000 });
	const pathname = new URL(page.url()).pathname;
	if (isPublicAppPath(pathname)) {
		await expect(page.locator('h1').first()).toBeVisible({ timeout: 20_000 });
		return;
	}
	await expect(page.locator('header')).toBeVisible({ timeout: 20_000 });
	const { dismissBlockingModals } = await import('./ui');
	await dismissBlockingModals(page);
}

export async function completeSetupIfNeeded(page: Page) {
	await page.goto('/setup');
	await expect(page.getByText('Загрузка…')).toHaveCount(0, { timeout: 20_000 });

	const displayName = page.locator('#display-name');
	if (!(await displayName.isVisible())) return;

	await displayName.fill(ADMIN.displayName);
	await page.locator('#login').fill(ADMIN.login);
	await page.locator('#password').fill(ADMIN.password);
	await page.locator('#password-confirm').fill(ADMIN.password);
	await page.getByRole('button', { name: 'Завершить настройку' }).click();
	await page.waitForURL('**/login**', { timeout: 15_000 });
}

export async function login(page: Page) {
	await page.goto('/login');
	await page.locator('#login').fill(ADMIN.login);
	await page.locator('#password').fill(ADMIN.password);
	await page.getByRole('button', { name: 'Войти' }).click();
	await expect(page).toHaveURL(/\/(\?.*)?$/, { timeout: 15_000 });
	await waitAppReady(page);
}

type MeResponse = { login: string; is_admin: boolean };

async function currentSession(page: Page): Promise<MeResponse | null> {
	const me = await page.request.get('/api/v1/auth/me');
	if (!me.ok()) return null;
	return (await me.json()) as MeResponse;
}

function isAdminSession(user: MeResponse | null): boolean {
	return user?.login === ADMIN.login && user.is_admin === true;
}

/** Restore admin cookies for page.request (after clearCookies or logout). */
export async function restoreAdminSession(page: Page) {
	if (isAdminSession(await currentSession(page))) return;

	await page.request.post('/api/v1/auth/logout').catch(() => {});

	const res = await page.request.post('/api/v1/auth/login', {
		data: { login: ADMIN.login, password: ADMIN.password }
	});
	if (res.ok() && isAdminSession(await currentSession(page))) return;

	if (fs.existsSync(authFile)) {
		const state = JSON.parse(fs.readFileSync(authFile, 'utf-8')) as {
			cookies?: Array<{
				name: string;
				value: string;
				domain: string;
				path: string;
				expires?: number;
				httpOnly?: boolean;
				secure?: boolean;
				sameSite?: 'Strict' | 'Lax' | 'None';
			}>;
		};
		if (state.cookies?.length) {
			await page.context().addCookies(state.cookies);
		}
		if (isAdminSession(await currentSession(page))) return;
	}

	await page.goto('/login');
	await page.locator('#login').fill(ADMIN.login);
	await page.locator('#password').fill(ADMIN.password);
	await page.getByRole('button', { name: 'Войти' }).click();
	await expect(page).toHaveURL(/\/(\?.*)?$/, { timeout: 15_000 });
	await waitAppReady(page);

	expect(isAdminSession(await currentSession(page)), 'admin session restore failed').toBeTruthy();
}

export async function deleteAdminUserByLogin(page: Page, login: string) {
	await restoreAdminSession(page);
	const usersRes = await page.request.get('/api/v1/admin/users');
	if (!usersRes.ok()) return;
	const users = (await usersRes.json()) as { id: string; login: string }[];
	const target = users.find((u) => u.login === login);
	if (!target) return;
	await page.request.delete(`/api/v1/admin/users/${target.id}`);
}

export function formatUTCDateTime(date: Date): string {
	const iso = new Date(date.getTime() - date.getMilliseconds()).toISOString();
	return iso.slice(0, 19).replace('T', ' ');
}

export async function apiJSON<T>(
	page: Page,
	method: 'GET' | 'POST' | 'PUT' | 'DELETE' | 'PATCH',
	path: string,
	body?: unknown
): Promise<T> {
	const response = await page.request.fetch(path, {
		method,
		data: body ?? undefined
	});
	expect(response.ok(), `API ${method} ${path} failed with ${response.status()}`).toBeTruthy();
	if (response.status() === 204) return undefined as T;
	return (await response.json()) as T;
}
