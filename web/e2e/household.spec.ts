import { test, expect } from '@playwright/test';
import {
	apiJSON,
	deleteAdminUserByLogin,
	loginViaAPI,
	restoreAdminSession,
	waitAppReady
} from './helpers/auth';
import { confirmDialog, expectToast, rowMenuAction } from './helpers/ui';
import { createCashAccount, createExpense, createTransfer } from './helpers/setup-data';

test.describe.configure({ mode: 'serial' });

test('household: create, invite, accept by link, kick', async ({ page }) => {
	const tag = Date.now();
	const memberLogin = `e2ehouse${tag}`;
	const memberPass = 'Housepass1';

	await restoreAdminSession(page);
	await apiJSON(page, 'POST', '/api/v1/admin/users', {
		login: memberLogin,
		password: memberPass,
		password_confirm: memberPass,
		display_name: 'E2E House Member',
		is_admin: false
	});

	try {
		const existing = await apiJSON<{ household: { id: string } | null }>(
			page,
			'GET',
			'/api/v1/households/me'
		);
		if (existing.household) {
			await apiJSON(page, 'POST', '/api/v1/households/me/dissolve', { confirm_members: true });
		}

		await page.goto('/settings/household');
		await waitAppReady(page);
		await expect(page.getByRole('heading', { name: 'Семья / команда' }).first()).toBeVisible();
		await page.getByTestId('household-create-family').click();
		await expectToast(page, 'success', 'Семья создана');
		await expect(page.getByText('Владелец')).toBeVisible();

		await page.getByTestId('household-invite-create').click();
		const dialog = page.getByRole('dialog');
		await expect(dialog).toBeVisible();
		const codeText = (await dialog.getByTestId('household-invite-dialog-code').innerText()).trim();
		expect(codeText).toMatch(/^[A-Z0-9]{3}-[A-Z0-9]{3}-[A-Z0-9]{3}$/);
		await dialog.getByRole('button', { name: 'Закрыть' }).click();
		await expect(dialog).toBeHidden();
		await rowMenuAction(page, page.getByTestId('household-invite-row'), 'Показать QR');
		await expect(dialog).toBeVisible();
		await expect(dialog.getByTestId('household-invite-dialog-code')).toHaveText(codeText);
		await expect(dialog.getByRole('img', { name: 'QR-код приглашения' })).toBeVisible();
		await dialog.getByRole('button', { name: 'Закрыть' }).click();

		await page.context().clearCookies();
		await page.goto(`/invite/${codeText}`);
		await expect(page).toHaveURL(/\/login\?next=/, { timeout: 15_000 });
		await waitAppReady(page);
		await page.locator('#login').fill(memberLogin);
		await page.locator('#password').fill(memberPass);
		await page.getByRole('button', { name: 'Войти' }).click();
		await expect(page).toHaveURL(new RegExp(`/invite/${codeText.replace(/-/g, '\\-')}`), {
			timeout: 15_000
		});
		await waitAppReady(page);
		await expect(page.getByTestId('household-join')).toBeVisible({ timeout: 15_000 });
		await page.getByTestId('household-join').click();
		await expectToast(page, 'success', 'Вы вступили в семью');
		await expect(page).toHaveURL(/\/settings\/household/);
		await expect(page.getByTestId('household-member-row')).toHaveCount(2);

		await page.context().clearCookies();
		await loginViaAPI(page);
		await page.goto('/settings/household');
		await waitAppReady(page);
		const memberRow = page.getByTestId('household-member-row').filter({ hasText: memberLogin });
		await rowMenuAction(page, memberRow, 'Исключить');
		await confirmDialog(page, 'Исключить');
		await expectToast(page, 'success', 'Участник исключён');
		await expect(page.getByTestId('household-member-row')).toHaveCount(1);
	} finally {
		await restoreAdminSession(page);
		const me = await page.request.get('/api/v1/households/me');
		if (me.ok()) {
			const body = (await me.json()) as { household: { id: string } | null };
			if (body.household) {
				await page.request.post('/api/v1/households/me/dissolve', {
					data: { confirm_members: true }
				});
			}
		}
		await deleteAdminUserByLogin(page, memberLogin);
	}
});

