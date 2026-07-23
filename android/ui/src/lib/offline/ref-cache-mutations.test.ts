import { beforeEach, describe, expect, it } from 'vitest';
import type { Category, Debt } from '$lib/api/client';
import {
	onCategoryCreated,
	onCategoryDeleted,
	onCategoryUpdated,
	onDebtCreated,
	onDebtDeleted,
	onDebtUpdated,
	patchRefCacheList
} from '$lib/offline/ref-cache-mutations';
import {
	categoriesRefPath,
	publishRefCachePath,
	readRefCache,
	refCacheReady,
	refCacheTick,
	resetRefCacheForTests,
	writeRefCache
} from '$lib/offline/ref-cache';

const expenseCat = (id: string, name: string): Category => ({
	id,
	name,
	type: 'expense',
	icon: 'food',
	sort_order: 0,
	is_primary: false,
	is_system: false,
	subcategory_count: 0,
	created_at: '2026-01-01T00:00:00Z'
});

const activeDebt = (id: string): Debt => ({
	id,
	debtor_id: 'd1',
	debtor_name: 'Иван',
	direction: 'lent',
	amount: 100_000,
	amount_display: '1000.00',
	affects_balance: false,
	debt_date: '2026-07-08 10:00:00',
	due_date: '2026-07-15 23:59:59',
	description: null,
	transaction_id: null,
	is_settled: false,
	settled_at: null,
	is_overdue: false,
	created_at: '2026-07-08T10:00:00Z'
});

describe('ref-cache-mutations categories', () => {
	beforeEach(() => {
		resetRefCacheForTests();
	});

	it('prepends created category to typed list cache', () => {
		writeRefCache(categoriesRefPath('expense'), [expenseCat('c1', 'Еда')]);
		onCategoryCreated(expenseCat('c2', 'Транспорт'));

		const list = readRefCache<Category[]>(categoriesRefPath('expense'));
		expect(list?.map((c) => c.id)).toEqual(['c2', 'c1']);
	});

	it('updates category in list and ui/meta caches', () => {
		writeRefCache(categoriesRefPath('expense'), [expenseCat('c1', 'Еда')]);
		writeRefCache('/api/v1/ui/meta', {
			expense_categories: [expenseCat('c1', 'Еда')],
			income_categories: []
		});

		onCategoryUpdated(expenseCat('c1', 'Продукты'));

		expect(readRefCache<Category[]>(categoriesRefPath('expense'))?.[0]?.name).toBe('Продукты');
		expect(
			readRefCache<{ expense_categories: Category[] }>('/api/v1/ui/meta')?.expense_categories[0]
				?.name
		).toBe('Продукты');
	});

	it('removes category from list and ui/meta caches', () => {
		writeRefCache(categoriesRefPath('expense'), [expenseCat('c1', 'Еда')]);
		writeRefCache('/api/v1/ui/meta', {
			expense_categories: [expenseCat('c1', 'Еда')],
			income_categories: []
		});

		onCategoryDeleted('c1', 'expense');

		expect(readRefCache<Category[]>(categoriesRefPath('expense'))).toEqual([]);
		expect(
			readRefCache<{ expense_categories: Category[] }>('/api/v1/ui/meta')?.expense_categories
		).toEqual([]);
	});
});

describe('ref-cache-mutations debts', () => {
	beforeEach(() => {
		resetRefCacheForTests();
	});

	it('prepends active debt and invalidates summary', () => {
		writeRefCache('/api/v1/debts?settled=false', [activeDebt('d-old')]);
		writeRefCache('/api/v1/debts/summary', { total_lent: 1 });

		onDebtCreated(activeDebt('d-new'));

		const list = readRefCache<Debt[]>('/api/v1/debts?settled=false');
		expect(list?.map((d) => d.id)).toEqual(['d-new', 'd-old']);
		expect(refCacheReady('/api/v1/debts/summary')).toBe(false);
	});

	it('adds new debtor to debtors list and ui/meta when debt is created', () => {
		writeRefCache('/api/v1/debtors', [
			{ id: 'old', name: 'Петр', created_at: '2026-01-01T00:00:00Z' }
		]);
		writeRefCache('/api/v1/ui/meta', {
			debtors: [{ id: 'old', name: 'Петр', created_at: '2026-01-01T00:00:00Z' }],
			expense_categories: [],
			income_categories: []
		});
		writeRefCache('/api/v1/debts?settled=false', []);

		onDebtCreated({
			...activeDebt('d-new'),
			debtor_id: 'debtor-new',
			debtor_name: 'Иван'
		});

		expect(
			readRefCache<{ id: string; name: string }[]>('/api/v1/debtors')?.map((d) => d.id)
		).toEqual(['debtor-new', 'old']);
		expect(
			readRefCache<{ debtors: { id: string }[] }>('/api/v1/ui/meta')?.debtors.map((d) => d.id)
		).toEqual(['debtor-new', 'old']);
	});

	it('does not duplicate debtor when debt reuses existing id', () => {
		writeRefCache('/api/v1/debtors', [
			{ id: 'debtor-1', name: 'Иван', created_at: '2026-01-01T00:00:00Z' }
		]);
		writeRefCache('/api/v1/debts?settled=false', []);

		onDebtCreated({ ...activeDebt('d-new'), debtor_id: 'debtor-1', debtor_name: 'Иван' });

		expect(readRefCache('/api/v1/debtors')).toHaveLength(1);
	});

	it('moves debt to settled list on update', () => {
		writeRefCache('/api/v1/debts?settled=false', [activeDebt('d1')]);
		writeRefCache('/api/v1/debts?settled=true', []);

		onDebtUpdated({ ...activeDebt('d1'), is_settled: true, settled_at: '2026-07-10T00:00:00Z' });

		expect(readRefCache<Debt[]>('/api/v1/debts?settled=false')).toEqual([]);
		expect(readRefCache<Debt[]>('/api/v1/debts?settled=true')?.[0]?.is_settled).toBe(true);
	});

	it('removes debt from both lists', () => {
		writeRefCache('/api/v1/debts?settled=false', [activeDebt('d1')]);
		writeRefCache('/api/v1/debts?settled=true', []);

		onDebtDeleted('d1');

		expect(readRefCache<Debt[]>('/api/v1/debts?settled=false')).toEqual([]);
		expect(readRefCache<Debt[]>('/api/v1/debts?settled=true')).toEqual([]);
	});
});

describe('patchRefCacheList', () => {
	beforeEach(() => {
		resetRefCacheForTests();
	});

	it('returns false when cache is empty', () => {
		expect(patchRefCacheList('/api/v1/missing', (list) => list)).toBe(false);
	});

	it('publishRefCachePath bumps refCacheTick', () => {
		let tick = 0;
		const unsub = refCacheTick.subscribe((n) => (tick = n));
		publishRefCachePath('/api/v1/test', { ok: true });
		expect(tick).toBe(1);
		expect(readRefCache('/api/v1/test')).toEqual({ ok: true });
		unsub();
	});
});
