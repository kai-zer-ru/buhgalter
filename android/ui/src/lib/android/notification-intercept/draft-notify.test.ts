import { beforeEach, describe, expect, it } from 'vitest';
import {
	buildShadeNotifyItems,
	shadeTransferItemId,
	toNativeDraftPayload,
	toNativeTransferPayload
} from './draft-notify';
import {
	clearInterceptDraftsForTests,
	addInterceptDraft,
	updateInterceptDraft,
	listInterceptDrafts
} from './drafts';
import type { InterceptDraft, ParsedPurchase } from './types';

const purchase: ParsedPurchase = {
	bankId: 'tinkoff',
	packageName: 'com.idamob.tinkoff.android',
	amount: '50.00',
	occurredAt: '2024-06-01T12:00:00.000Z',
	merchantText: 'Shop',
	rawHash: 'hash-notify-1',
	kind: 'purchase'
};

describe('draft-notify payload', () => {
	beforeEach(() => {
		clearInterceptDraftsForTests('user-1');
	});

	it('maps expense draft for native shade Accept', () => {
		const draft = addInterceptDraft(
			purchase,
			{ accountId: 'acc-1', merchantName: 'Shop', categoryId: 'cat-1', subcategoryId: 'sub-1' },
			'user-1'
		)!;
		const payload = toNativeDraftPayload(draft);
		expect(payload).toMatchObject({
			id: draft.id,
			type: 'expense',
			amount: '50.00',
			accountId: 'acc-1',
			merchantName: 'Shop',
			categoryId: 'cat-1',
			subcategoryId: 'sub-1',
			occurredAt: '2024-06-01T12:00:00.000Z'
		});
	});

	it('maps income label as description', () => {
		const draft: InterceptDraft = {
			id: 'd-inc',
			createdAt: new Date().toISOString(),
			parsed: { ...purchase, kind: 'income', merchantText: 'Пополнение', rawHash: 'inc' },
			accountId: 'acc-1',
			merchantName: 'Пополнение'
		};
		const payload = toNativeDraftPayload(draft);
		expect(payload.type).toBe('income');
		expect(payload.description).toBe('Пополнение');
		expect(payload.merchantName).toBeUndefined();
	});

	it('stores category via updateInterceptDraft', () => {
		const draft = addInterceptDraft(purchase, { accountId: 'acc-1', merchantId: 'm1' }, 'user-1')!;
		updateInterceptDraft(draft.id, { categoryId: 'c1', subcategoryId: 's1' }, 'user-1');
		expect(listInterceptDrafts('user-1')[0]).toMatchObject({
			categoryId: 'c1',
			subcategoryId: 's1'
		});
	});

	it('merges unique transfer pair into one shade item', () => {
		const from = addInterceptDraft(
			{ ...purchase, rawHash: 'exp', kind: 'purchase', bankId: 'tinkoff' },
			{ accountId: 'acc-t' },
			'user-1'
		)!;
		const to = addInterceptDraft(
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
		)!;
		const items = buildShadeNotifyItems(listInterceptDrafts('user-1'));
		expect(items).toHaveLength(1);
		expect(items[0]).toMatchObject({
			id: shadeTransferItemId(from.id, to.id),
			type: 'transfer',
			amount: '50.00',
			fromAccountId: 'acc-t',
			toAccountId: 'acc-s',
			fromDraftId: from.id,
			toDraftId: to.id
		});
		expect(toNativeTransferPayload({ from, to }).type).toBe('transfer');
	});

	it('keeps unpaired drafts as separate shade items', () => {
		addInterceptDraft(purchase, { accountId: 'acc-1' }, 'user-1');
		const items = buildShadeNotifyItems(listInterceptDrafts('user-1'));
		expect(items).toHaveLength(1);
		expect(items[0].type).toBe('expense');
	});
});
