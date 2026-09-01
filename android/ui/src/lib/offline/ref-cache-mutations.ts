import type {
	Account,
	BudgetItem,
	Category,
	Credit,
	Debt,
	Debtor,
	Merchant,
	RecurringOperation,
	Subcategory,
	Subscription,
	Tag,
	Transaction,
	UIMetaAccountRef
} from '$lib/api/client';
import {
	categoriesRefPath,
	invalidateRefCache,
	invalidateRefCachePrefix,
	publishRefCachePath,
	readRefCache,
	subcategoriesRefPath
} from '$lib/offline/ref-cache';
import { makeLocalKey } from '$lib/offline/types';

const UI_META_PATH = '/api/v1/ui/meta';
const DEBTORS_PATH = '/api/v1/debtors';
const MERCHANTS_PATH = '/api/v1/merchants';
const TAGS_PATH = '/api/v1/tags';
const ACCOUNTS_ACTIVE = '/api/v1/accounts?status=active';
const ACCOUNTS_ARCHIVED = '/api/v1/accounts?status=archived';
const ACCOUNTS_ALL = '/api/v1/accounts';
const CREDITS_ACTIVE = '/api/v1/credits?status=active';
const CREDITS_CLOSED = '/api/v1/credits?status=closed';
const RECURRING_PATH = '/api/v1/recurring-operations';
const SUBSCRIPTIONS_PATH = '/api/v1/subscriptions';

export function creditDetailPath(id: string): string {
	return `/api/v1/credits/${id}`;
}

export function patchRefCacheList<T>(path: string, mutator: (list: T[]) => T[]): boolean {
	const cached = readRefCache<T[]>(path);
	if (cached === null) return false;
	publishRefCachePath(path, mutator([...cached]));
	return true;
}

export function prependRefCacheList<T>(path: string, item: T): boolean {
	const cached = readRefCache<T[]>(path);
	if (cached === null) {
		publishRefCachePath(path, [item]);
		return true;
	}
	publishRefCachePath(path, [item, ...cached]);
	return true;
}

export function replaceRefCacheListItem<T extends { id: string }>(path: string, item: T): boolean {
	return patchRefCacheList<T>(path, (list) => list.map((row) => (row.id === item.id ? item : row)));
}

export function removeRefCacheListItem<T extends { id: string }>(
	path: string,
	id: string
): boolean {
	return patchRefCacheList<T>(path, (list) => list.filter((row) => row.id !== id));
}

function readUIMetaCategories(): {
	expense: Category[];
	income: Category[];
} | null {
	const meta = readRefCache<{
		expense_categories: Category[];
		income_categories: Category[];
	}>(UI_META_PATH);
	if (!meta) return null;
	return { expense: meta.expense_categories, income: meta.income_categories };
}

function writeUIMetaCategories(expense: Category[], income: Category[]): void {
	const meta = readRefCache<Record<string, unknown>>(UI_META_PATH);
	if (!meta) return;
	publishRefCachePath(UI_META_PATH, {
		...meta,
		expense_categories: expense,
		income_categories: income
	});
	seedCategoryListCaches(expense, income);
}

function seedCategoryListCaches(expense: Category[], income: Category[]): void {
	publishRefCachePath(categoriesRefPath('expense'), expense);
	publishRefCachePath(categoriesRefPath('income'), income);
	publishRefCachePath(categoriesRefPath(), [...expense, ...income]);
}

export function onCategoryCreated(category: Category): void {
	const typePath = categoriesRefPath(category.type);
	prependRefCacheList(typePath, category);
	prependRefCacheList(categoriesRefPath(), category);
	const meta = readUIMetaCategories();
	if (meta) {
		if (category.type === 'expense') {
			writeUIMetaCategories([category, ...meta.expense], meta.income);
		} else {
			writeUIMetaCategories(meta.expense, [category, ...meta.income]);
		}
	}
}

