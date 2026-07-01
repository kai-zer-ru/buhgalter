import { expect, test } from '@playwright/test';
import { apiJSON, formatUTCDateTime, waitAppReady } from './helpers/auth';

const AUTHENTICATED_ROUTES = [
	'/',
	'/accounts',
	'/accounts/new',
	'/transactions',
	'/stats',
	'/credits',
	'/debts',
	'/settings',
	'/settings/password',
	'/settings/tokens',
	'/settings/notifications',
	'/settings/categories',
	'/settings/import',
	'/settings/recurring-operations',
	'/admin',
	'/admin/users',
	'/admin/backups',
	'/admin/diagnostics'
];

test.describe('public', () => {
	test.beforeEach(async ({ page }) => {
		await page.context().clearCookies();
	});

	test('login and setup pages open', async ({ page }) => {
		await page.goto('/login');
		await expect(page.getByRole('button', { name: 'Войти' })).toBeVisible();

		await page.goto('/setup');
		await expect(page.getByText('Загрузка…')).toHaveCount(0, { timeout: 20_000 });

		await page.goto('/register');
		await expect(page.getByText('Загрузка…')).toHaveCount(0, { timeout: 20_000 });
	});

	test('API docs open without auth', async ({ page }) => {
		const response = await page.goto('/docs');
		expect(response?.ok()).toBeTruthy();
		await expect(page.locator('redoc')).toBeVisible({ timeout: 20_000 });
	});

	test('cold start at / redirects to login', async ({ page }) => {
		await page.goto('/');
		await expect(page.getByText('Загрузка…')).toHaveCount(0, { timeout: 20_000 });
		await expect(page).toHaveURL(/\/login/);
	});
});

test.describe('authenticated pages', () => {
	test('all main routes open', async ({ page }) => {
		const unique = Date.now();
		const account = await apiJSON<{ id: string }>(page, 'POST', '/api/v1/accounts', {
			name: `E2E Pages ${unique}`,
			type: 'cash',
			initial_balance: '1000.00'
		});

		const now = new Date();
		const credit = await apiJSON<{ id: string }>(page, 'POST', '/api/v1/credits', {
			name: `E2E Credit ${unique}`,
			principal_amount: '12000.00',
			issue_date: formatUTCDateTime(new Date(now.getTime() - 24 * 60 * 60 * 1000)),
			term_months: 6,
			interest_rate: 0,
			payment_interval: 'month',
			debit_account_id: account.id,
			added_retroactively: false,
			create_transactions: true
		});

		const debt = await apiJSON<{ debtor_id: string }>(page, 'POST', '/api/v1/debts', {
			debtor_name: `E2E Debtor ${unique}`,
			direction: 'lent',
			amount: '1000.00',
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
			await page.goto(route);
			await waitAppReady(page);
			await expect(page).not.toHaveURL(/\/login/);
		}
	});
});
