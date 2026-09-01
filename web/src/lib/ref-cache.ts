import { writable } from 'svelte/store';
import type {
	Account,
	AccountBalanceSummary,
	Bank,
	Dashboard,
	UIMeta,
	UIMetaAccountRef
} from '$lib/api/client';
import { isDashboardRefPath, isDashboardShape, stableEqual } from '$lib/state-utils';

const REF_CACHE_VERSION = 'buhgalter.ref_cache.web.v1';

const REF_CACHE_SKIP = new Set([
	'/api/v1/health',
	// Bootstrap flag (registration_enabled) — must not serve a pre-mutation snapshot on /login.
	'/api/v1/setup/status'
]);

const memoryStore = new Map<string, string>();
const inflightRevalidate = new Map<string, Promise<void>>();

let cacheUserId = '_anonymous';
/** Bumped on clear/invalidate so in-flight SWR revalidates don't rewrite stale data. */
let cacheEpoch = 0;

/** Bumped when a background revalidate writes new data — pages reload softly. */
export const refCacheTick = writable(0);

/** Path-aware notification after SWR revalidate. */
export const refCacheUpdate = writable<{ path: string; seq: number } | null>(null);

export function setRefCacheUserId(userId: string | null): void {
	cacheUserId = userId || '_anonymous';
}

function storageKey(path: string): string {
	return `${REF_CACHE_VERSION}::${cacheUserId}::${path}`;
}

function storageGet(key: string): string | null {
	if (typeof localStorage !== 'undefined') {
		try {
			return localStorage.getItem(key);
		} catch {
			// private mode / quota
		}
	}
	return memoryStore.get(key) ?? null;
}

function storageSet(key: string, value: string): void {
	if (typeof localStorage !== 'undefined') {
		try {
			localStorage.setItem(key, value);
			return;
		} catch {
			// quota — fall through to memory
		}
	}
	memoryStore.set(key, value);
}

function storageRemove(key: string): void {
	if (typeof localStorage !== 'undefined') {
		try {
			localStorage.removeItem(key);
		} catch {
			// ignore
		}
	}
	memoryStore.delete(key);
}

export function shouldPersistRefCache(path: string): boolean {
	const pathOnly = path.split('?')[0] ?? path;
	if (REF_CACHE_SKIP.has(pathOnly)) return false;
	if (!pathOnly.startsWith('/api/v1/')) return false;
	if (pathOnly.includes('/preview')) return false;
	if (pathOnly.includes('/import/jobs/')) return false;
	if (pathOnly.startsWith('/api/v1/export')) return false;
	if (pathOnly.startsWith('/api/v1/version')) return false;
	return true;
}

export function isStaleFetchError(err: unknown): boolean {
	if (err instanceof OfflineCacheMissError) return true;
	if (err instanceof TypeError) return true;
	if (err && typeof err === 'object' && 'status' in err) {
		const status = Number((err as { status: number }).status);
		return status === 0 || status === 408 || status === 502 || status === 503 || status === 504;
	}
	return false;
}

export function readRefCache<T>(path: string): T | null {
	const raw = storageGet(storageKey(path));
	if (!raw) return null;
	try {
		return JSON.parse(raw) as T;
	} catch {
		return null;
	}
}

export function refCacheReady(path: string): boolean {
	return readRefCache(path) !== null;
}

export function refCacheReadyAny(paths: string[]): boolean {
	return paths.some(refCacheReady);
}

/** Write cache. Returns false when payload matches existing (no disk/UI churn). */
export function writeRefCache<T>(path: string, value: T): boolean {
	try {
		const key = storageKey(path);
		const prevRaw = memoryStore.get(key) ?? null;
		const nextRaw = JSON.stringify(value);
		if (prevRaw === nextRaw) return false;
		if (isDashboardRefPath(path) && prevRaw !== null) {
			try {
				const prev = JSON.parse(prevRaw) as unknown;
				if (isDashboardShape(prev) && isDashboardShape(value) && stableEqual(prev, value)) {
					return false;
				}
			} catch {
				// fall through to write
			}
		}
		storageSet(key, nextRaw);
		return true;
	} catch {
		return false;
	}
}

export function invalidateRefCache(path: string): void {
	storageRemove(storageKey(path));
}