export function onCategoryUpdated(category: Category): void {
	const typePath = categoriesRefPath(category.type);
	replaceRefCacheListItem(typePath, category);
	replaceRefCacheListItem(categoriesRefPath(), category);
	const meta = readUIMetaCategories();
	if (meta) {
		if (category.type === 'expense') {
			writeUIMetaCategories(
				meta.expense.map((c) => (c.id === category.id ? category : c)),
				meta.income
			);
		} else {
			writeUIMetaCategories(
				meta.expense,
				meta.income.map((c) => (c.id === category.id ? category : c))
			);
		}
	}
}

export function onCategoryDeleted(id: string, type: 'income' | 'expense'): void {
	removeRefCacheListItem(categoriesRefPath(type), id);
	removeRefCacheListItem(categoriesRefPath(), id);
	invalidateRefCache(subcategoriesRefPath(id));
	const meta = readUIMetaCategories();
	if (meta) {
		if (type === 'expense') {
			writeUIMetaCategories(
				meta.expense.filter((c) => c.id !== id),
				meta.income
			);
		} else {
			writeUIMetaCategories(
				meta.expense,
				meta.income.filter((c) => c.id !== id)
			);
		}
	}
}

function bumpCategorySubcategoryCount(categoryId: string, delta: number): void {
	const patch = (list: Category[]) =>
		list.map((c) =>
			c.id === categoryId
				? { ...c, subcategory_count: Math.max(0, c.subcategory_count + delta) }
				: c
		);
	patchRefCacheList<Category>(categoriesRefPath(), patch);
	const meta = readUIMetaCategories();
	if (!meta) return;
	const inExpense = meta.expense.some((c) => c.id === categoryId);
	if (inExpense) {
		writeUIMetaCategories(patch(meta.expense), meta.income);
	} else {
		writeUIMetaCategories(meta.expense, patch(meta.income));
	}
}

export function onSubcategoryCreated(sub: Subcategory): void {
	const path = subcategoriesRefPath(sub.category_id);
	const cached = readRefCache<Subcategory[]>(path);
	if (cached?.some((row) => row.id === sub.id)) {
		replaceRefCacheListItem(path, sub);
		return;
	}
	prependRefCacheList(path, sub);
	bumpCategorySubcategoryCount(sub.category_id, 1);
}

export function onSubcategoryUpdated(sub: Subcategory): void {
	replaceRefCacheListItem(subcategoriesRefPath(sub.category_id), sub);
}

export function onSubcategoriesReordered(categoryId: string, subs: Subcategory[]): void {
	publishRefCachePath(subcategoriesRefPath(categoryId), subs);
}

export function onSubcategoryDeleted(categoryId: string, subId: string): void {
	removeRefCacheListItem<Subcategory>(subcategoriesRefPath(categoryId), subId);
	bumpCategorySubcategoryCount(categoryId, -1);
}

/** Patch subcategory list when a transaction creates or links a subcategory inline. */
export function ensureSubcategoryInCacheFromTransaction(
	tx: Pick<Transaction, 'category_id' | 'subcategory_id' | 'subcategory_name' | 'subcategory_icon'>
): void {
	if (!tx.category_id || !tx.subcategory_id || !tx.subcategory_name) return;
	const sub: Subcategory = {
		id: tx.subcategory_id,
		category_id: tx.category_id,
		name: tx.subcategory_name,
		icon: tx.subcategory_icon ?? 'default',
		sort_order: 0,
		created_at: new Date().toISOString()
	};
	onSubcategoryCreated(sub);
}

export function onDebtCreated(debt: Debt): void {
	if (!debt.is_settled) {
		prependRefCacheList('/api/v1/debts?settled=false', debt);
	} else {
		prependRefCacheList('/api/v1/debts?settled=true', debt);
	}
	invalidateRefCache('/api/v1/debts/summary');
	ensureDebtorInCache(debt);
}

