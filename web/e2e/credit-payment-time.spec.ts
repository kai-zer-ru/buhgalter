import { test, expect, type Page } from '@playwright/test';
import { apiJSON, formatUTCDateTime, waitAppReady } from './helpers/auth';
import { createCashAccount, createExpense } from './helpers/setup-data';
import { expectToast, openRowActions } from './helpers/ui';

function parseApiUtc(s: string): Date {
	const m = s.match(/^(\d{4})-(\d{2})-(\d{2})[ T](\d{2}):(\d{2})(?::(\d{2}))?/);
	if (m) return new Date(Date.UTC(+m[1], +m[2] - 1, +m[3], +m[4], +m[5], +(m[6] ?? 0)));
	const d = new Date(s);
	if (Number.isNaN(d.getTime())) throw new Error(`invalid datetime ${s}`);
	return d;
}

function localParts(apiUtc: string, timezone: string): { date: string; time: string } {
	const formatter = new Intl.DateTimeFormat('en-GB', {
		timeZone: timezone,
		day: '2-digit',
		month: '2-digit',
		year: 'numeric',
		hour: '2-digit',
		minute: '2-digit',
		hourCycle: 'h23'
	});
	const parts: Record<string, string> = {};
	for (const p of formatter.formatToParts(parseApiUtc(apiUtc))) {
		if (p.type !== 'literal') parts[p.type] = p.value;
	}
	return {
		date: `${parts.day}.${parts.month}.${parts.year}`,
		time: `${parts.hour}:${parts.minute}`
	};
}

test('credit payment menu changes clock and keeps the calendar day', async ({ page }) => {
	const account = await createCashAccount(page);
	const me = await apiJSON<{ timezone: string }>(page, 'GET', '/api/v1/auth/me');
	const tz = me.timezone || 'Europe/Moscow';
	const creditName = `E2E Clock ${Date.now()}`;
	const credit = await apiJSON<{ id: string }>(page, 'POST', '/api/v1/credits', {
		name: creditName,
		principal_amount: '3000.00',
		issue_date: formatUTCDateTime(new Date(Date.now() - 40 * 24 * 60 * 60 * 1000)),
		term_months: 3,
		interest_rate: 0,
		payment_interval: 'month',
		debit_account_id: account.id,
		debit_time_local: '08:00',
		added_retroactively: false,
		create_transactions: false
	});
	const detail = await apiJSON<{ monthly_payment_display: string }>(
		page,
		'GET',
		`/api/v1/credits/${credit.id}`
	);
	await apiJSON(page, 'POST', `/api/v1/credits/${credit.id}/payments`, {
		amount: detail.monthly_payment_display,
		payment_date: formatUTCDateTime(new Date())
	});
	const listed = await apiJSON<{ data: { id: string; transaction_date: string }[] }>(
		page,
		'GET',
		`/api/v1/transactions?search=${encodeURIComponent(creditName)}`
	);
	const tx = listed.data[0];
	expect(tx, 'credit payment transaction').toBeTruthy();
	const before = localParts(tx.transaction_date, tz);
	const nextTime = before.time === '15:45' ? '09:10' : '15:45';

	await page.goto('/transactions');
	await waitAppReady(page);
	const row = page.locator('details[open] table tbody tr').filter({ hasText: creditName });
	await expect(row).toBeVisible({ timeout: 15_000 });

	await openRowActions(row);
	await expect(page.getByRole('menuitem', { name: 'Сделать подпиской', exact: true })).toHaveCount(
		0
	);
	await expect(
		page.getByRole('menuitem', { name: 'Прикрепить к подписке', exact: true })
	).toHaveCount(0);
	await expect(page.getByRole('menuitem', { name: 'Изменить', exact: true })).toHaveCount(0);
	await expect(page.getByRole('menuitem', { name: 'Изменить время', exact: true })).toBeVisible();
	await expect(page.getByRole('menuitem', { name: 'Удалить', exact: true })).toBeVisible();
	await page.getByRole('menuitem', { name: 'Изменить время', exact: true }).click();

	const dialog = page.getByRole('dialog');
	await expect(dialog.locator('input[type="time"]')).toHaveValue(before.time);
	await dialog.locator('input[type="time"]').fill(nextTime);
	await dialog.getByRole('button', { name: 'Сохранить' }).click();
	await expect(dialog).toHaveCount(0, { timeout: 10_000 });
	await expect(row).toContainText(`${before.date} ${nextTime}`);

	const creditAfter = await apiJSON<{ debit_time_local?: string | null }>(
		page,
		'GET',
		`/api/v1/credits/${credit.id}`
	);
	expect(creditAfter.debit_time_local).toBe('08:00');
	const txAfter = await apiJSON<{ transaction_date: string }>(
		page,
		'GET',
		`/api/v1/transactions/${tx.id}`
	);
	const after = localParts(txAfter.transaction_date, tz);
	expect(after.date).toBe(before.date);
	expect(after.time).toBe(nextTime);
});

