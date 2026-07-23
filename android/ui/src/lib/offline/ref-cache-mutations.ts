import type {
	Account,
	BudgetItem,
	Category,
	Debt,
	Debtor,
	UIMetaAccountRef
} from '$lib/api/client';
import {
	categoriesRefPath,
	invalidateRefCache,
	publishRefCachePath,
	readRefCache
} from '$lib/offline/ref-cache';

const UI_META_PATH = '/api/v1/ui/meta';
const DEBTORS_PATH = '/api/v1/debtors';
const ACCOUNTS_ACTIVE = '/api/v1/accounts?status=active';
const ACCOUNTS_ARCHIVED = '/api/v1/accounts?status=archived';
const ACCOUNTS_ALL = '/api/v1/accounts';

export function patchRefCacheList<T>(path: string, mutator: (list: T[]) => T[]): boolean {
	const cached = readRefCache<T[]>(path);
	if (cached === null) return false;
	publishRefCachePath(path, mutator([...cached]));
	return true;
}

export function prependRefCacheList<T>(path: string, item: T): boolean {
	return patchRefCacheList<T>(path, (list) => [item, ...list]);
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
