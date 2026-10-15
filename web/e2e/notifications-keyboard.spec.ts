import { test, expect } from '@playwright/test';
import { apiJSON, restoreAdminSession, waitAppReady } from './helpers/auth';
import { expandCollapsibleSection, expectToast } from './helpers/ui';

test.describe('notification keyboards', () => {
	test.beforeEach(async ({ page }) => {
		await restoreAdminSession(page);
		await apiJSON(page, 'PUT', '/api/v1/admin/settings/notification-secret', {
			notification_secret_key: '12345678901234567890123456789012'
		});
	});

	test('global default buttons: add preset, save, reload', async ({ page }) => {
		await page.goto('/settings/notifications');
		await waitAppReady(page);

		await expandCollapsibleSection(page, 'Кнопки по умолчанию');
		const globalCard = page.getByTestId('notifications-global-keyboards');
		await expect(globalCard.getByText('Кнопки по умолчанию')).toBeVisible();

		await globalCard.getByRole('button', { name: 'Пресет: Открыть долги' }).click();
		await expect(page.getByTestId('keyboard-preview-global-tg')).toBeVisible();
		await expect(page.getByTestId('keyboard-preview-global-tg')).toContainText('Открыть долги');

		const putPromise = page.waitForResponse(
			(res) =>
				res.url().includes('/api/v1/user/notifications') &&
				res.request().method() === 'PUT' &&
				res.ok()
		);
		await globalCard.getByTestId('notifications-global-keyboards-save').click();
		const putRes = await putPromise;
		const putBody = putRes.request().postDataJSON() as {
			keyboard_telegram?: { rows: Array<Array<{ type: string; text: string; url?: string }>> };
		};
		expect(putBody.keyboard_telegram?.rows?.[0]?.[0]).toMatchObject({
			type: 'url',
			text: 'Открыть долги',
			url: '{debt_url}'
		});
		await expectToast(page, 'success', 'Изменения сохранены.');

		await page.reload();
		await waitAppReady(page);
		await expandCollapsibleSection(page, 'Кнопки по умолчанию');
		await expect(page.getByTestId('keyboard-preview-global-tg')).toContainText('Открыть долги');

		// cleanup: clear global keyboard
		await page
			.getByTestId('notifications-global-keyboards')
			.getByRole('button', {
				name: 'Удалить ряд'
			})
			.click();
		await page.getByTestId('notifications-global-keyboards-save').click();
		await expectToast(page, 'success', 'Изменения сохранены.');
	});

	test('template preview returns keyboard schemas', async ({ page }) => {
		await apiJSON(page, 'PUT', '/api/v1/user/notifications', {
			telegram_enabled: true,
			max_enabled: true
		});
		await page.goto('/settings/notifications');
		await waitAppReady(page);

		await expandCollapsibleSection(page, 'Тестовое сообщение');
		const card = page.getByTestId('notifications-template-test');
		await expect(card).toBeVisible();

		await card.getByRole('button', { name: 'Пресет: Настройки уведомлений' }).click();
		await expect(page.getByTestId('keyboard-preview-tpl-test-tg')).toContainText(
			'Настройки уведомлений'
		);

		const previewPromise = page.waitForResponse(
			(res) =>
				res.url().includes('/api/v1/user/notifications/templates/preview') &&
				res.request().method() === 'POST' &&
				res.ok()
		);
		await card.getByRole('button', { name: 'Предпросмотр' }).click();
		const previewRes = await previewPromise;
		const json = (await previewRes.json()) as {
			text: string;
			keyboard_telegram: { rows: unknown[] };
			keyboard_max: { rows: unknown[] };
		};
		expect(json.text.length).toBeGreaterThan(0);
		expect(json.keyboard_telegram).toBeTruthy();
		expect(json.keyboard_max).toBeTruthy();
		await expect(card.getByText('Клавиатура Telegram (preview)')).toBeVisible();
		await expect(card.getByText('Клавиатура MAX (preview)')).toBeVisible();
	});

	test('template keyboard hides MAX tab when max is disabled', async ({ page }) => {
		await apiJSON(page, 'PUT', '/api/v1/user/notifications', {
			telegram_enabled: true,
			max_enabled: false
		});
		await page.goto('/settings/notifications');
		await waitAppReady(page);

		await expandCollapsibleSection(page, 'Тестовое сообщение');
		const card = page.getByTestId('notifications-template-test');
		await expect(card.getByText('Кнопки шаблона')).toBeVisible();
		await expect(card.getByRole('button', { name: 'MAX' })).toHaveCount(0);
		await expect(card.getByRole('button', { name: 'Telegram' })).toHaveCount(0);
		await expect(page.getByTestId('keyboard-builder-tpl-test-tg')).toBeVisible();
	});
});
