import { beforeEach, describe, expect, it } from 'vitest';
import {
	clearInterceptSettingsForTests,
	loadInterceptSettings,
	saveInterceptSettings,
	normalizeLast4
} from './settings';

describe('intercept settings', () => {
	beforeEach(() => {
		clearInterceptSettingsForTests();
	});

	it('normalizes last4', () => {
		expect(normalizeLast4('*1234')).toBe('1234');
		expect(normalizeLast4('12')).toBe('12');
	});

	it('persists enabled independently of bindings', () => {
		saveInterceptSettings('u1', {
			enabled: true,
			bankBindings: [
				{
					bankId: 'tinkoff',
					packageName: 'com.idamob.tinkoff.android',
					accountId: 'acc-1'
				}
			],
			cardBindings: [{ bankId: 'tinkoff', last4: '4321', accountId: 'acc-2' }]
		});
		saveInterceptSettings('u1', {
			...loadInterceptSettings('u1'),
			enabled: false
		});
		const again = loadInterceptSettings('u1');
		expect(again.enabled).toBe(false);
		expect(again.bankBindings).toHaveLength(1);
		expect(again.cardBindings[0].last4).toBe('4321');
	});

	it('isolates users', () => {
		saveInterceptSettings('u1', {
			enabled: true,
			bankBindings: [],
			cardBindings: []
		});
		expect(loadInterceptSettings('u2').enabled).toBe(false);
	});
});
