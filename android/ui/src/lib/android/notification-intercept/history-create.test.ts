import { describe, expect, it } from 'vitest';
import {
	canCreateFromHistory,
	draftFromHistoryItem,
	historyTransferComplement,
	prefillFromHistoryItem,
	transferPrefillFromHistory
} from './history-create';
import type { InterceptSettings, NotificationHistoryItem } from './types';

const emptySettings: InterceptSettings = {
	enabled: true,
	shadeNotifications: true,
	bankBindings: [],
	cardBindings: []
};

function history(
	partial: Partial<NotificationHistoryItem> & { packageName: string; text: string }
): NotificationHistoryItem {
	return {
		title: '',
		bigText: '',
		postedAt: 1_700_000_000_000,
		dedupeKey: partial.dedupeKey ?? `k-${partial.packageName}-${partial.text.slice(0, 20)}`,
		inAllowlist: true,
		queued: true,
		...partial
	};
}

describe('history-create', () => {
	it('canCreateFromHistory for allowlisted purchase', () => {
		const row = history({
			packageName: 'com.idamob.tinkoff.android',
			title: 'Покупка',
			text: 'Покупка 1 234,50 ₽. Пятёрочка. Карта *4321'
		});
		expect(canCreateFromHistory(row)).toBe(true);
	});

	it('rejects cancel and non-allowlist', () => {
		expect(
			canCreateFromHistory(
				history({
					packageName: 'com.idamob.tinkoff.android',
					text: 'Отмена покупки 100 ₽. Магнит'
				})
			)
		).toBe(false);
		expect(
			canCreateFromHistory(
				history({
					packageName: 'com.other.app',
					text: 'Покупка 100 ₽',
					inAllowlist: false
				})
			)
		).toBe(false);
	});

	it('draftFromHistoryItem resolves account via last4 binding', () => {
		const settings: InterceptSettings = {
			...emptySettings,
			cardBindings: [{ bankId: 'tinkoff', last4: '4321', accountId: 'acc-t' }]
		};
		const draft = draftFromHistoryItem(
			history({
				packageName: 'com.idamob.tinkoff.android',
				title: 'Покупка',
				text: 'Покупка 99.00 ₽. Магнит. Карта *4321'
			}),
			settings,
			[{ id: 'm1', name: 'Магнит' }]
		);
		expect(draft).not.toBeNull();
		expect(draft!.accountId).toBe('acc-t');
		expect(draft!.merchantId).toBe('m1');
		expect(draft!.parsed.amount).toBe('99.00');
	});

	it('builds transfer prefill from complementary history rows', () => {
		const settings: InterceptSettings = {
			...emptySettings,
			bankBindings: [
				{
					packageName: 'com.idamob.tinkoff.android',
					bankId: 'tinkoff',
					accountId: 'acc-t'
				},
				{
					packageName: 'ru.sberbankmobile',
					bankId: 'sberbank',
					accountId: 'acc-s'
				}
			]
		};
		const expense = history({
			packageName: 'com.idamob.tinkoff.android',
			dedupeKey: 'exp',
			text: 'Покупка 1500.00 ₽. Перевод. Карта *1111'
		});
		const income = history({
			packageName: 'ru.sberbankmobile',
			dedupeKey: 'inc',
			postedAt: 1_700_000_000_000 + 60_000,
			text: 'Перевод от Иван И. 1 500 ₽. Баланс 10 000 ₽'
		});
		const complement = historyTransferComplement(expense, [expense, income], settings, []);
		expect(complement).not.toBeNull();
		const prefill = transferPrefillFromHistory(expense, [expense, income], settings, []);
		expect(prefill).not.toBeNull();
		expect(prefill!.amount).toBe('1500.00');
		expect(prefill!.fromAccountId).toBe('acc-t');
		expect(prefill!.toAccountId).toBe('acc-s');
		expect(prefill!.draftIds).toBeUndefined();
	});

	it('returns null transfer prefill without a unique pair', () => {
		const settings: InterceptSettings = {
			...emptySettings,
			bankBindings: [
				{
					packageName: 'com.idamob.tinkoff.android',
					bankId: 'tinkoff',
					accountId: 'acc-t'
				}
			]
		};
		const expense = history({
			packageName: 'com.idamob.tinkoff.android',
			text: 'Покупка 99.00 ₽. Магнит. Карта *1111'
		});
		expect(transferPrefillFromHistory(expense, [expense], settings, [])).toBeNull();
	});

	it('prefillFromHistoryItem omits draftId', async () => {
		const settings: InterceptSettings = {
			...emptySettings,
			cardBindings: [{ bankId: 'tinkoff', last4: '4321', accountId: 'acc-t' }]
		};
		const prefill = await prefillFromHistoryItem(
			history({
				packageName: 'com.idamob.tinkoff.android',
				title: 'Покупка',
				text: 'Покупка 10.00 ₽. Shop. Карта *4321'
			}),
			settings,
			[]
		);
		expect(prefill).not.toBeNull();
		expect(prefill!.amount).toBe('10.00');
		expect(prefill!.accountId).toBe('acc-t');
		expect(prefill!.type).toBe('expense');
		expect(prefill!.draftId).toBeUndefined();
	});
});
