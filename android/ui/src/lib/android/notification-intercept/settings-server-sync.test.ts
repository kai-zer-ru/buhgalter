import { beforeEach, describe, expect, it, vi } from 'vitest';

const getBindings = vi.fn();
const putBindings = vi.fn();

vi.mock('$lib/api/client', () => ({
	getNotificationInterceptBindings: (...args: unknown[]) => getBindings(...args),
	putNotificationInterceptBindings: (...args: unknown[]) => putBindings(...args)
}));

import {
	clearInterceptSettingsForTests,
	loadInterceptSettings,
	mergeInterceptBindings,
	saveInterceptSettings,
	syncInterceptSettingsFromServer
} from './settings';

describe('mergeInterceptBindings', () => {
	it('unionizes distinct keys and prefers local on conflict', () => {
		const merged = mergeInterceptBindings(
			{
				bankBindings: [
					{
						bankId: 'tinkoff',
						packageName: 'com.idamob.tinkoff.android',
						accountId: 'acc-1'
					}
				],
				cardBindings: [{ bankId: 'tinkoff', last4: '1111', accountId: 'acc-local' }]
			},
			{
				bankBindings: [
					{
						bankId: 'sberbank',
						packageName: 'ru.sberbankmobile',
						accountId: 'acc-2'
					}
				],
				cardBindings: [{ bankId: 'tinkoff', last4: '1111', accountId: 'acc-remote' }]
			}
		);
		expect(merged.bankBindings).toHaveLength(2);
		expect(merged.cardBindings).toEqual([
			{ bankId: 'tinkoff', last4: '1111', accountId: 'acc-local' }
		]);
	});
});

describe('syncInterceptSettingsFromServer', () => {
	beforeEach(() => {
		clearInterceptSettingsForTests();
		getBindings.mockReset();
		putBindings.mockReset();
	});

	it('pulls from server when local is empty (reinstall)', async () => {
		getBindings.mockResolvedValue({
			bank_bindings: [
				{
					package_name: 'com.idamob.tinkoff.android',
					bank_id: 'tinkoff',
					account_id: 'acc-1'
				}
			],
			card_bindings: [{ bank_id: 'tinkoff', last4: '1234', account_id: 'acc-1' }]
		});

		await syncInterceptSettingsFromServer('u1');

		const local = loadInterceptSettings('u1');
		expect(local.bankBindings).toHaveLength(1);
		expect(local.cardBindings[0].last4).toBe('1234');
		expect(putBindings).not.toHaveBeenCalled();
	});

	it('uploads local bindings when server is empty', async () => {
		saveInterceptSettings(
			'u1',
			{
				enabled: false,
				shadeNotifications: true,
				bankBindings: [
					{
						bankId: 'sberbank',
						packageName: 'ru.sberbankmobile',
						accountId: 'acc-sber'
					}
				],
				cardBindings: []
			},
			{ skipServer: true }
		);
		getBindings.mockResolvedValue({ bank_bindings: [], card_bindings: [] });
		putBindings.mockResolvedValue({ bank_bindings: [], card_bindings: [] });

		await syncInterceptSettingsFromServer('u1');

		expect(putBindings).toHaveBeenCalledWith({
			bank_bindings: [
				{
					package_name: 'ru.sberbankmobile',
					bank_id: 'sberbank',
					account_id: 'acc-sber'
				}
			],
			card_bindings: []
		});
		expect(loadInterceptSettings('u1').bankBindings).toHaveLength(1);
	});

	it('does not wipe local when server is empty', async () => {
		saveInterceptSettings(
			'u1',
			{
				enabled: true,
				shadeNotifications: false,
				bankBindings: [
					{
						bankId: 'tinkoff',
						packageName: 'com.idamob.tinkoff.android',
						accountId: 'acc-1'
					}
				],
				cardBindings: []
			},
			{ skipServer: true }
		);
		getBindings.mockResolvedValue({ bank_bindings: [], card_bindings: [] });
		putBindings.mockRejectedValue(new Error('offline'));

		await syncInterceptSettingsFromServer('u1');

		expect(loadInterceptSettings('u1').bankBindings).toHaveLength(1);
		expect(loadInterceptSettings('u1').enabled).toBe(true);
	});

	it('merges both sides and pushes when server differs', async () => {
		saveInterceptSettings(
			'u1',
			{
				enabled: true,
				shadeNotifications: true,
				bankBindings: [
					{
						bankId: 'tinkoff',
						packageName: 'com.idamob.tinkoff.android',
						accountId: 'acc-local'
					}
				],
				cardBindings: []
			},
			{ skipServer: true }
		);
		getBindings.mockResolvedValue({
			bank_bindings: [
				{
					package_name: 'ru.sberbankmobile',
					bank_id: 'sberbank',
					account_id: 'acc-remote'
				}
			],
			card_bindings: []
		});
		putBindings.mockResolvedValue({ bank_bindings: [], card_bindings: [] });

		await syncInterceptSettingsFromServer('u1');

		expect(loadInterceptSettings('u1').bankBindings).toHaveLength(2);
		expect(putBindings).toHaveBeenCalledWith({
			bank_bindings: expect.arrayContaining([
				{
					package_name: 'com.idamob.tinkoff.android',
					bank_id: 'tinkoff',
					account_id: 'acc-local'
				},
				{
					package_name: 'ru.sberbankmobile',
					bank_id: 'sberbank',
					account_id: 'acc-remote'
				}
			]),
			card_bindings: []
		});
	});
});
