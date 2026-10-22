import { test, expect } from '@playwright/test';
import { apiJSON, formatUTCDateTime } from './helpers/auth';
import { createCashAccount } from './helpers/setup-data';
import { selectLabeledCombobox } from './helpers/combobox';

test('consumer credit principal income toggle updates account balance', async ({ page }) => {
	const account = await createCashAccount(page);
	const before = await apiJSON<{ balance: number }>(page, 'GET', `/api/v1/accounts/${account.id}`);
	const principal = 250_000;
	const issue = new Date();

	const credit = await apiJSON<{ id: string; principal_transaction_id: string | null }>(
		page,
		'POST',
		'/api/v1/credits',
		{
			name: `E2E Principal Income ${Date.now()}`,
			credit_kind: 'consumer',
			principal_amount: '2500.00',
			issue_date: formatUTCDateTime(issue),
			term_months: 6,
			interest_rate: 12,
			payment_interval: 'month',
			debit_account_id: account.id,
			principal_affects_balance: true,
			create_transactions: false
		}
	);

	expect(credit.principal_transaction_id).toBeTruthy();
	const after = await apiJSON<{ balance: number }>(page, 'GET', `/api/v1/accounts/${account.id}`);
	expect(after.balance).toBe(before.balance + principal);
});

test('principal income toggle visible only for consumer credit form', async ({ page }) => {
	await page.goto('/credits');
	await page.getByRole('button', { name: 'Новый кредит' }).click();
	const modal = page.getByRole('dialog');
	await expect(modal.getByText('Учитывать доход в балансе')).toBeVisible();

	await modal.getByRole('radio', { name: 'Рассрочка' }).check();
	await expect(modal.getByText('Учитывать доход в балансе')).not.toBeVisible();

	await modal.getByRole('radio', { name: 'Кредит' }).check();
	await expect(modal.getByText('Учитывать доход в балансе')).toBeVisible();
});

test('credit form defaults auto-debit time to 08:00', async ({ page }) => {
	await page.goto('/credits');
	await page.getByRole('button', { name: 'Новый кредит' }).click();
	const modal = page.getByRole('dialog');
	await expect(modal.locator('input[type="time"]')).toHaveValue('08:00');
});

test('principal income toggle enabled when schedule payments are in the future', async ({
	page
}) => {
	const account = await createCashAccount(page);
	await page.goto('/credits');
	await page.getByRole('button', { name: 'Новый кредит' }).click();
	const modal = page.getByRole('dialog');
	await modal.getByRole('textbox', { name: 'Сумма кредита' }).fill('50000');
	await selectLabeledCombobox(page, 'Счёт списания', { label: account.name });
	await expect(modal.getByText(/Сумма:\s*\d/)).toBeVisible({ timeout: 15_000 });
	// «Уже платил по графику» показывается только при прошлых платежах в графике.
	await expect(modal.getByRole('switch', { name: 'Уже платил по графику' })).toHaveCount(0);
	const toggle = modal.getByRole('switch', { name: 'Учитывать доход в балансе' });
	await expect(toggle).toBeEnabled();
	await expect(
		modal.getByText('Изменение баланса невозможно: в графике есть платёж с датой в прошлом')
	).toHaveCount(0);
});

test('principal income API rejected when schedule has past payment', async ({ page }) => {
	const account = await createCashAccount(page);
	const issue = new Date();
	issue.setFullYear(issue.getFullYear() - 2);
	const res = await page.request.post('/api/v1/credits', {
		data: {
			credit_kind: 'consumer',
			principal_amount: '3000.00',
			issue_date: formatUTCDateTime(issue),
			term_months: 6,
			interest_rate: 12,
			payment_interval: 'month',
			debit_account_id: account.id,
			principal_affects_balance: true,
			create_transactions: false
		}
	});
	expect(res.status()).toBe(400);
	const body = (await res.json()) as { error?: { code?: string; message?: string } };
	expect(body.error?.code).toBe('VALIDATION_ERROR');
	expect(body.error?.message).toContain('ERR_CREDIT_PRINCIPAL_INCOME_PAST_PAYMENT');
});