test('ordinary expense keeps subscription actions and has no clock item', async ({ page }) => {
	const account = await createCashAccount(page);
	const description = `E2E Plain ${Date.now()}`;
	await createExpense(page, account.id, '12.00', description);

	await page.goto('/transactions');
	await waitAppReady(page);
	const row = page.locator('details[open] table tbody tr').filter({ hasText: description });
	await expect(row).toBeVisible({ timeout: 15_000 });
	await openRowActions(row);
	await expect(
		page.getByRole('menuitem', { name: 'Сделать подпиской', exact: true })
	).toBeVisible();
	await expect(
		page.getByRole('menuitem', { name: 'Прикрепить к подписке', exact: true })
	).toBeVisible();
	await expect(page.getByRole('menuitem', { name: 'Изменить время', exact: true })).toHaveCount(0);
});

test('rejected clock change keeps the row time and shows a toast', async ({ page }) => {
	const seeded = await seedCreditPayment(page);
	await page.goto('/transactions');
	await waitAppReady(page);
	const row = page.locator('details[open] table tbody tr').filter({ hasText: seeded.creditName });
	await expect(row).toBeVisible({ timeout: 15_000 });
	await expect(row).toContainText(`${seeded.before.date} ${seeded.before.time}`);

	await openRowActions(row);
	await page.getByRole('menuitem', { name: 'Изменить время', exact: true }).click();
	await page.route('**/api/v1/transactions/*/time', async (route) => {
		if (route.request().method() !== 'PUT') {
			await route.continue();
			return;
		}
		await route.fulfill({
			status: 400,
			contentType: 'application/json',
			body: JSON.stringify({
				error: {
					code: 'VALIDATION_ERROR',
					message: 'Некорректное время (ожидается ЧЧ:ММ)'
				}
			})
		});
	});

	const dialog = page.getByRole('dialog');
	await dialog.locator('input[type="time"]').fill(seeded.nextTime);
	await dialog.getByRole('button', { name: 'Сохранить' }).click();
	await expectToast(page, 'error', 'Некорректное время (ожидается ЧЧ:ММ)');
	await expect(dialog).toBeVisible();
	await expect(row).toContainText(`${seeded.before.date} ${seeded.before.time}`);
	await expect(row).not.toContainText(seeded.nextTime);
});

async function seedCreditPayment(page: Page) {
	const account = await createCashAccount(page);
	const me = await apiJSON<{ timezone: string }>(page, 'GET', '/api/v1/auth/me');
	const tz = me.timezone || 'Europe/Moscow';
	const creditName = `E2E Clock Reject ${Date.now()}`;
	const credit = await apiJSON<{ id: string }>(page, 'POST', '/api/v1/credits', {
		name: creditName,
		principal_amount: '3000.00',
		issue_date: formatUTCDateTime(new Date(Date.now() - 40 * 24 * 60 * 60 * 1000)),
		term_months: 3,
		interest_rate: 0,
		payment_interval: 'month',
		debit_account_id: account.id,
		debit_time_local: '08:00',
		added_retroactively: false,
		create_transactions: false
	});
	const detail = await apiJSON<{ monthly_payment_display: string }>(
		page,
		'GET',
		`/api/v1/credits/${credit.id}`
	);
	await apiJSON(page, 'POST', `/api/v1/credits/${credit.id}/payments`, {
		amount: detail.monthly_payment_display,
		payment_date: formatUTCDateTime(new Date())
	});
	const listed = await apiJSON<{ data: { id: string; transaction_date: string }[] }>(
		page,
		'GET',
		`/api/v1/transactions?search=${encodeURIComponent(creditName)}`
	);
	const tx = listed.data[0];
	expect(tx, 'credit payment transaction').toBeTruthy();
	const before = localParts(tx.transaction_date, tz);
	const nextTime = before.time === '15:45' ? '09:10' : '15:45';
	return { creditName, before, nextTime };
}
