import { describe, expect, it } from 'vitest';
import { resolveAccountId } from './account-resolve';
import type { InterceptSettings, ParsedPurchase } from './types';

const parsed: ParsedPurchase = {
	bankId: 'tinkoff',
	packageName: 'com.idamob.tinkoff.android',
	amount: '10.00',
	occurredAt: new Date().toISOString(),
	merchantText: 'Shop',
	last4: '4321',
	rawHash: 'h1'
};

const settings: InterceptSettings = {
	enabled: true,
	bankBindings: [
		{
			bankId: 'tinkoff',
			packageName: 'com.idamob.tinkoff.android',
			accountId: 'acc-bank'
		}
	],
	cardBindings: [{ bankId: 'tinkoff', last4: '4321', accountId: 'acc-card' }]
};

describe('resolveAccountId', () => {
	it('prefers last4 binding', () => {
		expect(resolveAccountId(parsed, settings)).toBe('acc-card');
	});

	it('falls back to bank binding', () => {
		expect(resolveAccountId({ ...parsed, last4: undefined }, settings)).toBe('acc-bank');
		expect(resolveAccountId({ ...parsed, last4: '9999' }, settings)).toBe('acc-bank');
	});

	it('returns undefined when nothing matches', () => {
		expect(
			resolveAccountId(parsed, { enabled: true, bankBindings: [], cardBindings: [] })
		).toBeUndefined();
	});
});
