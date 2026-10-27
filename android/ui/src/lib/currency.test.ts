import { describe, expect, it } from 'vitest';
import {
	currencySelectOptions,
	hasOtherAccountCurrencies,
	mergeAvailableCurrencies,
	ownFundsTotalsByCurrency,
	resolveStoredStatsCurrency,
	sortRowsByProfileCurrency
} from './currency';

describe('currencySelectOptions', () => {
	it('labels options as code plus symbol', () => {
		expect(currencySelectOptions([{ code: 'USD', symbol: '$' }])).toEqual([
			{ value: 'USD', label: 'USD $' }
		]);
	});
});

describe('sortRowsByProfileCurrency', () => {
	it('puts the profile currency first, then alphabetical', () => {
		const rows = [{ currency: 'USD' }, { currency: 'EUR' }, { currency: 'RUB' }];
		expect(sortRowsByProfileCurrency(rows, 'RUB').map((r) => r.currency)).toEqual([
			'RUB',
			'EUR',
			'USD'
		]);
	});
});

describe('ownFundsTotalsByCurrency', () => {
	it('splits cash and bank by account currency and skips credit cards', () => {
		const rows = ownFundsTotalsByCurrency(
			[
				{ type: 'cash', currency: 'GBP', balance: 100, forecast_balance: 100 },
				{ type: 'bank', currency: 'RUB', balance: 200, forecast_balance: 250 },
				{ type: 'credit_card', currency: 'RUB', balance: 999, forecast_balance: 999 }
			],
			'RUB'
		);
		expect(rows.map((r) => r.currency)).toEqual(['RUB', 'GBP']);
		expect(rows[0]).toMatchObject({ total_balance: 200, total_forecast: 250 });
		expect(rows[1]).toMatchObject({ total_balance: 100, total_forecast: 100 });
	});
});

describe('resolveStoredStatsCurrency', () => {
	it('keeps a stored code that is still available', () => {
		expect(resolveStoredStatsCurrency(['RUB', 'USD'], 'USD', 'RUB')).toBe('USD');
	});

	it('falls back when the stored code is gone', () => {
		expect(resolveStoredStatsCurrency(['RUB', 'EUR'], 'USD', 'RUB')).toBe('RUB');
	});
});

describe('mergeAvailableCurrencies', () => {
	it('unions API codes with account currencies and puts the profile first', () => {
		expect(
			mergeAvailableCurrencies(['USD'], [{ currency: 'GBP' }, { currency: 'RUB' }], 'RUB')
		).toEqual(['RUB', 'GBP', 'USD']);
	});
});

describe('hasOtherAccountCurrencies', () => {
	it('is true when any account is not the profile currency', () => {
		expect(hasOtherAccountCurrencies([{ currency: 'RUB' }, { currency: 'USD' }], 'RUB')).toBe(true);
		expect(hasOtherAccountCurrencies([{ currency: 'RUB' }], 'RUB')).toBe(false);
	});
});