export function invalidateRefCachePrefix(pathPrefix: string): void {
	const needle = `::${pathPrefix}`;
	const prefix = `${REF_CACHE_VERSION}::`;
	if (typeof localStorage !== 'undefined') {
		try {
			const keys: string[] = [];
			for (let i = 0; i < localStorage.length; i++) {
				const key = localStorage.key(i);
				if (key?.startsWith(prefix) && key.includes(needle)) keys.push(key);
			}
			for (const key of keys) localStorage.removeItem(key);
		} catch {
			// ignore
		}
	}
	for (const key of [...memoryStore.keys()]) {
		if (key.startsWith(prefix) && key.includes(needle)) memoryStore.delete(key);
	}
}

/** Drop preserved offline dictionary cache entries touched by a write (see clearRefCache preserveAuthMe). */
export function invalidateRefCacheAfterWrite(apiPath: string): void {
	const pathOnly = apiPath.split('?')[0] ?? apiPath;
	if (pathOnly.startsWith('/api/v1/categories') || pathOnly.startsWith('/api/v1/subcategories')) {
		invalidateRefCachePrefix('/api/v1/categories');
		return;
	}
	if (pathOnly.startsWith('/api/v1/accounts')) {
		invalidateRefCachePrefix('/api/v1/accounts');
		invalidateRefCache('/api/v1/dashboard');
		return;
	}
	if (pathOnly.startsWith('/api/v1/merchants')) {
		invalidateRefCachePrefix('/api/v1/merchants');
		return;
	}
	if (pathOnly.startsWith('/api/v1/tags')) {
		invalidateRefCachePrefix('/api/v1/tags');
		return;
	}
	if (pathOnly.startsWith('/api/v1/debtors')) {
		invalidateRefCachePrefix('/api/v1/debtors');
		return;
	}
	if (pathOnly.startsWith('/api/v1/transaction-templates')) {
		invalidateRefCachePrefix('/api/v1/transaction-templates');
	}
}

export class OfflineCacheMissError extends Error {
	constructor(path: string) {
		super(`No cached data for ${path}`);
		this.name = 'OfflineCacheMissError';
	}
}

function notifyRefCacheUpdated(path: string): void {
	refCacheUpdate.set({ path, seq: Date.now() });
	refCacheTick.update((n) => n + 1);
}

function scheduleRevalidate<T>(path: string, fetcher: () => Promise<T>): void {
	if (inflightRevalidate.has(path)) return;
	const epoch = cacheEpoch;
	const job = (async () => {
		try {
			const value = await fetcher();
			if (epoch !== cacheEpoch) return;
			if (writeRefCache(path, value)) {
				notifyRefCacheUpdated(path);
			}
		} catch {
			// background refresh failed — keep stale on screen
		} finally {
			inflightRevalidate.delete(path);
		}
	})();
	inflightRevalidate.set(path, job);
}

export async function fetchWithRefCache<T>(path: string, fetcher: () => Promise<T>): Promise<T> {
	const cached = readRefCache<T>(path);
	if (cached !== null) {
		scheduleRevalidate(path, fetcher);
		return cached;
	}

	try {
		const value = await fetcher();
		writeRefCache(path, value);
		return value;
	} catch (err) {
		if (isStaleFetchError(err)) {
			const stale = readRefCache<T>(path);
			if (stale !== null) return stale;
		}
		throw err;
	}
}

export function clearRefCache(opts?: { preserveAuthMe?: boolean }): void {
	cacheEpoch++;
	const preserveAuthMe = opts?.preserveAuthMe === true;
	const prefix = `${REF_CACHE_VERSION}::`;
	if (typeof localStorage !== 'undefined') {
		try {
			const keys: string[] = [];
			for (let i = 0; i < localStorage.length; i++) {
				const key = localStorage.key(i);
				if (key?.startsWith(prefix)) keys.push(key);
			}
			for (const key of keys) {
				if (preserveAuthMe && isPreservedOfflineRefKey(key)) continue;
				localStorage.removeItem(key);
			}
		} catch {
			// ignore
		}
	}
	for (const key of [...memoryStore.keys()]) {
		if (!key.startsWith(prefix)) continue;
		if (preserveAuthMe && isPreservedOfflineRefKey(key)) continue;
		memoryStore.delete(key);
	}
	inflightRevalidate.clear();
	if (preserveAuthMe) {
		const meta = readRefCache<UIMeta>(UI_META_PATH);
		if (meta) seedAccountsFromUIMetaIfEmpty(meta);
	}
}

