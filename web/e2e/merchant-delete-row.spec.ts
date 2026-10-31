import { expect, test, type Page, type Route } from '@playwright/test';
import { apiJSON, waitAppReady } from './helpers/auth';
import { confirmDialog, expectToast, rowMenuAction } from './helpers/ui';

function merchantCard(page: Page, name: string) {
	return page.getByTestId('merchant-row').filter({
		has: page.locator('.font-medium').getByText(name, { exact: true })
	});
}

async function createMerchant(page: Page, name: string) {
	await apiJSON(page, 'POST', '/api/v1/merchants', { name, icon: 'default' });
}

async function watchLoading(page: Page) {
	await page.evaluate(() => {
		const flag = window as Window & { __sawMerchantLoading?: boolean };
		flag.__sawMerchantLoading = false;
		const note = () => {
			// innerText skips scripts, so the i18n catalog does not count as a loading flash.
			if (document.body.innerText.includes('Загрузка…')) flag.__sawMerchantLoading = true;
		};
		note();
		const observer = new MutationObserver(note);
		observer.observe(document.body, { subtree: true, childList: true, characterData: true });
	});
}

async function sawLoading(page: Page) {
	return page.evaluate(
		() => (window as Window & { __sawMerchantLoading?: boolean }).__sawMerchantLoading === true
	);
}

async function merchantsRefSnapshot(page: Page): Promise<{ name: string }[] | null> {
	return page.evaluate(() => {
		const suffix = '::/api/v1/merchants';
		for (let i = 0; i < localStorage.length; i++) {
			const key = localStorage.key(i);
			if (!key?.endsWith(suffix)) continue;
			const raw = localStorage.getItem(key);
			if (!raw) return null;
			return JSON.parse(raw) as { name: string }[];
		}
		return null;
	});
}

function isMerchantList(url: URL) {
	return url.pathname === '/api/v1/merchants';
}

function isMerchantItem(url: URL) {
	return /^\/api\/v1\/merchants\/[^/]+$/.test(url.pathname);
}

/** Hold DELETE until released so the list can be checked while the request is in flight. */
async function holdMerchantDeletes(page: Page) {
	let release: () => void = () => {};
	const gate = new Promise<void>((resolve) => {
		release = resolve;
	});
	let released = false;
	await page.route(isMerchantItem, async (route: Route) => {
		if (route.request().method() !== 'DELETE') {
			await route.continue();
			return;
		}
		if (!released) await gate;
		await route.continue();
	});
	return () => {
		released = true;
		release();
	};
}

test('deleting a merchant below the fold keeps scroll and the neighbor', async ({ page }) => {
	const tag = Date.now();
	const prefix = `DelRow ${tag}`;
	for (let i = 0; i < 28; i++) {
		await createMerchant(page, `${prefix} ${String(i).padStart(2, '0')}`);
	}
	const keepName = `${prefix} 10`;

	let listGets = 0;
	await page.route(isMerchantList, async (route) => {
		if (route.request().method() === 'GET') listGets += 1;
		await route.continue();
	});
	const releaseDelete = await holdMerchantDeletes(page);

	await page.goto('/merchants');
	await waitAppReady(page);
	await expect(merchantCard(page, keepName)).toBeVisible();
	const getsAfterLoad = listGets;

	await merchantCard(page, keepName).evaluate((el) => {
		const top = el.getBoundingClientRect().top + window.scrollY;
		window.scrollTo(0, Math.max(0, top - 160));
	});
	const beforeY = await page.evaluate(() => window.scrollY);
	expect(beforeY).toBeGreaterThan(200);
	const topBefore = await merchantCard(page, keepName).evaluate(
		(el) => el.getBoundingClientRect().top
	);
	const dropName = await page.evaluate(
		({ prefix, keepName }) => {
			const rows = [...document.querySelectorAll('.font-medium')].map((el) => {
				const box = el.getBoundingClientRect();
				return { name: (el.textContent ?? '').trim(), top: box.top, bottom: box.bottom };
			});
			const visible = rows.find(
				(row) =>
					row.name.startsWith(`${prefix} `) &&
					row.name !== keepName &&
					row.top > 200 &&
					row.bottom < window.innerHeight - 8
			);
			return visible?.name ?? null;
		},
		{ prefix, keepName }
	);
	if (!dropName) throw new Error('no visible merchant below the neighbor');
	await watchLoading(page);

	try {
		await rowMenuAction(page, merchantCard(page, dropName), 'Удалить');
		await confirmDialog(page, 'Удалить');
		await expect(page.getByText('Загрузка…')).toHaveCount(0);
		const duringY = await page.evaluate(() => window.scrollY);
		expect(Math.abs(duringY - beforeY)).toBeLessThanOrEqual(1);
		await expect(merchantCard(page, dropName)).toBeVisible();
		await expect(merchantCard(page, keepName)).toBeVisible();
	} finally {
		releaseDelete();
	}

	await expect(page.locator('.font-medium').getByText(dropName, { exact: true })).toHaveCount(0);
	await expect(merchantCard(page, keepName)).toBeVisible();
	const afterY = await page.evaluate(() => window.scrollY);
	const topAfter = await merchantCard(page, keepName).evaluate(
		(el) => el.getBoundingClientRect().top
	);
	expect(Math.abs(afterY - beforeY)).toBeLessThanOrEqual(1);
	expect(Math.abs(topAfter - topBefore)).toBeLessThanOrEqual(1);
	expect(listGets).toBe(getsAfterLoad);
	expect(await sawLoading(page)).toBe(false);
	await expectToast(page, 'success', 'Удалено');
});

