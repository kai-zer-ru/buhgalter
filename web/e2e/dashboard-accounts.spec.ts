import { test, expect } from '@playwright/test';
import { apiJSON, waitAppReady } from './helpers/auth';
import { createCashAccount, createCreditCardAccount } from './helpers/setup-data';

test('home: account groups collapsed on mobile by default', async ({ page }) => {
	const unique = Date.now();
	const account = await apiJSON<{ name: string }>(page, 'POST', '/api/v1/accounts', {
		name: `E2E Mobile Accounts ${unique}`,
		type: 'cash',
		initial_balance: '500.00'
	});

	await page.setViewportSize({ width: 390, height: 844 });
	await page.goto('/');
	await waitAppReady(page);

	const panel = page.locator('details.account-group-panel').filter({ hasText: 'Мои средства' });
	const summary = panel.locator('summary.account-group-summary');
	await expect(summary).toBeVisible({ timeout: 10_000 });

	const link = page.getByRole('link', { name: account.name });
	await expect(link).toBeHidden();

	await summary.click();
	await expect(link).toBeVisible();

	await summary.click();
	await expect(link).toBeHidden();
});

test('home: recent transactions collapsed by default', async ({ page }) => {
	const account = await createCashAccount(page, `E2E Recent Collapsed ${Date.now()}`);
	await apiJSON(page, 'POST', '/api/v1/transactions', {
		account_id: account.id,
		type: 'expense',
		amount: '10.00',
		description: `E2E recent ${Date.now()}`,
		transaction_date: new Date().toISOString().slice(0, 19).replace('T', ' ')
	});

	await page.goto('/');
	await waitAppReady(page);

	const panel = page
		.locator('details.account-group-panel')
		.filter({ hasText: 'Последние операции' });
	await expect(panel).not.toHaveAttribute('open');
	await expect(panel.getByRole('row')).toHaveCount(0);

	await panel.locator('summary.account-group-summary').click();
	await expect(panel).toHaveAttribute('open', '');
	await expect(panel.getByRole('row').first()).toBeVisible({ timeout: 10_000 });
});

test('home: account group spoilers keep the same gap as other spoilers', async ({ page }) => {
	const unique = Date.now();
	await createCashAccount(page, `E2E Gap Cash ${unique}`);
	await createCreditCardAccount(page, `E2E Gap Card ${unique}`);

	// Mobile width: every home spoiler starts collapsed.
	await page.setViewportSize({ width: 390, height: 844 });
	await page.goto('/');
	await waitAppReady(page);

	const panel = (label: string) =>
		page.locator('details.account-group-panel').filter({ hasText: label }).first();
	const own = panel('Мои средства');
	const credit = panel('Кредитные средства');
	const recent = panel('Последние операции');
	await expect(credit).toBeVisible({ timeout: 10_000 });

	const box = async (loc: typeof own) => {
		const b = await loc.boundingBox();
		if (!b) throw new Error('panel has no box');
		return b;
	};
	const [a, b, c] = [await box(own), await box(credit), await box(recent)];
	const groupGap = b.y - (a.y + a.height);
	const sectionGap = c.y - (b.y + b.height);
	expect(groupGap).toBeGreaterThan(0);
	expect(Math.abs(groupGap - sectionGap)).toBeLessThanOrEqual(1);
});
