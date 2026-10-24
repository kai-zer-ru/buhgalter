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
		await apiJSON(page, 'PUT', '/api/v1/user/notifications', {
			telegram_enabled: true
		});
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

	test('global default buttons: both channels use tabs', async ({ page }) => {
		await apiJSON(page, 'PUT', '/api/v1/user/notifications', {
			telegram_enabled: true,
			max_enabled: true
		});
		await page.goto('/settings/notifications');
		await waitAppReady(page);

		await expandCollapsibleSection(page, 'Кнопки по умолчанию');
		const globalCard = page.getByTestId('notifications-global-keyboards');
		await expect(globalCard.getByRole('tab', { name: 'Telegram' })).toBeVisible();
		await expect(globalCard.getByRole('tab', { name: 'MAX' })).toBeVisible();
		await expect(globalCard.getByTestId('keyboard-builder-global-tg')).toBeVisible();
	});

	test('global default buttons: disabled MAX has no tabs and shows Telegram', async ({ page }) => {
		await apiJSON(page, 'PUT', '/api/v1/user/notifications', {
			telegram_enabled: true,
			max_enabled: false
		});
		await page.goto('/settings/notifications');
		await waitAppReady(page);

		await expandCollapsibleSection(page, 'Кнопки по умолчанию');
		const globalCard = page.getByTestId('notifications-global-keyboards');
		await expect(globalCard.getByRole('tab')).toHaveCount(0);
		await expect(globalCard.getByTestId('keyboard-builder-global-tg')).toBeVisible();
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

	test('template includes global buttons, add keeps them, delete is hidden', async ({ page }) => {
		const settings = await apiJSON<{
			templates: { trigger_type: string; template: string }[];
		}>(page, 'GET', '/api/v1/user/notifications');
		const testTpl = settings.templates.find((tpl) => tpl.trigger_type === 'test');
		const templateText = testTpl?.template?.trim() || 'Тест';

		await apiJSON(page, 'PUT', '/api/v1/user/notifications', {
			telegram_enabled: true,
			keyboard_telegram: {
				rows: [
					[
						{ id: 'g-debts', type: 'url', text: 'Глобальные долги', url: '{debt_url}' },
						{ id: 'g-budget', type: 'url', text: 'Глобальный бюджет', url: '{budget_url}' }
					]
				]
			},
			templates: [
				{
					trigger_type: 'test',
					template: templateText,
					keyboard_telegram: { rows: [] },
					keyboard_max: { rows: [] }
				}
			]
		});

		try {
			await page.goto('/settings/notifications');
			await waitAppReady(page);
			await expandCollapsibleSection(page, 'Тестовое сообщение');
			const card = page.getByTestId('notifications-template-test');
			const builder = page.getByTestId('keyboard-builder-tpl-test-tg');
			await expect(builder).toContainText('Глобальные долги');
			await expect(builder).toContainText('Глобальный бюджет');

			const locked = builder.locator('[data-locked="true"]');
			await expect(locked).toHaveCount(2);
			await expect(locked.getByRole('button', { name: 'Удалить' })).toHaveCount(0);
			await expect(locked.locator('input').first()).toBeDisabled();

			await builder.getByRole('button', { name: 'Добавить кнопку' }).click();
			const local = builder.locator('[data-locked="false"]');
			await expect(local).toHaveCount(1);
			await local.locator('input').nth(0).fill('Своя кнопка');
			await local.locator('input').nth(1).fill('https://example.com/own');

			const putPromise = page.waitForResponse(
				(res) =>
					res.url().includes('/api/v1/user/notifications') &&
					res.request().method() === 'PUT' &&
					res.ok()
			);
			await card.getByRole('button', { name: 'Сохранить' }).click();
			const putRes = await putPromise;
			const body = putRes.request().postDataJSON() as {
				templates?: {
					keyboard_telegram?: {
						rows: { text?: string }[][];
						placement?: string[][];
					};
				}[];
			};
			const kb = body.templates?.[0]?.keyboard_telegram;
			const flat = (kb?.placement ?? []).flat();
			expect(flat).toContain('g:g-debts');
			expect(flat).toContain('g:g-budget');
			expect(flat.some((token) => token.startsWith('l:'))).toBe(true);
			expect(kb?.rows?.some((row) => row.some((btn) => btn.text === 'Своя кнопка'))).toBe(true);
			await expectToast(page, 'success', 'Шаблон «Тестовое сообщение» сохранён.');
		} finally {
			await apiJSON(page, 'PUT', '/api/v1/user/notifications', {
				keyboard_telegram: { rows: [] },
				templates: [
					{
						trigger_type: 'test',
						template: templateText,
						keyboard_telegram: { rows: [] },
						keyboard_max: { rows: [] }
					}
				]
			});
		}
	});
});
