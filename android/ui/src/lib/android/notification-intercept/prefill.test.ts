import { beforeEach, describe, expect, it } from 'vitest';
import {
	prefillFromDraft,
	resetInterceptPrefillForTests,
	setInterceptPrefill,
	takeInterceptPrefill
} from './prefill';
import type { InterceptDraft } from './types';

describe('intercept prefill', () => {
	beforeEach(() => {
		resetInterceptPrefillForTests();
	});

	it('set/take once', () => {
		setInterceptPrefill({ description: '  Shop  ', amount: '10.00', draftId: 'd1' });
		expect(takeInterceptPrefill()).toEqual({
			description: 'Shop',
			amount: '10.00',
			accountId: undefined,
			merchantId: undefined,
			merchantName: undefined,
			categoryId: undefined,
			subcategoryId: undefined,
			occurredAt: undefined,
			draftId: 'd1'
		});
		expect(takeInterceptPrefill()).toBeNull();
	});

	it('keeps category suggestion fields', () => {
		setInterceptPrefill({
			amount: '10.00',
			categoryId: 'cat1',
			subcategoryId: 'sub1',
			draftId: 'd1'
		});
		expect(takeInterceptPrefill()).toMatchObject({
			categoryId: 'cat1',
			subcategoryId: 'sub1'
		});
	});

	it('builds prefill from draft', () => {
		const draft: InterceptDraft = {
			id: 'd2',
			createdAt: new Date().toISOString(),
			parsed: {
				bankId: 'tinkoff',
				packageName: 'com.idamob.tinkoff.android',
				amount: '5.00',
				occurredAt: '2024-01-01T10:00:00.000Z',
				merchantText: 'Cafe',
				rawHash: 'h'
			},
			accountId: 'a1',
			merchantName: 'Cafe'
		};
		const prefill = prefillFromDraft(draft);
		expect(prefill.draftId).toBe('d2');
		expect(prefill.accountId).toBe('a1');
		expect(prefill.merchantName).toBe('Cafe');
		expect(prefill.description).toBeUndefined();
		expect(prefill.categoryId).toBeUndefined();
	});
});
