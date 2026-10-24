import { beforeEach, describe, expect, it } from 'vitest';
import { clearServerUrl, setServerUrl } from '$lib/platform/server-url';
import type { Category, Debt } from '$lib/api/client';
import {
	onCategoryCreated,
	onCategoryDeleted,
	onCategoryUpdated,
	ensureMerchantsTagsFromTransaction,
	onMerchantDeleted,
	ensureSubcategoryInCacheFromTransaction,
	onSubcategoryCreated,
	onSubcategoryDeleted,
	onDebtCreated,
	onDebtDeleted,
	onDebtUpdated,
	patchRefCacheList
} from '$lib/offline/ref-cache-mutations';
import {
	categoriesRefPath,
	persistStableOfflineCatalog,
	publishRefCachePath,
	readRefCache,
	readRefCacheActive,
	readStableOfflineCatalog,
	reconcileOfflineCatalogsOnUnlock,
	refCacheTick,
	resetRefCacheForTests,
	subcategoriesRefPath,
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

	it('prepends active debt and recomputes summary', () => {
		writeRefCache('/api/v1/debts?settled=false', [activeDebt('d-old')]);
		writeRefCache('/api/v1/debts/summary', { total_lent: 1 });

		onDebtCreated(activeDebt('d-new'));

		const list = readRefCache<Debt[]>('/api/v1/debts?settled=false');
		expect(list?.map((d) => d.id)).toEqual(['d-new', 'd-old']);
		expect(readRefCache('/api/v1/debts/summary')).toMatchObject({
			owed_to_me: 200_000,
			active_count: 2
		});
		expect(readRefCache('/api/v1/debtors/d1')).toMatchObject({ id: 'd1', name: 'Иван' });
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

describe('ensureMerchantsTagsFromTransaction', () => {
	beforeEach(() => {
		resetRefCacheForTests();
	});

	it('adds merchant and tags to lists and ui/meta', () => {
		writeRefCache('/api/v1/merchants', [
			{ id: 'old-m', name: 'Старый', icon: 'default', created_at: '2026-01-01T00:00:00Z' }
		]);
		writeRefCache('/api/v1/tags', [
			{ id: 'old-t', name: 'old', created_at: '2026-01-01T00:00:00Z' }
		]);
		writeRefCache('/api/v1/ui/meta', {
			merchants: [
				{ id: 'old-m', name: 'Старый', icon: 'default', created_at: '2026-01-01T00:00:00Z' }
			],
			tags: [{ id: 'old-t', name: 'old', created_at: '2026-01-01T00:00:00Z' }],
			expense_categories: [],
			income_categories: []
		});

		ensureMerchantsTagsFromTransaction({
			id: 'tx1',
			account_id: 'a1',
			type: 'expense',
			kind: 'manual',
			amount: 100,
			amount_display: '1.00',
			description: null,
			merchant_id: 'm-new',
			merchant_name: 'Пятёрочка',
			merchant_icon: 'shopping',
			tags: [{ id: 't-new', name: 'еда' }],
			category_id: null,
			subcategory_id: null,
			transaction_date: '2026-07-08 10:00:00',
			created_at: '2026-07-08T10:00:00Z',
			updated_at: '2026-07-08T10:00:00Z'
		});

		expect(readRefCache<{ id: string }[]>('/api/v1/merchants')?.map((m) => m.id)).toEqual([
			'm-new',
			'old-m'
		]);
		expect(
			readRefCache<{ id: string; icon: string }[]>('/api/v1/merchants')?.find(
				(m) => m.id === 'm-new'
			)?.icon
		).toBe('shopping');
		expect(readRefCache<{ id: string }[]>('/api/v1/tags')?.map((t) => t.id)).toEqual([
			't-new',
			'old-t'
		]);
		expect(
			readRefCache<{ merchants: { id: string }[] }>('/api/v1/ui/meta')?.merchants.map((m) => m.id)
		).toEqual(['m-new', 'old-m']);
	});

	it('creates local merchant from merchantName when id is missing', () => {
		writeRefCache('/api/v1/merchants', []);
		ensureMerchantsTagsFromTransaction(
			{
				id: 'tx1',
				account_id: 'a1',
				type: 'expense',
				kind: 'manual',
				amount: 100,
				amount_display: '1.00',
				description: null,
				category_id: null,
				subcategory_id: null,
				transaction_date: '2026-07-08 10:00:00',
				created_at: '2026-07-08T10:00:00Z',
				updated_at: '2026-07-08T10:00:00Z'
			},
			{ merchantName: 'Новый', tagNames: ['отпуск'] }
		);

		const merchants = readRefCache<{ id: string; name: string }[]>('/api/v1/merchants');
		expect(merchants).toHaveLength(1);
		expect(merchants?.[0]?.name).toBe('Новый');
		expect(merchants?.[0]?.id.startsWith('local:')).toBe(true);
		expect(readRefCache<{ name: string }[]>('/api/v1/tags')?.[0]?.name).toBe('отпуск');
	});

	it('does not create a local merchant when the name matches an alias', () => {
		writeRefCache('/api/v1/merchants', [
			{
				id: 'm1',
				name: 'Пятёрочка',
				icon: 'default',
				aliases: ['Pyaterochka'],
				created_at: '2026-01-01T00:00:00Z'
			}
		]);
		ensureMerchantsTagsFromTransaction(
			{
				id: 'tx2',
				account_id: 'a1',
				type: 'expense',
				kind: 'manual',
				amount: 100,
				amount_display: '1.00',
				description: null,
				category_id: null,
				subcategory_id: null,
				transaction_date: '2026-07-08 10:00:00',
				created_at: '2026-07-08T10:00:00Z',
				updated_at: '2026-07-08T10:00:00Z'
			},
			{ merchantName: 'pyaterochka' }
		);
		const merchants = readRefCache<{ id: string }[]>('/api/v1/merchants');
		expect(merchants?.map((m) => m.id)).toEqual(['m1']);
	});
});

describe('subcategory ref-cache mutations', () => {
	beforeEach(() => {
		resetRefCacheForTests();
	});

	it('onSubcategoryCreated prepends to category subcategory list and bumps count', () => {
		const cat = expenseCat('c1', 'Развлечения');
		cat.subcategory_count = 0;
		writeRefCache('/api/v1/ui/meta', {
			expense_categories: [cat],
			income_categories: []
		});
		seedCategoryLists([cat]);

		onSubcategoryCreated({
			id: 's1',
			category_id: 'c1',
			name: 'Карусели',
			icon: 'fun',
			sort_order: 1,
			created_at: '2026-09-01T00:00:00Z'
		});

		expect(readRefCache<{ id: string; name: string }[]>(subcategoriesRefPath('c1'))).toEqual([
			{
				id: 's1',
				category_id: 'c1',
				name: 'Карусели',
				icon: 'fun',
				sort_order: 1,
				created_at: '2026-09-01T00:00:00Z'
			}
		]);
		expect(
			readRefCache<{ subcategory_count: number }[]>(categoriesRefPath('expense'))?.[0]
		).toMatchObject({
			subcategory_count: 1
		});
	});

	it('ensureSubcategoryInCacheFromTransaction adds inline subcategory from transaction', () => {
		writeRefCache(subcategoriesRefPath('c1'), []);
		ensureSubcategoryInCacheFromTransaction({
			category_id: 'c1',
			subcategory_id: 's-new',
			subcategory_name: 'Карусели',
			subcategory_icon: 'fun'
		});
		expect(
			readRefCache<{ id: string; name: string }[]>(subcategoriesRefPath('c1'))?.[0]
		).toMatchObject({
			id: 's-new',
			name: 'Карусели'
		});
	});

	it('onSubcategoryDeleted removes row and decrements count', () => {
		const cat = expenseCat('c1', 'Развлечения');
		cat.subcategory_count = 1;
		writeRefCache(subcategoriesRefPath('c1'), [
			{
				id: 's1',
				category_id: 'c1',
				name: 'Карусели',
				icon: 'fun',
				sort_order: 1,
				created_at: '2026-09-01T00:00:00Z'
			}
		]);
		writeRefCache('/api/v1/ui/meta', {
			expense_categories: [cat],
			income_categories: []
		});
		seedCategoryLists([cat]);

		onSubcategoryDeleted('c1', 's1');

		expect(readRefCache(subcategoriesRefPath('c1'))).toEqual([]);
		expect(
			readRefCache<{ subcategory_count: number }[]>(categoriesRefPath('expense'))?.[0]
		).toMatchObject({
			subcategory_count: 0
		});
	});
});

function seedCategoryLists(categories: Category[]): void {
	writeRefCache(categoriesRefPath('expense'), categories);
	writeRefCache(categoriesRefPath(), categories);
}

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

type CachedMerchant = { id: string; name: string; icon: string; created_at: string };

function cachedMerchant(id: string, name = id): CachedMerchant {
	return { id, name, icon: 'default', created_at: '2026-01-01T00:00:00Z' };
}

describe('onMerchantDeleted', () => {
	beforeEach(() => {
		resetRefCacheForTests();
	});

	it('overwrites the merchants snapshot with the screen list and drops the id from meta and the stable catalog', () => {
		const keep = cachedMerchant('keep', 'Keep');
		const drop = cachedMerchant('drop', 'Drop');
		const extra = cachedMerchant('extra', 'Extra');
		writeRefCache('/api/v1/merchants', [keep, drop, extra]);
		writeRefCache('/api/v1/ui/meta', {
			expense_categories: [],
			income_categories: [],
			merchants: [keep, drop, extra],
			tags: [{ id: 't1' }]
		});
		const accounts = [
			{ id: 'a1', name: 'Наличные', type: 'cash' as const, status: 'active' as const }
		];
		const debtors = [{ id: 'd1', name: 'Иван' }];
		const transactionTemplates = [{ id: 'tpl1', name: 'Кофе' }];
		persistStableOfflineCatalog({
			accounts,
			expense_categories: [{ id: 'c1' }],
			income_categories: [{ id: 'c2' }],
			merchants: [keep, drop, extra],
			tags: [{ id: 't1' }],
			banks: [{ id: 'b1' }],
			debtors,
			transaction_templates: transactionTemplates
		});

		onMerchantDeleted('drop', [keep]);

		expect(readRefCache<CachedMerchant[]>('/api/v1/merchants')).toEqual([keep]);
		expect(
			readRefCache<{ merchants: CachedMerchant[]; tags: { id: string }[] }>(
				'/api/v1/ui/meta'
			)?.merchants.map((row) => row.id)
		).toEqual(['keep', 'extra']);
		expect(readRefCache<{ tags: { id: string }[] }>('/api/v1/ui/meta')?.tags).toEqual([
			{ id: 't1' }
		]);
		const stable = readStableOfflineCatalog();
		expect(stable?.merchants).toEqual([keep, extra]);
		expect(stable?.expense_categories).toEqual([{ id: 'c1' }]);
		expect(stable?.income_categories).toEqual([{ id: 'c2' }]);
		expect(stable?.tags).toEqual([{ id: 't1' }]);
		expect(stable?.banks).toEqual([{ id: 'b1' }]);
		expect(stable?.accounts).toEqual(accounts);
		expect(stable?.debtors).toEqual(debtors);
		expect(stable?.transaction_templates).toEqual(transactionTemplates);
	});

	it('stores an empty screen list and removes the last id from meta and the stable catalog', () => {
		const only = cachedMerchant('only', 'Only');
		writeRefCache('/api/v1/merchants', [only]);
		writeRefCache('/api/v1/ui/meta', {
			expense_categories: [],
			income_categories: [],
			merchants: [only]
		});
		persistStableOfflineCatalog({
			expense_categories: [],
			income_categories: [],
			merchants: [only]
		});

		onMerchantDeleted('only', []);

		expect(readRefCache<CachedMerchant[]>('/api/v1/merchants')).toEqual([]);
		expect(readRefCache<{ merchants: CachedMerchant[] }>('/api/v1/ui/meta')?.merchants).toEqual([]);
		expect(readStableOfflineCatalog()?.merchants).toEqual([]);
	});

	it('does not resurrect the deleted merchant when catalogs are reconciled on unlock', () => {
		const keep = cachedMerchant('keep', 'Keep');
		const drop = cachedMerchant('drop', 'Drop');
		const meta = {
			expense_categories: [{ id: 'c1' }],
			income_categories: [] as unknown[],
			merchants: [keep, drop],
			tags: [] as unknown[],
			banks: [] as unknown[]
		};
		writeRefCache('/api/v1/ui/meta', meta);
		writeRefCache('/api/v1/merchants', [keep, drop]);
		persistStableOfflineCatalog(meta);

		onMerchantDeleted('drop', [keep]);
		reconcileOfflineCatalogsOnUnlock();

		expect(readRefCache<CachedMerchant[]>('/api/v1/merchants')).toEqual([keep]);
		expect(readRefCache<{ merchants: CachedMerchant[] }>('/api/v1/ui/meta')?.merchants).toEqual([
			keep
		]);
		expect(readStableOfflineCatalog()?.merchants).toEqual([keep]);
	});

	it('does not copy ui/meta from another server origin onto the active one', () => {
		const drop = cachedMerchant('drop', 'Drop');
		const foreign = cachedMerchant('foreign', 'Foreign');
		const keep = cachedMerchant('keep', 'Keep');
		try {
			setServerUrl('http://192.168.8.2:8765');
			writeRefCache('/api/v1/ui/meta', {
				expense_categories: [],
				income_categories: [],
				merchants: [foreign, drop]
			});
			setServerUrl('http://192.168.8.3:8765');
			expect(readRefCacheActive('/api/v1/ui/meta')).toBeNull();

			onMerchantDeleted('drop', [keep]);

			expect(readRefCacheActive('/api/v1/ui/meta')).toBeNull();
			expect(readRefCacheActive<CachedMerchant[]>('/api/v1/merchants')).toEqual([keep]);
			setServerUrl('http://192.168.8.2:8765');
			expect(
				readRefCacheActive<{ merchants: { id: string }[] }>('/api/v1/ui/meta')?.merchants.map(
					(row) => row.id
				)
			).toEqual(['foreign', 'drop']);
		} finally {
			clearServerUrl();
		}
	});

	it('leaves a stable catalog alone when it does not contain the deleted id', () => {
		const keep = cachedMerchant('keep', 'Keep');
		persistStableOfflineCatalog({
			expense_categories: [{ id: 'c1' }],
			income_categories: [],
			merchants: [keep]
		});
		const before = readStableOfflineCatalog();

		onMerchantDeleted('drop', [keep]);

		expect(readRefCache<CachedMerchant[]>('/api/v1/merchants')).toEqual([keep]);
		expect(readStableOfflineCatalog()).toBe(before);
	});
});
