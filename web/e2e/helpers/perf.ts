import { expect, type Page } from '@playwright/test';
import { waitAppReady } from './auth';

/** Max time from navigation start until the page is interactive. */
export const MAX_PAGE_LOAD_MS = 1000;

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

export async function expectPageLoadsWithin(
	page: Page,
	route: string,
	maxMs = MAX_PAGE_LOAD_MS
): Promise<void> {
	const elapsed = await measurePageLoad(page, route);
	expect(elapsed, `${route} loaded in ${elapsed}ms (limit ${maxMs}ms)`).toBeLessThanOrEqual(maxMs);
}

export async function warmRoute(page: Page, route: string) {
	await page.goto(route);
	await waitForRouteReady(page, route);
}
