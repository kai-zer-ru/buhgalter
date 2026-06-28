import { expect, type Page } from '@playwright/test';
import { waitAppReady } from './auth';

/** Target max time from navigation start until the page is interactive. Exceeding logs a warning only. */
export const MAX_PAGE_LOAD_MS = 1000;

export function warnIfPageLoadSlow(elapsed: number, label: string, maxMs = MAX_PAGE_LOAD_MS): void {
	if (elapsed > maxMs) {
		console.warn(`[perf] ${label} took ${elapsed}ms (limit ${maxMs}ms)`);
	}
}

async function waitForRouteReady(page: Page, route: string) {
	if (route === '/docs') {
		await expect(page.locator('redoc')).toBeVisible({ timeout: 20_000 });
		return;
	}
	if (route === '/login' || route === '/register') {
		await expect(page.getByText('Загрузка…')).toHaveCount(0, { timeout: 20_000 });
		return;
	}
	await waitAppReady(page);
}

export async function measurePageLoad(page: Page, route: string): Promise<number> {
	const started = Date.now();
	await page.goto(route);
	await waitForRouteReady(page, route);
	return Date.now() - started;
}

export async function checkPageLoadWithin(
	page: Page,
	route: string,
	maxMs = MAX_PAGE_LOAD_MS
): Promise<number> {
	const elapsed = await measurePageLoad(page, route);
	warnIfPageLoadSlow(elapsed, route, maxMs);
	return elapsed;
}

export async function warmRoute(page: Page, route: string) {
	await page.goto(route);
	await waitForRouteReady(page, route);
}
