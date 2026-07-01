import { test } from '@playwright/test';
import { apiJSON, formatUTCDateTime } from './helpers/auth';
import {
	checkPageLoadWithin,
	MAX_PAGE_LOAD_MS,
	measurePageLoad,
	warnIfPageLoadSlow,
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
	'/settings/import',
	'/settings',
	'/settings/password',
	'/settings/tokens',
	'/settings/notifications',
	'/settings/categories',
	'/settings/recurring-operations',
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

test(`authenticated routes page load (warn if > ${MAX_PAGE_LOAD_MS}ms)`, async ({ page }) => {
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
		`/debtors/${debt.debtor_id}`
	];

	for (const route of routes) {
		const elapsed = await measurePageLoad(page, route);
		warnIfPageLoadSlow(elapsed, route);
	}
});

test(`public routes page load (warn if > ${MAX_PAGE_LOAD_MS}ms)`, async ({ page }) => {
	await page.context().clearCookies();

	for (const route of PUBLIC_ROUTES) {
		await checkPageLoadWithin(page, route);
	}

	await checkPageLoadWithin(page, '/docs');
});