export function resetRefCacheForTests(): void {
	clearRefCache();
	memoryStore.clear();
	inflightRevalidate.clear();
	refCacheTick.set(0);
	refCacheUpdate.set(null);
	cacheUserId = '_anonymous';
	cacheEpoch = 0;
}

const UI_META_PATH = '/api/v1/ui/meta';

function pathFromStorageKey(key: string): string {
	const parts = key.split('::');
	return parts.length >= 3 ? parts.slice(2).join('::') : '';
}

export function isPreservedOfflineRefPath(path: string): boolean {
	const pathOnly = path.split('?')[0] ?? path;
	if (pathOnly === UI_META_PATH) return true;
	if (pathOnly === '/api/v1/categories') return true;
	if (pathOnly === '/api/v1/merchants') return true;
	if (pathOnly === '/api/v1/tags') return true;
	if (pathOnly === '/api/v1/banks') return true;
	if (pathOnly === '/api/v1/debtors') return true;
	if (pathOnly === '/api/v1/transaction-templates') return true;
	if (pathOnly === '/api/v1/accounts') return true;
	return /^\/api\/v1\/categories\/[^/]+\/subcategories$/.test(pathOnly);
}

function isPreservedOfflineRefKey(key: string): boolean {
	return isPreservedOfflineRefPath(pathFromStorageKey(key));
}

export function categoriesRefPath(type?: 'income' | 'expense'): string {
	const q = type ? `?type=${type}` : '';
	return `/api/v1/categories${q}`;
}

export function seedCategoriesFromUIMeta(meta: {
	expense_categories: unknown[];
	income_categories: unknown[];
}): void {
	writeRefCache(categoriesRefPath('expense'), meta.expense_categories);
	writeRefCache(categoriesRefPath('income'), meta.income_categories);
	writeRefCache(categoriesRefPath(), [...meta.expense_categories, ...meta.income_categories]);
}

export function readCategoriesFromUIMetaCache<T>(type?: 'income' | 'expense'): T[] | null {
	const meta = readRefCache<{
		expense_categories: T[];
		income_categories: T[];
	}>(UI_META_PATH);
	if (!meta) return null;
	if (type === 'expense') return meta.expense_categories;
	if (type === 'income') return meta.income_categories;
	return [...meta.expense_categories, ...meta.income_categories];
}

function isNonEmptyList<T>(value: T[] | null | undefined): value is T[] {
	return Array.isArray(value) && value.length > 0;
}

export function readCategoriesFromOfflineCache<T>(type?: 'income' | 'expense'): T[] | null {
	const fromList = readRefCache<T[]>(categoriesRefPath(type));
	if (isNonEmptyList(fromList)) return fromList;

	if (type) {
		const all = readRefCache<Array<{ type?: string }>>(categoriesRefPath());
		if (isNonEmptyList(all)) {
			const filtered = all.filter((row) => row.type === type) as T[];
			if (isNonEmptyList(filtered)) return filtered;
		}
	}

	return readCategoriesFromUIMetaCache<T>(type);
}

export function accountsRefPath(status?: 'active' | 'archived' | 'deleted'): string {
	const q = status ? `?status=${status}` : '';
	return `/api/v1/accounts${q}`;
}

function accountFromBalanceSummary(summary: AccountBalanceSummary): Account {
	return {
		id: summary.id,
		name: summary.name,
		type: summary.type,
		bank_id: null,
		bank_icon: summary.bank_icon ?? null,
		initial_balance: summary.balance,
		balance: summary.balance,
		balance_display: summary.balance_display,
		credit_limit: summary.credit_limit ?? null,
		credit_limit_display: summary.credit_limit_display ?? null,
		payment_account_id: null,
		auto_topup_enabled: summary.auto_topup_enabled ?? false,
		auto_topup_threshold: summary.auto_topup_threshold ?? null,
		auto_topup_threshold_display: summary.auto_topup_threshold_display ?? null,
		auto_topup_target: summary.auto_topup_target ?? null,
		auto_topup_target_display: summary.auto_topup_target_display ?? null,
		auto_topup_source_account_id: summary.auto_topup_source_account_id ?? null,
		status: 'active',
		is_primary: summary.is_primary,
		created_at: '',
		updated_at: ''
	};
}