function ensureDebtorInCache(debt: Debt): void {
	if (!debt.debtor_id || !debt.debtor_name) return;
	const debtor: Debtor = {
		id: debt.debtor_id,
		name: debt.debtor_name,
		created_at: debt.created_at
	};
	const list = readRefCache<Debtor[]>(DEBTORS_PATH);
	if (list !== null && !list.some((row) => row.id === debtor.id)) {
		prependRefCacheList(DEBTORS_PATH, debtor);
	}
	const meta = readRefCache<{ debtors: Debtor[] } & Record<string, unknown>>(UI_META_PATH);
	if (meta && !meta.debtors.some((row) => row.id === debtor.id)) {
		publishRefCachePath(UI_META_PATH, {
			...meta,
			debtors: [debtor, ...meta.debtors]
		});
	}
}

function ensureMerchantInCache(merchant: Merchant): void {
	const list = readRefCache<Merchant[]>(MERCHANTS_PATH);
	if (list !== null && !list.some((row) => row.id === merchant.id)) {
		prependRefCacheList(MERCHANTS_PATH, merchant);
	} else if (list === null) {
		publishRefCachePath(MERCHANTS_PATH, [merchant]);
	}
	const meta = readRefCache<{ merchants?: Merchant[] } & Record<string, unknown>>(UI_META_PATH);
	if (meta) {
		const merchants = meta.merchants ?? [];
		if (!merchants.some((row) => row.id === merchant.id)) {
			publishRefCachePath(UI_META_PATH, {
				...meta,
				merchants: [merchant, ...merchants]
			});
		}
	}
}

function ensureTagInCache(tag: Tag): void {
	const list = readRefCache<Tag[]>(TAGS_PATH);
	if (list !== null && !list.some((row) => row.id === tag.id)) {
		prependRefCacheList(TAGS_PATH, tag);
	} else if (list === null) {
		publishRefCachePath(TAGS_PATH, [tag]);
	}
	const meta = readRefCache<{ tags?: Tag[] } & Record<string, unknown>>(UI_META_PATH);
	if (meta) {
		const tags = meta.tags ?? [];
		if (!tags.some((row) => row.id === tag.id)) {
			publishRefCachePath(UI_META_PATH, {
				...meta,
				tags: [tag, ...tags]
			});
		}
	}
}

/** Patch merchants/tags ref-cache when a transaction creates or links them. */
export function ensureMerchantsTagsFromTransaction(
	tx: Transaction,
	opts?: { merchantName?: string; tagNames?: string[] }
): void {
	const createdAt = tx.created_at;
	if (tx.merchant_id && (tx.merchant_name || opts?.merchantName)) {
		ensureMerchantInCache({
			id: tx.merchant_id,
			name: tx.merchant_name || opts?.merchantName || '',
			icon: tx.merchant_icon || 'default',
			created_at: createdAt
		});
	} else if (opts?.merchantName) {
		const existing = readRefCache<Merchant[]>(MERCHANTS_PATH);
		const byName = existing?.find((m) => m.name.toLowerCase() === opts.merchantName!.toLowerCase());
		if (!byName) {
			ensureMerchantInCache({
				id: makeLocalKey(),
				name: opts.merchantName,
				icon: 'default',
				created_at: createdAt
			});
		}
	}
	for (const tag of tx.tags ?? []) {
		if (!tag.id && !tag.name) continue;
		if (tag.id) {
			ensureTagInCache({
				id: tag.id,
				name: tag.name,
				created_at: createdAt
			});
		}
	}
	for (const name of opts?.tagNames ?? []) {
		const trimmed = name.trim();
		if (!trimmed) continue;
		const existing = readRefCache<Tag[]>(TAGS_PATH);
		const byName = existing?.find((t) => t.name.toLowerCase() === trimmed.toLowerCase());
		if (!byName) {
			ensureTagInCache({
				id: makeLocalKey(),
				name: trimmed,
				created_at: createdAt
			});
		}
	}
}

