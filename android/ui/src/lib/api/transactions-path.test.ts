import { describe, expect, it } from 'vitest';
import {
	HOME_PAST_TRANSACTIONS_PATH,
	HOME_PLANNED_TRANSACTIONS_PATH,
	transactionsListPath
} from './transactions-path';

describe('transactionsListPath', () => {
	it('sorts query keys so object key order does not split ref-cache', () => {
		const a = transactionsListPath({
			kind: 'manual',
			sort: 'date_desc',
			page: '1',
			limit: '10'
		});
		const b = transactionsListPath({
			kind: 'manual',
			limit: '10',
			page: '1',
			sort: 'date_desc'
		});
		expect(a).toBe(b);
		expect(a).toBe('/api/v1/transactions?kind=manual&limit=10&page=1&sort=date_desc');
	});

	it('home path constants match listTransactions param shapes used on home/warm', () => {
		expect(HOME_PAST_TRANSACTIONS_PATH).toBe(
			transactionsListPath({
				kind: 'manual',
				sort: 'date_desc',
				page: '1',
				limit: '10'
			})
		);
		expect(HOME_PLANNED_TRANSACTIONS_PATH).toBe(
			transactionsListPath({
				kind: 'future',
				sort: 'date_desc',
				page: '1',
				limit: '10'
			})
		);
	});
});
