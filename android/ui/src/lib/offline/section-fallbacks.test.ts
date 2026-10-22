import { beforeEach, describe, expect, it } from 'vitest';
import type { Debt, DebtorDetail } from '$lib/api/client';
import { resetRefCacheForTests, writeRefCache } from '$lib/offline/ref-cache';
import {
	findCachedCredit,
	findCachedDebt,
	recomputeDebtsSummaryFromCache,
	resolveDebtorDetailOffline
} from '$lib/offline/section-fallbacks';

const debt = (overrides: Partial<Debt> = {}): Debt => ({
	id: 'debt-1',
	debtor_id: 'person-1',
	debtor_name: 'Иван',
	direction: 'lent',
	amount: 150_000,
	amount_display: '1500.00',
	affects_balance: false,
	debt_date: '2026-07-08 10:00:00',
	due_date: '2026-07-15 23:59:59',
	description: null,
	transaction_id: null,
	is_settled: false,
	settled_at: null,
	is_overdue: false,
	created_at: '2026-07-08T07:00:00Z',
	...overrides
});

describe('section-fallbacks', () => {
	beforeEach(() => {
		resetRefCacheForTests();
	});

	it('builds debtor detail from debt lists when the card was never cached', () => {
		writeRefCache('/api/v1/debtors', [
			{ id: 'person-1', name: 'Иван', created_at: '2026-01-01T00:00:00Z' }
		]);
		writeRefCache('/api/v1/debts?settled=false', [debt()]);
		writeRefCache('/api/v1/debts?settled=true', [
			debt({ id: 'debt-2', direction: 'borrowed', amount: 40_000, is_settled: true })
		]);

		expect(resolveDebtorDetailOffline('person-1')).toMatchObject({
			id: 'person-1',
			name: 'Иван',
			i_owe: 0,
			owed_to_me: 150_000,
			debts: [{ id: 'debt-1' }, { id: 'debt-2' }]
		});
	});

	it('rebuilds totals from lists and keeps cached related transactions', () => {
		const cached: DebtorDetail = {
			id: 'person-1',
			name: 'Старое',
			created_at: '2026-01-01T00:00:00Z',
			i_owe: 0,
			owed_to_me: 0,
			debts: [],
			transactions: [
				{
					id: 'tx-1',
					account_id: 'a1',
					type: 'expense',
					kind: 'manual',
					amount: 150_000,
					amount_display: '1500.00',
					description: null,
					transaction_date: '2026-07-08 10:00:00',
					deletable: true
				}
			]
		};
		writeRefCache('/api/v1/debtors/person-1', cached);
		writeRefCache('/api/v1/debts?settled=false', [debt({ direction: 'borrowed', amount: 80_000 })]);

		const next = resolveDebtorDetailOffline('person-1');
		expect(next?.i_owe).toBe(80_000);
		expect(next?.owed_to_me).toBe(0);
		expect(next?.transactions).toHaveLength(1);
		expect(next?.name).toBe('Иван');
	});

	it('returns null when there is no debtor, list row, or cached card', () => {
		expect(resolveDebtorDetailOffline('missing')).toBeNull();
	});

	it('finds a debt in either settled list', () => {
		writeRefCache('/api/v1/debts?settled=true', [debt({ id: 'settled-1', is_settled: true })]);
		expect(findCachedDebt('settled-1')?.id).toBe('settled-1');
	});

	it('recomputes debts summary from the active list', () => {
		writeRefCache('/api/v1/debts?settled=false', [
			debt({ amount: 100_000, direction: 'lent' }),
			debt({ id: 'd2', amount: 25_000, direction: 'borrowed', is_overdue: true })
		]);
		writeRefCache('/api/v1/debts?settled=true', []);
		expect(recomputeDebtsSummaryFromCache()).toEqual({
			i_owe: 25_000,
			owed_to_me: 100_000,
			overdue_i_owe: 25_000,
			overdue_owed_to_me: 0,
			active_count: 2
		});
	});

	it('finds a credit in the list when the detail key is missing', () => {
		writeRefCache('/api/v1/credits?status=active', [{ id: 'c1', name: 'Ипотека' }]);
		expect(findCachedCredit('c1')).toMatchObject({ id: 'c1', name: 'Ипотека' });
	});
});