test('household: shared account, author, mask, leave, dissolve', async ({ page, browser }) => {
	test.setTimeout(120_000);
	const tag = Date.now();
	const memberLogin = `e2eshare${tag}`;
	const memberPass = 'Sharepass1';
	const familyName = `E2E Family ${tag}`;
	const personalName = `E2E Personal ${tag}`;

	await restoreAdminSession(page);
	await apiJSON(page, 'POST', '/api/v1/admin/users', {
		login: memberLogin,
		password: memberPass,
		password_confirm: memberPass,
		display_name: 'E2E House Member',
		is_admin: false
	});

	const memberContext = await browser.newContext();
	const memberPage = await memberContext.newPage();

	try {
		const existing = await apiJSON<{ household: { id: string } | null }>(
			page,
			'GET',
			'/api/v1/households/me'
		);
		if (existing.household) {
			await apiJSON(page, 'POST', '/api/v1/households/me/dissolve', { confirm_members: true });
		}

		await page.goto('/settings/household');
		await waitAppReady(page);
		await page.getByTestId('household-create-family').click();
		await expectToast(page, 'success', 'Семья создана');
		await page.getByTestId('household-invite-create').click();
		const dialog = page.getByRole('dialog');
		const codeText = (await dialog.getByTestId('household-invite-dialog-code').innerText()).trim();
		await dialog.getByRole('button', { name: 'Закрыть' }).click();

		const loginRes = await memberPage.request.post('/api/v1/auth/login', {
			data: { login: memberLogin, password: memberPass }
		});
		expect(loginRes.ok()).toBeTruthy();
		await memberPage.request.post('/api/v1/household-invites/accept', {
			data: { code: codeText }
		});

		const family = await apiJSON<{ id: string; name: string }>(page, 'POST', '/api/v1/accounts', {
			name: familyName,
			type: 'cash',
			initial_balance: '500.00',
			shared: true
		});
		const householdCat = await apiJSON<{ id: string }>(page, 'POST', '/api/v1/categories', {
			name: `E2E Food ${tag}`,
			type: 'expense',
			icon: 'food',
			scope: 'household'
		});
		const personal = await createCashAccount(page, personalName);
		await createExpense(page, family.id, '10.00', `Admin family ${tag}`, householdCat.id);

		await page.goto('/accounts');
		await waitAppReady(page);
		const familyCard = page.locator('.card').filter({ hasText: familyName });
		await expect(familyCard.getByTestId('family-badge')).toBeVisible();

		await memberPage.goto('/accounts');
		await waitAppReady(memberPage);
		await expect(memberPage.locator('.card').filter({ hasText: familyName })).toBeVisible();
		await expect(
			memberPage.locator('.card').filter({ hasText: familyName }).getByTestId('family-badge')
		).toBeVisible();

		await createExpense(memberPage, family.id, '15.00', `Member family ${tag}`, householdCat.id);
		await createTransfer(page, personal.id, family.id, '20.00');

		await page.goto('/');
		await waitAppReady(page);
		await page.getByRole('button', { name: 'Семья', exact: true }).click();
		await expect(page.getByText(familyName).first()).toBeVisible();
		await expect(page.getByText(personalName)).toHaveCount(0);
		await page.getByRole('button', { name: 'Личное', exact: true }).click();
		await expect(page.getByText(personalName).first()).toBeVisible();
		await expect(page.getByText(familyName)).toHaveCount(0);
		await page.getByRole('button', { name: 'Всё', exact: true }).click();
		await expect(page.getByText(personalName).first()).toBeVisible();
		await expect(page.getByText(familyName).first()).toBeVisible();
		await page.getByRole('button', { name: 'Семья', exact: true }).click();
		await expect(page.getByText(familyName).first()).toBeVisible();
		await expect(page.getByText(personalName)).toHaveCount(0);

		await page.goto(`/accounts/${family.id}`);
		await waitAppReady(page);
		await expect(page.getByText(`Member family ${tag}`).first()).toBeVisible({ timeout: 15_000 });
		await expect(page.getByText('Внёс: E2E House Member').first()).toBeVisible();
		await expect(page.getByText(`Admin family ${tag}`).first()).toBeVisible();
		await expect(page.getByText('Внёс: я')).toHaveCount(0);

		await memberPage.goto(`/accounts/${family.id}`);
		await waitAppReady(memberPage);
		await expect(memberPage.getByText(/Личный счёт/).first()).toBeVisible();
		await expect(memberPage.getByText('Внёс: E2E Admin').first()).toBeVisible();

		await memberPage.goto('/settings/household');
		await waitAppReady(memberPage);
		await memberPage.getByRole('button', { name: 'Выйти из семьи' }).click();
		await confirmDialog(memberPage, 'Выйти из семьи');
		await expectToast(memberPage, 'success', 'Вы вышли из семьи');
		await memberPage.goto('/accounts');
		await waitAppReady(memberPage);
		await expect(memberPage.getByText(familyName)).toHaveCount(0);

		await page.goto('/settings/household');
		await waitAppReady(page);
		await page.getByRole('button', { name: 'Распустить семью' }).click();
		await confirmDialog(page, 'Распустить семью');
		await expectToast(page, 'success', 'Семья распущена');
		await page.goto('/accounts');
		await waitAppReady(page);
		await expect(page.locator('.card').filter({ hasText: familyName })).toBeVisible();
		await expect(
			page.locator('.card').filter({ hasText: familyName }).getByTestId('family-badge')
		).toHaveCount(0);
	} finally {
		await memberContext.close();
		await restoreAdminSession(page);
		const me = await page.request.get('/api/v1/households/me');
		if (me.ok()) {
			const body = (await me.json()) as { household: { id: string } | null };
			if (body.household) {
				await page.request.post('/api/v1/households/me/dissolve', {
					data: { confirm_members: true }
				});
			}
		}
		await deleteAdminUserByLogin(page, memberLogin);
	}
});
