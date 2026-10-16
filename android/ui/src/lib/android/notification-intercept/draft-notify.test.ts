import { beforeEach, describe, expect, it, vi } from 'vitest';
import { createTransaction, createTransfer } from '$lib/offline/transactions-api';
import {
	applyNativeDraftNotifySync,
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
import * as plugin from './plugin';
import type { InterceptDraft, ParsedPurchase } from './types';

vi.mock('$lib/offline/transactions-api', () => ({
	createTransaction: vi.fn().mockResolvedValue({ id: 'tx1' }),
	createTransfer: vi.fn().mockResolvedValue({ id: 'tr1' })
}));

vi.mock('./plugin', async (importOriginal) => {
	const actual = await importOriginal<typeof import('./plugin')>();
	return {
		...actual,
		consumeDraftNotifySync: vi.fn()
	};
});

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

describe('applyNativeDraftNotifySync date format', () => {
	beforeEach(() => {
		vi.mocked(plugin.consumeDraftNotifySync).mockReset();
		vi.mocked(createTransaction).mockClear();
		vi.mocked(createTransfer).mockClear();
	});

	it('converts ISO offlineAccept transaction_date to API Layout before create', async () => {
		vi.mocked(plugin.consumeDraftNotifySync).mockResolvedValue({
			rejectedIds: [],
			acceptedIds: [],
			offlineAccepts: [
				{
					kind: 'transaction',
					draftId: 'd1',
					account_id: 'acc-1',
					type: 'expense',
					amount: '10.00',
					transaction_date: '2026-10-16T05:54:00.000Z'
				}
			]
		});
		await applyNativeDraftNotifySync('user-1');
		expect(createTransaction).toHaveBeenCalledWith(
			expect.objectContaining({
				transaction_date: '2026-10-16 05:54:00'
			})
		);
	});

	it('passes through Layout offlineAccept transaction_date', async () => {
		vi.mocked(plugin.consumeDraftNotifySync).mockResolvedValue({
			rejectedIds: [],
			acceptedIds: [],
			offlineAccepts: [
				{
					kind: 'transaction',
					draftId: 'd1',
					account_id: 'acc-1',
					type: 'expense',
					amount: '10.00',
					transaction_date: '2026-10-16 05:54:00'
				}
			]
		});
		await applyNativeDraftNotifySync('user-1');
		expect(createTransaction).toHaveBeenCalledWith(
			expect.objectContaining({
				transaction_date: '2026-10-16 05:54:00'
			})
		);
	});

	it('converts ISO transfer offlineAccept date to Layout', async () => {
		vi.mocked(plugin.consumeDraftNotifySync).mockResolvedValue({
			rejectedIds: [],
			acceptedIds: [],
			offlineAccepts: [
				{
					kind: 'transfer',
					draftIds: ['a', 'b'],
					from_account_id: 'acc-from',
					to_account_id: 'acc-to',
					amount: '10.00',
					transaction_date: '2026-10-16T05:54:00.000Z'
				}
			]
		});
		await applyNativeDraftNotifySync('user-1');
		expect(createTransfer).toHaveBeenCalledWith(
			expect.objectContaining({
				transaction_date: '2026-10-16 05:54:00'
			})
		);
	});
});