test('search text stays and the deleted row leaves the filtered list', async ({ page }) => {
	const tag = Date.now();
	const query = `Srch${tag}`;
	const alpha = `${query} Alpha`;
	const beta = `${query} Beta`;
	const other = `Other${tag} Zebra`;
	await createMerchant(page, alpha);
	await createMerchant(page, beta);
	await createMerchant(page, other);

	await page.goto('/merchants');
	await waitAppReady(page);
	const search = page.getByPlaceholder('Поиск…');
	await search.fill(query);
	await expect(merchantCard(page, alpha)).toBeVisible();
	await expect(merchantCard(page, beta)).toBeVisible();
	await expect(page.locator('.font-medium').getByText(other, { exact: true })).toHaveCount(0);
	await watchLoading(page);

	await rowMenuAction(page, merchantCard(page, alpha), 'Удалить');
	await confirmDialog(page, 'Удалить');

	await expect(search).toHaveValue(query);
	await expect(page.locator('.font-medium').getByText(alpha, { exact: true })).toHaveCount(0);
	await expect(merchantCard(page, beta)).toBeVisible();
	await expect(page.locator('.font-medium').getByText(other, { exact: true })).toHaveCount(0);
	await search.fill('');
	await expect(merchantCard(page, other)).toBeVisible();
	await expect(page.locator('.font-medium').getByText(alpha, { exact: true })).toHaveCount(0);
	const names = (await merchantsRefSnapshot(page))?.map((row) => row.name) ?? [];
	expect(names).toContain(beta);
	expect(names).toContain(other);
	expect(names).not.toContain(alpha);
	await expect(page.getByText('Загрузка…')).toHaveCount(0);
	expect(await sawLoading(page)).toBe(false);
	await expectToast(page, 'success', 'Удалено');
});

test('deleting the last merchant shows the empty state without a loading flash', async ({
	page
}) => {
	const tag = Date.now();
	const name = `Последний ${tag}`;
	const only = {
		id: `e2e-last-${tag}`,
		name,
		icon: 'default',
		aliases: [] as string[],
		created_at: '2026-01-01T00:00:00Z'
	};
	let listGets = 0;
	await page.route(isMerchantList, async (route) => {
		if (route.request().method() !== 'GET') {
			await route.continue();
			return;
		}
		listGets += 1;
		// A later GET still returns the deleted row. Reloading the list would paint it again.
		await route.fulfill({
			status: 200,
			contentType: 'application/json',
			body: JSON.stringify([only])
		});
	});
	await page.route(isMerchantItem, async (route) => {
		if (route.request().method() === 'DELETE') {
			await route.fulfill({ status: 204 });
			return;
		}
		await route.continue();
	});

	await page.goto('/merchants');
	await waitAppReady(page);
	await expect(merchantCard(page, name)).toBeVisible();
	const getsAfterLoad = listGets;
	await watchLoading(page);

	await rowMenuAction(page, merchantCard(page, name), 'Удалить');
	await confirmDialog(page, 'Удалить');

	await expect(page.getByText('Магазинов пока нет')).toBeVisible();
	await expect(page.locator('.font-medium').getByText(name, { exact: true })).toHaveCount(0);
	await expect(page.getByText('Загрузка…')).toHaveCount(0);
	expect(await sawLoading(page)).toBe(false);
	expect(listGets).toBe(getsAfterLoad);
	await expectToast(page, 'success', 'Удалено');
});

test('a failed delete leaves the merchant row in place', async ({ page }) => {
	const tag = Date.now();
	const name = `FailDel ${tag}`;
	const created = await apiJSON<{ id: string }>(page, 'POST', '/api/v1/merchants', {
		name,
		icon: 'default'
	});

	await page.goto('/merchants');
	await waitAppReady(page);
	await expect(merchantCard(page, name)).toBeVisible();
	await watchLoading(page);
	await page.route(
		(url) => url.pathname === `/api/v1/merchants/${created.id}`,
		async (route) => {
			if (route.request().method() !== 'DELETE') {
				await route.continue();
				return;
			}
			await route.fulfill({
				status: 500,
				contentType: 'application/json',
				body: JSON.stringify({
					error: { code: 'INTERNAL_ERROR', message: 'Магазин не удалён' }
				})
			});
		}
	);

	await rowMenuAction(page, merchantCard(page, name), 'Удалить');
	await confirmDialog(page, 'Удалить');

	await expectToast(page, 'error', 'Магазин не удалён');
	await expect(merchantCard(page, name)).toBeVisible();
	await expect(page.getByText('Загрузка…')).toHaveCount(0);
	expect(await sawLoading(page)).toBe(false);
});
