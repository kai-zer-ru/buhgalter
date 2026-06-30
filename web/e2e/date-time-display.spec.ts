import { test, expect } from '@playwright/test';
import { apiJSON, waitAppReady } from './helpers/auth';
import { createCashAccount } from './helpers/setup-data';
import { selectCombobox } from './helpers/transactions';

function parseApiUtc(s: string): Date {
	const m = s.match(/^(\d{4})-(\d{2})-(\d{2}) (\d{2}):(\d{2}):(\d{2})$/);
	if (m) return new Date(Date.UTC(+m[1], +m[2] - 1, +m[3], +m[4], +m[5], +m[6]));
	const d = new Date(s);
	if (Number.isNaN(d.getTime())) throw new Error(`invalid datetime ${s}`);
	return d;
}

function formatInTimezone(utc: Date, timezone: string, withSeconds: boolean): string {
	const formatter = new Intl.DateTimeFormat('en-GB', {
		timeZone: timezone,
		day: '2-digit',
		month: '2-digit',
		year: 'numeric',
		hour: '2-digit',
		minute: '2-digit',
		...(withSeconds ? { second: '2-digit' } : {}),
		hour12: false
	});
	const parts: Record<string, string> = {};
	for (const p of formatter.formatToParts(utc)) {
		if (p.type !== 'literal') parts[p.type] = p.value;
	}
	const time = withSeconds
		? `${parts.hour}:${parts.minute}:${parts.second}`
		: `${parts.hour}:${parts.minute}`;
	return `${parts.day}.${parts.month}.${parts.year} ${time}`;
}

/** Mirrors formatAPIOperationDateTimeForDisplay for e2e expectations. */
function expectedOperationDateTime(apiUtc: string, timezone: string): string {
	return formatInTimezone(parseApiUtc(apiUtc), timezone, false);
}

/** Mirrors formatAPIDateTimeForDisplay for e2e expectations. */
function expectedDateTimeWithSeconds(apiUtc: string, timezone: string): string {
	return formatInTimezone(parseApiUtc(apiUtc), timezone, true);
}

test('transaction list shows date-time without seconds', async ({ page }) => {
	const me = await apiJSON<{ timezone: string }>(page, 'GET', '/api/v1/auth/me');
	const tz = me.timezone || 'Europe/Moscow';
	const account = await createCashAccount(page, `E2E DateFmt ${Date.now()}`);
	const apiDate = '2026-06-15 05:30:45';
	const desc = `E2E date format ${Date.now()}`;

	await apiJSON(page, 'POST', '/api/v1/transactions', {
		account_id: account.id,
		type: 'expense',
		amount: '12.34',
		description: desc,
		transaction_date: apiDate
	});

	const expected = expectedOperationDateTime(apiDate, tz);

	await page.goto('/transactions');
	await waitAppReady(page);
	await selectCombobox(page, 'tx-filter-account', { label: account.name });

	const row = page.getByRole('row', { name: new RegExp(`${account.name}.*12\\.34`) });
	await expect(row).toBeVisible({ timeout: 10_000 });
	await expect(row.getByText(expected)).toBeVisible();
	expect(expected).toMatch(/^\d{2}\.\d{2}\.\d{4} \d{2}:\d{2}$/);
	expect(expected).not.toMatch(/^\d{2}\.\d{2}\.\d{4} \d{2}:\d{2}:\d{2}$/);
});

test('admin diagnostics shows build_time in display format', async ({ page }) => {
	const me = await apiJSON<{ timezone: string }>(page, 'GET', '/api/v1/auth/me');
	const tz = me.timezone || 'Europe/Moscow';
	const diagnostics = await apiJSON<{ build_time: string }>(
		page,
		'GET',
		'/api/v1/admin/diagnostics'
	);
	expect(diagnostics.build_time).toBeTruthy();

	const expected = expectedDateTimeWithSeconds(diagnostics.build_time, tz);

	await page.goto('/admin/diagnostics');
	await waitAppReady(page);

	const buildTimeRow = page.getByRole('row', { name: /build_time/ });
	await expect(buildTimeRow.getByText(expected)).toBeVisible();
	expect(expected).toMatch(/^\d{2}\.\d{2}\.\d{4} \d{2}:\d{2}:\d{2}$/);
});
