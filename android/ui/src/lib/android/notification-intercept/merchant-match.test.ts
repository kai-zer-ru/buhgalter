import { describe, expect, it } from 'vitest';
import { matchMerchant } from './merchant-match';

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
});
