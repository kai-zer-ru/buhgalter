import { expect, test } from '@playwright/test';
import { apiJSON, formatUTCDateTime } from './helpers/auth';
import {
	expectPageLoadsWithin,
	MAX_PAGE_LOAD_MS,
	measurePageLoad,
	warmRoute
} from './helpers/perf';

const AUTHENTICATED_ROUTES = [
	'/',
	'/accounts',
	'/accounts/new',
	'/transactions',
	'/stats',
	'/credits',
	'/debts',
	'/recurring-operations',
	'/settings',
	'/settings?tab=password',
	'/settings?tab=tokens',
	'/settings?tab=notifications',
	'/settings?tab=categories',
	'/settings?tab=import',
	'/settings?tab=admin&admin_tab=system',
	'/settings?tab=admin&admin_tab=users',
	'/settings?tab=admin&admin_tab=backups',
	'/settings?tab=admin&admin_tab=diagnostics',
	'/admin',
	'/admin/users',
	'/admin/backups',
	'/admin/diagnostics'
];

const PUBLIC_ROUTES = ['/login', '/register'];

test.describe.configure({ mode: 'serial' });

test('warm up server caches before perf checks', async ({ page }) => {
	for (const route of AUTHENTICATED_ROUTES) {
		await warmRoute(page, route);
	}
	await page.context().clearCookies();
	for (const route of PUBLIC_ROUTES) {
		await warmRoute(page, route);
	}
	await warmRoute(page, '/docs');
});

test(`authenticated routes load within ${MAX_PAGE_LOAD_MS}ms`, async ({ page }) => {
	const unique = Date.now();
	const account = await apiJSON<{ id: string }>(page, 'POST', '/api/v1/accounts', {
		name: `E2E Perf ${unique}`,
		type: 'cash',
		initial_balance: '1000.00'
	});

	const now = new Date();
	const credit = await apiJSON<{ id: string }>(page, 'POST', '/api/v1/credits', {
		name: `E2E Perf Credit ${unique}`,
		principal_amount: '6000.00',
		issue_date: formatUTCDateTime(new Date(now.getTime() - 24 * 60 * 60 * 1000)),
		term_months: 3,
		interest_rate: 0,
		payment_interval: 'month',
		debit_account_id: account.id,
		added_retroactively: false,
		create_transactions: true
	});

	const debt = await apiJSON<{ debtor_id: string }>(page, 'POST', '/api/v1/debts', {
		debtor_name: `E2E Perf Debtor ${unique}`,
		direction: 'lent',
		amount: '500.00',
		debt_date: formatUTCDateTime(now),
		due_date: formatUTCDateTime(new Date(now.getTime() + 7 * 24 * 60 * 60 * 1000)),
		affects_balance: false,
		account_id: account.id
	});

	const routes = [
		...AUTHENTICATED_ROUTES,
		`/accounts/${account.id}`,
		`/credits/${credit.id}`,
		`/debtors/${debt.debtor_id}`,
		'/import',
		'/settings/categories'
	];

	for (const route of routes) {
		const elapsed = await measurePageLoad(page, route);
		expect.soft(elapsed, `${route} took ${elapsed}ms`).toBeLessThanOrEqual(MAX_PAGE_LOAD_MS);
	}
});

test(`public routes load within ${MAX_PAGE_LOAD_MS}ms`, async ({ page }) => {
	await page.context().clearCookies();

	for (const route of PUBLIC_ROUTES) {
		await expectPageLoadsWithin(page, route);
	}

	await expectPageLoadsWithin(page, '/docs');
});

test('legacy import redirect loads within limit', async ({ page }) => {
	const elapsed = await measurePageLoad(page, '/import');
	expect(elapsed).toBeLessThanOrEqual(MAX_PAGE_LOAD_MS);
	await expect(page).toHaveURL(/\/settings\?tab=import/);
});