function accountFromUIMetaRef(
	ref: UIMetaAccountRef,
	banks: readonly Bank[],
	summary?: AccountBalanceSummary
): Account {
	const bank = ref.bank_id ? banks.find((b) => b.id === ref.bank_id) : undefined;
	if (summary) {
		return {
			...accountFromBalanceSummary(summary),
			name: ref.name,
			type: ref.type,
			bank_id: ref.bank_id ?? null,
			bank_name: bank?.name ?? null,
			status: ref.status
		};
	}
	return {
		id: ref.id,
		name: ref.name,
		type: ref.type,
		bank_id: ref.bank_id ?? null,
		bank_name: bank?.name ?? null,
		bank_icon: bank?.icon_path ?? null,
		initial_balance: 0,
		balance: 0,
		balance_display: '0.00',
		credit_limit: null,
		credit_limit_display: null,
		payment_account_id: null,
		auto_topup_enabled: false,
		auto_topup_threshold: null,
		auto_topup_threshold_display: null,
		auto_topup_target: null,
		auto_topup_target_display: null,
		auto_topup_source_account_id: null,
		status: ref.status,
		is_primary: false,
		created_at: '',
		updated_at: ''
	};
}

export function seedAccountsFromUIMetaIfEmpty(meta: {
	accounts?: UIMetaAccountRef[];
	banks?: Bank[] | unknown[];
}): void {
	const refs = meta.accounts;
	if (!refs?.length) return;
	const banks = Array.isArray(meta.banks) ? (meta.banks as Bank[]) : [];
	const dash = readRefCache<Dashboard>('/api/v1/dashboard');
	const summaryById = new Map((dash?.accounts ?? []).map((row) => [row.id, row]));
	const mapped = refs.map((ref) => accountFromUIMetaRef(ref, banks, summaryById.get(ref.id)));
	const allCached = readRefCache<Account[]>(accountsRefPath());
	if (!isNonEmptyList(allCached)) {
		writeRefCache(accountsRefPath(), mapped);
	}
	for (const status of ['active', 'archived', 'deleted'] as const) {
		const statusCached = readRefCache<Account[]>(accountsRefPath(status));
		if (isNonEmptyList(statusCached)) continue;
		const rows = mapped.filter((row) => row.status === status);
		if (rows.length) writeRefCache(accountsRefPath(status), rows);
	}
}

export function seedDictionariesFromUIMeta(meta: {
	expense_categories: unknown[];
	income_categories: unknown[];
	accounts?: UIMetaAccountRef[];
	banks?: unknown[];
	merchants?: unknown[];
	tags?: unknown[];
	transaction_templates?: unknown[];
	debtors?: unknown[];
}): void {
	seedCategoriesFromUIMeta(meta);
	if (meta.banks !== undefined) writeRefCache('/api/v1/banks', meta.banks);
	if (meta.merchants !== undefined) writeRefCache('/api/v1/merchants', meta.merchants);
	if (meta.tags !== undefined) writeRefCache('/api/v1/tags', meta.tags);
	if (meta.transaction_templates !== undefined) {
		writeRefCache('/api/v1/transaction-templates', meta.transaction_templates);
	}
	if (meta.debtors !== undefined) writeRefCache('/api/v1/debtors', meta.debtors);
	seedAccountsFromUIMetaIfEmpty(meta);
}

export function readAccountsFromOfflineCache(
	status?: 'active' | 'archived' | 'deleted'
): Account[] | null {
	const path = accountsRefPath(status);
	const direct = readRefCache<Account[]>(path);
	if (isNonEmptyList(direct)) return direct;

	const all = readRefCache<Account[]>('/api/v1/accounts');
	if (isNonEmptyList(all)) {
		const filtered = status ? all.filter((row) => row.status === status) : all;
		if (isNonEmptyList(filtered)) return filtered;
	}

	if (!status || status === 'active') {
		const dash = readRefCache<Dashboard>('/api/v1/dashboard');
		if (isNonEmptyList(dash?.accounts)) {
			return dash.accounts.map((row) => accountFromBalanceSummary(row));
		}
	}

	const meta = readRefCache<UIMeta>(UI_META_PATH);
	if (!meta?.accounts?.length) return null;
	const dash = readRefCache<Dashboard>('/api/v1/dashboard');
	const summaryById = new Map((dash?.accounts ?? []).map((row) => [row.id, row]));
	let refs = meta.accounts;
	if (status) refs = refs.filter((row) => row.status === status);
	if (!refs.length) return null;
	return refs.map((ref) => accountFromUIMetaRef(ref, meta.banks, summaryById.get(ref.id)));
}
