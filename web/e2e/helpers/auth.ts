import { expect, type Page } from '@playwright/test';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));

export const authFile = path.join(__dirname, '..', '.auth', 'admin.json');

export const ADMIN = {
	login: 'admin',
	password: 'secret123',
	displayName: 'E2E Admin'
};

export async function waitAppReady(page: Page) {
	await expect(page.getByText('Загрузка…')).toHaveCount(0, { timeout: 20_000 });
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
	await waitAppReady(page);
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
