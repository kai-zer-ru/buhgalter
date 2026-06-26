import { expect, test, type Page } from '@playwright/test';

const ADMIN = {
	login: 'admin',
	password: 'secret123',
	displayName: 'E2E Admin'
};

function formatUTCDateTime(date: Date): string {
	const iso = new Date(date.getTime() - date.getMilliseconds()).toISOString();
	return iso.slice(0, 19).replace('T', ' ');
}

async function waitAppReady(page: Page) {
	await expect(page.getByText('Загрузка…')).toHaveCount(0, { timeout: 20_000 });
	await expect(page.locator('header')).toBeVisible({ timeout: 20_000 });
}

async function completeSetupIfNeeded(page: Page) {
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

async function login(page: Page) {
	await page.goto('/login');
	await page.locator('#login').fill(ADMIN.login);
	await page.locator('#password').fill(ADMIN.password);
	await page.getByRole('button', { name: 'Войти' }).click();
	await waitAppReady(page);
}

async function apiJSON<T>(
	page: Page,
	method: 'GET' | 'POST',
	path: string,
	body?: unknown
): Promise<T> {
	const response = await page.request.fetch(path, {
		method,
		data: body ?? undefined
	});
	expect(response.ok(), `API ${method} ${path} failed with ${response.status()}`).toBeTruthy();
	return (await response.json()) as T;
}

test('all main authenticated pages open', async ({ page }) => {
	await completeSetupIfNeeded(page);
	await login(page);

	const unique = Date.now();
	const account = await apiJSON<{ id: string }>(page, 'POST', '/api/v1/accounts', {
		name: `E2E Smoke ${unique}`,
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
		'/',
		'/accounts',
		'/accounts/new',
		`/accounts/${account.id}`,
		'/transactions',
		'/stats',
		'/credits',
		`/credits/${credit.id}`,
		'/debts',
		`/debtors/${debt.debtor_id}`,
		'/recurring-operations',
		'/settings',
		'/settings/categories',
		'/import',
		'/admin',
		'/admin/users',
		'/admin/backups',
		'/admin/diagnostics'
	];

	for (const route of routes) {
		await page.goto(route);
		await waitAppReady(page);
		await expect(page).not.toHaveURL(/\/login/);
	}
});

test('public pages open', async ({ page }) => {
	await page.goto('/login');
	await expect(page.getByRole('button', { name: 'Войти' })).toBeVisible();

	await page.goto('/setup');
	await expect(page.getByText('Загрузка…')).toHaveCount(0, { timeout: 20_000 });

	await page.goto('/register');
	await expect(page.getByText('Загрузка…')).toHaveCount(0, { timeout: 20_000 });
});
