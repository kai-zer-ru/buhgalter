import { expect, test } from '@playwright/test';
import { apiJSON, waitAppReady } from './helpers/auth';

test('opening a searchable select does not scroll the page', async ({ page }) => {
	await apiJSON(page, 'POST', '/api/v1/merchants', {
		name: `E2E Scroll ${Date.now()}`,
		icon: 'default'
	});
	// A short viewport puts the dropdown search field below the fold.
	await page.setViewportSize({ width: 1150, height: 600 });
	await page.goto('/stats');
	await waitAppReady(page);

	const before = await page.evaluate(() => window.scrollY);
	await page.locator('#stats-merchant').click();
	await expect(page.locator('#stats-merchant-list')).toBeVisible();
	await expect(page.locator('#stats-merchant-search')).toBeFocused();
	expect(await page.evaluate(() => window.scrollY)).toBe(before);
});