export function onDebtUpdated(debt: Debt): void {
	const activePath = '/api/v1/debts?settled=false';
	const settledPath = '/api/v1/debts?settled=true';
	removeRefCacheListItem<Debt>(activePath, debt.id);
	removeRefCacheListItem<Debt>(settledPath, debt.id);
	if (debt.is_settled) {
		prependRefCacheList(settledPath, debt);
	} else {
		prependRefCacheList(activePath, debt);
	}
	invalidateRefCache('/api/v1/debts/summary');
}

export function onDebtDeleted(id: string): void {
	removeRefCacheListItem<Debt>('/api/v1/debts?settled=false', id);
	removeRefCacheListItem<Debt>('/api/v1/debts?settled=true', id);
	invalidateRefCache('/api/v1/debts/summary');
}

/** Optimistic summary bump when exact totals are unknown offline. */
export function touchDebtsSummary(): void {
	invalidateRefCache('/api/v1/debts/summary');
}

function toAccountRef(account: Account): UIMetaAccountRef {
	return {
		id: account.id,
		name: account.name,
		type: account.type,
		status: account.status,
		bank_id: account.bank_id ?? undefined
	};
}

function patchUIMetaAccounts(mutator: (list: UIMetaAccountRef[]) => UIMetaAccountRef[]): void {
	const meta = readRefCache<{ accounts: UIMetaAccountRef[] } & Record<string, unknown>>(
		UI_META_PATH
	);
	if (!meta) return;
	publishRefCachePath(UI_META_PATH, {
		...meta,
		accounts: mutator([...meta.accounts])
	});
}

export function onAccountCreated(account: Account): void {
	prependRefCacheList(ACCOUNTS_ACTIVE, account);
	prependRefCacheList(ACCOUNTS_ALL, account);
	patchUIMetaAccounts((list) => [
		toAccountRef(account),
		...list.filter((a) => a.id !== account.id)
	]);
	invalidateRefCache('/api/v1/dashboard');
}

export function onAccountUpdated(account: Account): void {
	replaceRefCacheListItem(ACCOUNTS_ACTIVE, account);
	replaceRefCacheListItem(ACCOUNTS_ARCHIVED, account);
	replaceRefCacheListItem(ACCOUNTS_ALL, account);
	if (account.status === 'archived') {
		removeRefCacheListItem<Account>(ACCOUNTS_ACTIVE, account.id);
		prependRefCacheList(ACCOUNTS_ARCHIVED, account);
	} else if (account.status === 'active') {
		removeRefCacheListItem<Account>(ACCOUNTS_ARCHIVED, account.id);
		const active = readRefCache<Account[]>(ACCOUNTS_ACTIVE);
		if (active && !active.some((a) => a.id === account.id)) {
			prependRefCacheList(ACCOUNTS_ACTIVE, account);
		}
	}
	patchUIMetaAccounts((list) => {
		const ref = toAccountRef(account);
		const idx = list.findIndex((a) => a.id === account.id);
		if (idx >= 0) {
			const next = [...list];
			next[idx] = ref;
			return next;
		}
		return [ref, ...list];
	});
	invalidateRefCache('/api/v1/dashboard');
}

export function onAccountArchived(account: Account): void {
	removeRefCacheListItem<Account>(ACCOUNTS_ACTIVE, account.id);
	prependRefCacheList(ACCOUNTS_ARCHIVED, { ...account, status: 'archived' });
	replaceRefCacheListItem(ACCOUNTS_ALL, { ...account, status: 'archived' });
	patchUIMetaAccounts((list) =>
		list.map((a) => (a.id === account.id ? { ...a, status: 'archived' as const } : a))
	);
	invalidateRefCache('/api/v1/dashboard');
}

