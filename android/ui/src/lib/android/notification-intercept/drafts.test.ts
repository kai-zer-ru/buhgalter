import { beforeEach, describe, expect, it } from 'vitest';
import {
	addInterceptDraft,
	clearInterceptDraftsForTests,
	countInterceptDrafts,
	deleteInterceptDraft,
	deleteInterceptDrafts,
	listInterceptDrafts,
	removeDraftMatchingCancel
} from './drafts';
import type { ParsedPurchase } from './types';

const purchase: ParsedPurchase = {
	bankId: 'tinkoff',
	packageName: 'com.idamob.tinkoff.android',
	amount: '50.00',
	occurredAt: new Date().toISOString(),
	merchantText: 'Shop',
	rawHash: 'hash-1',
	kind: 'purchase'
};

describe('intercept drafts', () => {
	beforeEach(() => {
		clearInterceptDraftsForTests('user-1');
	});

	it('adds and lists drafts', () => {
		const d = addInterceptDraft(purchase, { accountId: 'a1' }, 'user-1');
		expect(d).not.toBeNull();
		expect(listInterceptDrafts('user-1')).toHaveLength(1);
		expect(countInterceptDrafts('user-1')).toBe(1);
	});

	it('counts a unique transfer pair as one item', () => {
		addInterceptDraft(
			{ ...purchase, rawHash: 'exp', kind: 'purchase', bankId: 'tinkoff' },
			{ accountId: 'acc-t' },
			'user-1'
		);
		addInterceptDraft(
			{
				...purchase,
				rawHash: 'inc',
				kind: 'income',
				bankId: 'sberbank',
				packageName: 'ru.sberbankmobile',
				merchantText: 'Пополнение'
			},
			{ accountId: 'acc-s' },
			'user-1'
		);
		expect(listInterceptDrafts('user-1')).toHaveLength(2);
		expect(countInterceptDrafts('user-1')).toBe(1);
	});

	it('dedupes by rawHash', () => {
		addInterceptDraft(purchase, {}, 'user-1');
		expect(addInterceptDraft(purchase, {}, 'user-1')).toBeNull();
		expect(listInterceptDrafts('user-1')).toHaveLength(1);
	});

	it('dedupes push vs SMS by amount/merchant within 2h', () => {
		addInterceptDraft(purchase, { merchantName: 'Shop' }, 'user-1');
		const fromSms: ParsedPurchase = {
			...purchase,
			rawHash: 'sms-hash-different',
			occurredAt: new Date(Date.now() + 60_000).toISOString()
		};
		expect(addInterceptDraft(fromSms, { merchantName: 'Shop' }, 'user-1')).toBeNull();
		expect(listInterceptDrafts('user-1')).toHaveLength(1);
	});

	it('allows same amount different merchant', () => {
		addInterceptDraft(purchase, { merchantName: 'Shop A' }, 'user-1');
		const other: ParsedPurchase = {
			...purchase,
			rawHash: 'hash-2',
			merchantText: 'Shop B'
		};
		expect(addInterceptDraft(other, { merchantName: 'Shop B' }, 'user-1')).not.toBeNull();
		expect(listInterceptDrafts('user-1')).toHaveLength(2);
	});

	it('deletes draft without creating transaction', () => {
		const d = addInterceptDraft(purchase, {}, 'user-1');
		expect(deleteInterceptDraft(d!.id, 'user-1')).toBe(true);
		expect(listInterceptDrafts('user-1')).toHaveLength(0);
	});

	it('deletes several drafts at once', () => {
		const a = addInterceptDraft(purchase, {}, 'user-1');
		const b = addInterceptDraft({ ...purchase, rawHash: 'hash-2', amount: '10.00' }, {}, 'user-1');
		expect(a && b).toBeTruthy();
		expect(deleteInterceptDrafts([a!.id, b!.id], 'user-1')).toBe(2);
		expect(listInterceptDrafts('user-1')).toHaveLength(0);
	});

	it('dedupes bank push vs wallet push for the same purchase', () => {
		addInterceptDraft(purchase, { merchantName: 'Shop' }, 'user-1');
		const fromWallet: ParsedPurchase = {
			...purchase,
			bankId: 'mir_pay',
			packageName: 'ru.nspk.mirpay',
			rawHash: 'wallet-hash',
			occurredAt: new Date(Date.now() + 30_000).toISOString()
		};
		expect(addInterceptDraft(fromWallet, { merchantName: 'Shop' }, 'user-1')).toBeNull();
		expect(listInterceptDrafts('user-1')).toHaveLength(1);
	});

	it('removes wallet cancel matching bank purchase draft', () => {
		addInterceptDraft(
			{
				bankId: 'tinkoff',
				packageName: 'com.idamob.tinkoff.android',
				amount: '155.00',
				occurredAt: new Date().toISOString(),
				merchantText: 'Поехали!',
				last4: '3349',
				rawHash: 'purchase-wallet-pair',
				kind: 'purchase'
			},
			{ merchantName: 'Поехали!' },
			'user-1'
		);
		const removed = removeDraftMatchingCancel(
			{
				bankId: 'mir_pay',
				packageName: 'ru.nspk.mirpay',
				amount: '155.00',
				occurredAt: new Date().toISOString(),
				merchantText: 'Поехали!',
				rawHash: 'cancel-wallet',
				kind: 'cancel'
			},
			'user-1'
		);
		expect(removed).toBe(1);
		expect(listInterceptDrafts('user-1')).toHaveLength(0);
	});

	it('removes draft matching cancel by amount and merchant', () => {
		addInterceptDraft(
			{
				bankId: 'yandex',
				packageName: 'com.yandex.bank',
				amount: '155.00',
				occurredAt: new Date().toISOString(),
				merchantText: 'Поехали!',
				last4: '3349',
				rawHash: 'purchase-1',
				kind: 'purchase'
			},
			{ merchantName: 'Поехали!' },
			'user-1'
		);
		const removed = removeDraftMatchingCancel(
			{
				bankId: 'yandex',
				packageName: 'com.yandex.bank',
				amount: '155.00',
				occurredAt: new Date().toISOString(),
				merchantText: 'Поехали!',
				rawHash: 'cancel-1',
				kind: 'cancel'
			},
			'user-1'
		);
		expect(removed).toBe(1);
		expect(listInterceptDrafts('user-1')).toHaveLength(0);
	});
});
