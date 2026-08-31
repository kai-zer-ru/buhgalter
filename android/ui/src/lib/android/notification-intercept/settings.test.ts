import { beforeEach, describe, expect, it } from 'vitest';
import {
	clearInterceptSettingsForTests,
	loadInterceptSettings,
	saveInterceptSettings,
	normalizeLast4,
	setAccountBankBinding
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

	it('allows the same bank on several accounts', () => {
		const pkg = 'com.idamob.tinkoff.android';
		let bindings = setAccountBankBinding([], 'acc-1', { bankId: 'tinkoff', packageName: pkg });
		bindings = setAccountBankBinding(bindings, 'acc-2', { bankId: 'tinkoff', packageName: pkg });
		bindings = setAccountBankBinding(bindings, 'acc-3', { bankId: 'tinkoff', packageName: pkg });
		expect(bindings).toEqual([
			{ bankId: 'tinkoff', packageName: pkg, accountId: 'acc-1' },
			{ bankId: 'tinkoff', packageName: pkg, accountId: 'acc-2' },
			{ bankId: 'tinkoff', packageName: pkg, accountId: 'acc-3' }
		]);
	});

	it('replaces bank for one account without clearing siblings', () => {
		const tinkoffPkg = 'com.idamob.tinkoff.android';
		const sberPkg = 'ru.sberbankmobile';
		let bindings = [
			{ bankId: 'tinkoff', packageName: tinkoffPkg, accountId: 'acc-1' },
			{ bankId: 'tinkoff', packageName: tinkoffPkg, accountId: 'acc-2' }
		];
		bindings = setAccountBankBinding(bindings, 'acc-2', {
			bankId: 'sberbank',
			packageName: sberPkg
		});
		expect(bindings).toEqual([
			{ bankId: 'tinkoff', packageName: tinkoffPkg, accountId: 'acc-1' },
			{ bankId: 'sberbank', packageName: sberPkg, accountId: 'acc-2' }
		]);
	});

	it('clears binding for one account', () => {
		const pkg = 'com.idamob.tinkoff.android';
		const bindings = setAccountBankBinding(
			[
				{ bankId: 'tinkoff', packageName: pkg, accountId: 'acc-1' },
				{ bankId: 'tinkoff', packageName: pkg, accountId: 'acc-2' }
			],
			'acc-1',
			null
		);
		expect(bindings).toEqual([{ bankId: 'tinkoff', packageName: pkg, accountId: 'acc-2' }]);
	});
});