export function onAccountUnarchived(account: Account): void {
	removeRefCacheListItem<Account>(ACCOUNTS_ARCHIVED, account.id);
	prependRefCacheList(ACCOUNTS_ACTIVE, { ...account, status: 'active' });
	replaceRefCacheListItem(ACCOUNTS_ALL, { ...account, status: 'active' });
	patchUIMetaAccounts((list) =>
		list.map((a) => (a.id === account.id ? { ...a, status: 'active' as const } : a))
	);
	invalidateRefCache('/api/v1/dashboard');
}

export function onBudgetCreated(item: BudgetItem, month?: string): void {
	const m = month ?? item.month;
	if (m) {
		prependRefCacheList(`/api/v1/budgets?month=${encodeURIComponent(m)}`, item);
		invalidateRefCache(`/api/v1/budgets/summary?month=${encodeURIComponent(m)}`);
	}
	invalidateRefCache('/api/v1/budgets/summary');
}

export function onBudgetUpdated(item: BudgetItem, month?: string): void {
	const m = month ?? item.month;
	if (m) {
		replaceRefCacheListItem(`/api/v1/budgets?month=${encodeURIComponent(m)}`, item);
		invalidateRefCache(`/api/v1/budgets/summary?month=${encodeURIComponent(m)}`);
	}
	invalidateRefCache('/api/v1/budgets/summary');
}

export function onBudgetDeleted(id: string, month?: string): void {
	if (month) {
		removeRefCacheListItem<BudgetItem>(`/api/v1/budgets?month=${encodeURIComponent(month)}`, id);
		invalidateRefCache(`/api/v1/budgets/summary?month=${encodeURIComponent(month)}`);
	}
	invalidateRefCache('/api/v1/budgets/summary');
}

export function onCreditUpdated(credit: Credit): void {
	publishRefCachePath(creditDetailPath(credit.id), credit);
	removeRefCacheListItem<Credit>(CREDITS_ACTIVE, credit.id);
	removeRefCacheListItem<Credit>(CREDITS_CLOSED, credit.id);
	if (credit.status === 'closed') {
		prependRefCacheList(CREDITS_CLOSED, credit);
	} else {
		prependRefCacheList(CREDITS_ACTIVE, credit);
	}
}

export function onCreditDeleted(id: string): void {
	invalidateRefCache(creditDetailPath(id));
	removeRefCacheListItem<Credit>(CREDITS_ACTIVE, id);
	removeRefCacheListItem<Credit>(CREDITS_CLOSED, id);
}

/** Pay/complete/delete payment may change balances — drop stale snapshots until next warm. */
export function touchBalancesAfterCreditMutation(): void {
	invalidateRefCache('/api/v1/dashboard');
	invalidateRefCache(ACCOUNTS_ACTIVE);
	invalidateRefCache(ACCOUNTS_ALL);
	invalidateRefCache(ACCOUNTS_ARCHIVED);
}

export function onRecurringCreated(item: RecurringOperation): void {
	prependRefCacheList(RECURRING_PATH, item);
}

export function onRecurringUpdated(item: RecurringOperation): void {
	replaceRefCacheListItem(RECURRING_PATH, item);
}

export function onRecurringDeleted(id: string): void {
	removeRefCacheListItem<RecurringOperation>(RECURRING_PATH, id);
}

function touchSubscriptionSummaryCache(): void {
	invalidateRefCachePrefix('/api/v1/subscriptions/summary');
}

export function onSubscriptionCreated(item: Subscription): void {
	prependRefCacheList(SUBSCRIPTIONS_PATH, item);
	touchSubscriptionSummaryCache();
}

export function onSubscriptionUpdated(item: Subscription): void {
	replaceRefCacheListItem(SUBSCRIPTIONS_PATH, item);
	touchSubscriptionSummaryCache();
}

export function onSubscriptionDeleted(id: string): void {
	removeRefCacheListItem<Subscription>(SUBSCRIPTIONS_PATH, id);
	touchSubscriptionSummaryCache();
}
