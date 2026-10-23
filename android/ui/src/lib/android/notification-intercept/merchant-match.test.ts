import { describe, expect, it } from 'vitest';
import { matchMerchant, toMerchantRefs } from './merchant-match';

describe('matchMerchant', () => {
	const merchants = [
		{ id: 'm1', name: 'Пятёрочка' },
		{ id: 'm2', name: 'MAGNIT' }
	];

	it('matches exact ignore case', () => {
		expect(matchMerchant('пятёрочка', merchants)).toEqual({
			merchantId: 'm1',
			merchantName: 'Пятёрочка'
		});
	});

	it('proposes name when no match', () => {
		expect(matchMerchant('Unknown Shop', merchants)).toEqual({
			merchantName: 'Unknown Shop'
		});
	});

	it('returns empty for blank', () => {
		expect(matchMerchant('  ', merchants)).toEqual({});
	});

	it('keeps aliases when building refs for history matching', () => {
		const refs = toMerchantRefs([
			{ id: 'm1', name: 'Пятёрочка', aliases: [' Pyaterochka ', ''] },
			{ id: 'm2', name: 'MAGNIT' }
		]);
		expect(refs).toEqual([
			{ id: 'm1', name: 'Пятёрочка', aliases: ['Pyaterochka'] },
			{ id: 'm2', name: 'MAGNIT', aliases: [] }
		]);
		expect(matchMerchant('pyaterochka', refs)).toEqual({
			merchantId: 'm1',
			merchantName: 'Пятёрочка'
		});
	});

	it('matches an alias and returns the primary name', () => {
		const withAlias = [{ id: 'm1', name: 'Пятёрочка', aliases: ['Pyaterochka'] }];
		expect(matchMerchant('pyaterochka', withAlias)).toEqual({
			merchantId: 'm1',
			merchantName: 'Пятёрочка'
		});
		expect(matchMerchant('pyater', withAlias)).toEqual({
			merchantId: 'm1',
			merchantName: 'Пятёрочка'
		});
	});
});
