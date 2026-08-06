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

	// SPA client crash (failed chunk / render) shows SvelteKit's default error inside the shell.
	// Header is still visible, so without this check later locators time out for 60s.
	const crashed = page.getByRole('heading', { name: '500', exact: true });
	if (await crashed.isVisible().catch(() => false)) {
		await page.reload();
		await expect(page.getByText('Загрузка…')).toHaveCount(0, { timeout: 20_000 });
		await expect(page.locator('header')).toBeVisible({ timeout: 20_000 });
		await expect(crashed).toHaveCount(0, { timeout: 15_000 });
	}

	const { dismissBlockingModals } = await import('./ui');
	await dismissBlockingModals(page);
}

/** First-run admin via API — avoids UI/i18n races on /setup (flake → login 401). */
export async function completeSetupIfNeeded(page: Page) {
	const statusRes = await page.request.get('/api/v1/setup/status');
	expect(statusRes.ok(), `setup status failed: ${statusRes.status()}`).toBeTruthy();
	const status = (await statusRes.json()) as { configured?: boolean };
	if (status.configured) return;

	const res = await page.request.post('/api/v1/setup', {
		data: {
			admin_login: ADMIN.login,
			admin_display_name: ADMIN.displayName,
			admin_password: ADMIN.password,
			admin_password_confirm: ADMIN.password,
			registration_enabled: false,
			external_url: ''
		}
	});
	// 409: already configured between status check and POST.
	if (res.status() === 409) return;
	expect(res.ok(), `API setup failed: ${res.status()} ${await res.text()}`).toBeTruthy();
}

async function fillLoginForm(page: Page, loginName: string, password: string) {
	await waitAppReady(page);
	const submit = page.getByRole('button', { name: 'Войти' });
	await expect(submit).toBeVisible({ timeout: 20_000 });
	await page.locator('#login').fill(loginName);
	await page.locator('#password').fill(password);
	await submit.click();
}

export async function login(page: Page) {
	await page.goto('/login');
	await fillLoginForm(page, ADMIN.login, ADMIN.password);
	await expect(page).toHaveURL(/\/(\?.*)?$/, { timeout: 15_000 });
	await waitAppReady(page);
}

/** Cookie session via API — preferred for setup / restore (no UI race). */
export async function loginViaAPI(page: Page): Promise<void> {
	let lastStatus = 0;
	let lastBody = '';
	for (let attempt = 0; attempt < 5; attempt++) {
		const res = await page.request.post('/api/v1/auth/login', {
			data: { login: ADMIN.login, password: ADMIN.password }
		});
		lastStatus = res.status();
		if (res.ok()) {
			if (isAdminSession(await currentSession(page))) return;
		} else {
			lastBody = await res.text().catch(() => '');
		}
		await page.waitForTimeout(150 * (attempt + 1));
	}
	expect(lastStatus, `API login failed: ${lastStatus}${lastBody ? ` ${lastBody}` : ''}`).toBe(200);
	expect(
		isAdminSession(await currentSession(page)),
		'API login did not yield admin session'
	).toBeTruthy();
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
	await fillLoginForm(page, ADMIN.login, ADMIN.password);
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

/** Drop web SWR localStorage after API mutations done outside the UI client. */
export async function clearBrowserRefCache(page: Page): Promise<void> {
	await page.evaluate(() => {
		const keys: string[] = [];
		for (let i = 0; i < localStorage.length; i++) {
			const key = localStorage.key(i);
			if (key?.includes('ref_cache')) keys.push(key);
		}
		for (const key of keys) localStorage.removeItem(key);
	});
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
	if (!response.ok()) {
		const detail = await response.text().catch(() => '');
		expect(
			response.ok(),
			`API ${method} ${path} failed with ${response.status()}${detail ? `: ${detail}` : ''}`
		).toBeTruthy();
	}
	if (method !== 'GET') {
		await clearBrowserRefCache(page).catch(() => {
			// page may not have localStorage yet (about:blank)
		});
	}
	if (response.status() === 204) return undefined as T;
	return (await response.json()) as T;
}
