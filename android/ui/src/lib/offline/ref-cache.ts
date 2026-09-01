import { writable } from 'svelte/store';
import type {
	Account,
	AccountBalanceSummary,
	Bank,
	Dashboard,
	UIMeta,
	UIMetaAccountRef
} from '$lib/api/client';
import { getServerProfile } from '$lib/platform/server-profile';
import { getServerUrl } from '$lib/platform/server-url';
import {
	isConnectionError,
	isServerOfflineMode,
	markServerOffline,
	markServerOnline
} from '$lib/offline/server-connectivity';
import { debugLogInfo, debugLogWarn } from '$lib/platform/debug-log';
import { isDashboardRefPath, isDashboardShape, stableEqual } from '$lib/state-utils';

const REF_CACHE_VERSION = 'buhgalter.ref_cache.v1';

/** GET paths that must never be served from stale cache. */
const REF_CACHE_SKIP = new Set([
	'/api/v1/health',
	// Bootstrap flag (registration_enabled) — must not serve a pre-mutation snapshot on /login.
	'/api/v1/setup/status'
]);

/** Kept across mutation clears so offline cold start can unlock (PIN/biometrics). */
export const AUTH_ME_PATH = '/api/v1/auth/me';

const UI_META_PATH = '/api/v1/ui/meta';

/**
 * Form catalogs not keyed by server URL — survives LAN/remote switch, stale IP,
 * `_no_server` race, and ref-cache key drift after days offline (like `last_user`).
 */
const STABLE_OFFLINE_CATALOG_KEY = 'buhgalter.offline_catalogs.v1';

export type StableOfflineCatalog = {
	savedAt: number;
	accounts?: UIMetaAccountRef[];
	expense_categories: unknown[];
	income_categories: unknown[];
	merchants?: unknown[];
	tags?: unknown[];
	debtors?: unknown[];
	banks?: unknown[];
	transaction_templates?: unknown[];
};

/** In-memory mirror when localStorage is unavailable (vitest) or as read-through cache. */
let stableCatalogMemory: StableOfflineCatalog | null = null;

const memoryStore = new Map<string, string>();
const inflightRevalidate = new Map<string, Promise<void>>();
const pendingDiskWrites = new Map<string, string>();
const lastRevalidatedAt = new Map<string, number>();
const revalidateTimers = new Map<string, ReturnType<typeof setTimeout>>();
let diskFlushTimer: ReturnType<typeof setTimeout> | null = null;

/** Wait before background GET so taps/scroll are not competing with revalidate. */
const REVALIDATE_DEFER_MS = import.meta.env.MODE === 'test' ? 0 : 3_000;
/** Per-path cooldown — avoid revalidate storms on every navigation. */
export const REVALIDATE_COOLDOWN_MS = 60_000;

function flushDiskWrites(): void {
	diskFlushTimer = null;
	if (typeof localStorage === 'undefined') return;
	for (const [key, value] of pendingDiskWrites) {
		try {
			localStorage.setItem(key, value);
		} catch {
			// quota
		}
	}
	pendingDiskWrites.clear();
}

/** Bumped when a background revalidate writes new data — pages reload softly. */
export const refCacheTick = writable(0);

/** Path-aware notification after SWR revalidate (preferred over refCacheTick). */
export const refCacheUpdate = writable<{ path: string; seq: number } | null>(null);

function storageKeyForServer(server: string, path: string): string {
	return `${REF_CACHE_VERSION}::${server || '_no_server'}::${path}`;
}

function storageKey(path: string): string {
	return storageKeyForServer(getServerUrl() || '_no_server', path);
}

function parseCachedValue<T>(raw: string | null): T | null {
	if (!raw) return null;
	try {
		return JSON.parse(raw) as T;
	} catch {
		return null;
	}
}

function readRefCacheForServer<T>(server: string, path: string): T | null {
	return parseCachedValue<T>(storageGet(storageKeyForServer(server || '_no_server', path)));
}

function pathFromStorageKey(key: string): string {
	const parts = key.split('::');
	return parts.length >= 3 ? parts.slice(2).join('::') : '';
}

/**
 * Dictionary paths never wiped on mutation clears (`preserveAuthMe`).
 * They stay on device and are only overwritten by fresh GET / seed helpers.
 * Full clear (logout / disconnect server) still removes everything.
 */
export function isPreservedOfflineRefPath(path: string): boolean {
	const pathOnly = path.split('?')[0] ?? path;
	if (pathOnly === AUTH_ME_PATH) return true;
	if (pathOnly === UI_META_PATH) return true;
	if (pathOnly === '/api/v1/categories') return true;
	if (pathOnly === '/api/v1/merchants') return true;
	if (pathOnly === '/api/v1/tags') return true;
	if (pathOnly === '/api/v1/banks') return true;
	if (pathOnly === '/api/v1/debtors') return true;
	if (pathOnly === '/api/v1/transaction-templates') return true;
	// Form catalogs: wiping this on write left empty account/category selects offline.
	if (pathOnly === '/api/v1/accounts') return true;
	return /^\/api\/v1\/categories\/[^/]+\/subcategories$/.test(pathOnly);
}

function isPreservedOfflineRefKey(key: string): boolean {
	return isPreservedOfflineRefPath(pathFromStorageKey(key));
}

function storageGet(key: string): string | null {
	const mem = memoryStore.get(key);
	if (mem !== undefined) return mem;
	if (typeof localStorage !== 'undefined') {
		try {
			const raw = localStorage.getItem(key);
			if (raw !== null) memoryStore.set(key, raw);
			return raw;
		} catch {
			// private mode / quota
		}
	}
	return null;
}

function storageSet(key: string, value: string): void {
	memoryStore.set(key, value);
	pendingDiskWrites.set(key, value);
	if (diskFlushTimer === null) {
		diskFlushTimer = setTimeout(flushDiskWrites, 32);
	}
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
	if (pathOnly.includes('/preview')) return false;
	return true;
}

/** Network / server errors where a cached GET response is acceptable. */
export function isOfflineFetchError(err: unknown): boolean {
	if (err instanceof OfflineCacheMissError) return true;
	if (isConnectionError(err)) return true;
	if (err && typeof err === 'object' && 'status' in err) {
		const status = Number((err as { status: number }).status);
		return status === 408 || status === 502 || status === 503 || status === 504;
	}
	return false;
}

export function readRefCache<T>(path: string): T | null {
	const current = getServerUrl() || '_no_server';
	const direct = readRefCacheForServer<T>(current, path);
	if (direct !== null && !isEmptyListValue(direct)) return direct;

	const fromProfile = readRefCacheAnyConfiguredServer<T>(path);
	if (fromProfile !== null && !isEmptyListValue(fromProfile)) return fromProfile;

	const fromAny = readRefCacheFromAnyStoredOrigin<T>(path);
	if (fromAny !== null) return fromAny;

	return direct;
}

function isEmptyListValue(value: unknown): boolean {
	return Array.isArray(value) && value.length === 0;
}

/**
 * Scan every ref-cache origin in storage (old LAN IP, `_no_server`, etc.).
 * Prefers a non-empty list over an empty one.
 */
export function readRefCacheFromAnyStoredOrigin<T>(path: string): T | null {
	const prefix = `${REF_CACHE_VERSION}::`;
	const suffix = `::${path}`;
	let emptyList: T | null = null;

	const consider = (raw: string | null | undefined): T | null | undefined => {
		const parsed = parseCachedValue<T>(raw ?? null);
		if (parsed === null) return undefined;
		if (isEmptyListValue(parsed)) {
			if (emptyList === null) emptyList = parsed;
			return undefined;
		}
		return parsed;
	};

	for (const [key, raw] of memoryStore) {
		if (!key.startsWith(prefix) || !key.endsWith(suffix)) continue;
		const hit = consider(raw);
		if (hit !== undefined) return hit;
	}

	if (typeof localStorage !== 'undefined') {
		try {
			for (let i = 0; i < localStorage.length; i++) {
				const key = localStorage.key(i);
				if (!key?.startsWith(prefix) || !key.endsWith(suffix)) continue;
				const hit = consider(localStorage.getItem(key));
				if (hit !== undefined) return hit;
			}
		} catch {
			// ignore
		}
	}

	return emptyList;
}

/**
 * Read path from active URL, then LAN/remote from the server profile.
 * Cold start may resolve a different origin than the one that wrote the cache.
 */
export function readRefCacheAnyConfiguredServer<T>(path: string): T | null {
	const current = getServerUrl() || '_no_server';
	const profile = getServerProfile();
	const candidates = [profile.lanUrl, profile.remoteUrl].filter(
		(u): u is string => Boolean(u) && u !== current
	);
	let emptyList: T | null = null;
	for (const origin of candidates) {
		const cached = readRefCacheForServer<T>(origin, path);
		if (cached === null) continue;
		if (isEmptyListValue(cached)) {
			if (emptyList === null) emptyList = cached;
			continue;
		}
		return cached;
	}
	return emptyList;
}

export function readStableOfflineCatalog(): StableOfflineCatalog | null {
	if (stableCatalogMemory) return stableCatalogMemory;
	if (typeof localStorage === 'undefined') return null;
	try {
		const raw = localStorage.getItem(STABLE_OFFLINE_CATALOG_KEY);
		if (!raw) return null;
		const parsed = JSON.parse(raw) as StableOfflineCatalog;
		if (!parsed || typeof parsed !== 'object') return null;
		if (!Array.isArray(parsed.expense_categories) || !Array.isArray(parsed.income_categories)) {
			return null;
		}
		stableCatalogMemory = parsed;
		return parsed;
	} catch {
		return null;
	}
}

export function persistStableOfflineCatalog(meta: {
	accounts?: UIMetaAccountRef[];
	expense_categories: unknown[];
	income_categories: unknown[];
	merchants?: unknown[];
	tags?: unknown[];
	debtors?: unknown[];
	banks?: unknown[];
	transaction_templates?: unknown[];
}): void {
	const snapshot: StableOfflineCatalog = {
		savedAt: Date.now(),
		accounts: meta.accounts,
		expense_categories: meta.expense_categories,
		income_categories: meta.income_categories,
		merchants: meta.merchants,
		tags: meta.tags,
		debtors: meta.debtors,
		banks: meta.banks,
		transaction_templates: meta.transaction_templates
	};
	stableCatalogMemory = snapshot;
	if (typeof localStorage === 'undefined') return;
	try {
		localStorage.setItem(STABLE_OFFLINE_CATALOG_KEY, JSON.stringify(snapshot));
	} catch {
		// quota
	}
}

export function clearStableOfflineCatalog(): void {
	stableCatalogMemory = null;
	if (typeof localStorage !== 'undefined') {
		try {
			localStorage.removeItem(STABLE_OFFLINE_CATALOG_KEY);
		} catch {
			// ignore
		}
	}
}

/**
 * Cold start after days offline: re-seed form catalogs for the active server URL
 * from stable snapshot and any legacy ref-cache origin.
 */
export function reconcileOfflineCatalogsOnUnlock(): void {
	flushRefCacheDisk();

	const stable = readStableOfflineCatalog();
	if (stable) {
		seedDictionariesFromUIMeta(stable);
	}

	const legacyMeta = readRefCacheFromAnyStoredOrigin<UIMeta>(UI_META_PATH);
	if (legacyMeta) {
		writeRefCache(UI_META_PATH, legacyMeta);
		seedDictionariesFromUIMeta(legacyMeta);
	}
}

export function refCacheReady(path: string): boolean {
	return readRefCache(path) !== null;
}

/** True when any of the paths has a cached GET response. */
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
		if (isDashboardRefPath(path) && isDashboardShape(value)) {
			patchAccountListCachesFromDashboard(value as Dashboard);
		}
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

export class OfflineCacheMissError extends Error {
	constructor(path: string) {
		super(`No cached data for ${path}`);
		this.name = 'OfflineCacheMissError';
	}
}

/** Write cache and notify subscribers (mutation / optimistic update). */
export function publishRefCachePath<T>(path: string, value: T): void {
	if (!writeRefCache(path, value)) return;
	notifyRefCacheUpdated(path, { bumpTick: true });
}

let suppressNotifyDepth = 0;
/** Skip SWR revalidate while warmRefCache is writing many paths at once. */
let warmRefCacheActive = false;
let warmRefCacheGraceUntil = 0;

export function setWarmRefCacheActive(active: boolean): void {
	warmRefCacheActive = active;
	if (!active) {
		warmRefCacheGraceUntil = Date.now() + 8_000;
	}
}

function isWarmOrGracePeriod(): boolean {
	return warmRefCacheActive || Date.now() < warmRefCacheGraceUntil;
}

export function isWarmRefCacheActive(): boolean {
	return warmRefCacheActive;
}

/** Batch cache writes (warmRefCache) without per-path UI reload storms. */
export async function runWithSuppressedRefCacheNotifications<T>(fn: () => Promise<T>): Promise<T> {
	suppressNotifyDepth++;
	try {
		return await fn();
	} finally {
		suppressNotifyDepth--;
	}
}

function notifyRefCacheUpdated(path: string, opts?: { bumpTick?: boolean }): void {
	if (suppressNotifyDepth > 0 || isWarmOrGracePeriod()) {
		return;
	}
	refCacheUpdate.set({ path, seq: Date.now() });
	if (opts?.bumpTick) {
		refCacheTick.update((n) => n + 1);
	}
}

function scheduleRevalidate<T>(path: string, fetcher: () => Promise<T>): void {
	if (isWarmOrGracePeriod() || inflightRevalidate.has(path)) return;
	const last = lastRevalidatedAt.get(path) ?? 0;
	if (Date.now() - last < REVALIDATE_COOLDOWN_MS) return;

	const existing = revalidateTimers.get(path);
	if (existing !== undefined) clearTimeout(existing);

	const timer = setTimeout(() => {
		revalidateTimers.delete(path);
		if (inflightRevalidate.has(path) || isWarmOrGracePeriod()) return;

		const job = (async () => {
			try {
				const value = await fetcher();
				markServerOnline();
				const changed = writeRefCache(path, value);
				lastRevalidatedAt.set(path, Date.now());
				if (changed) {
					queueMicrotask(() => {
						debugLogInfo('cache', `SWR revalidated ${path}`);
						notifyRefCacheUpdated(path);
					});
				}
			} catch (err) {
				if (isOfflineFetchError(err)) markServerOffline();
			} finally {
				inflightRevalidate.delete(path);
			}
		})();
		inflightRevalidate.set(path, job);
	}, REVALIDATE_DEFER_MS);

	revalidateTimers.set(path, timer);
}

export async function fetchWithRefCache<T>(path: string, fetcher: () => Promise<T>): Promise<T> {
	if (isServerOfflineMode()) {
		const cached = readRefCache<T>(path);
		if (cached !== null) {
			debugLogInfo('cache', `Offline cache hit ${path}`);
			return cached;
		}
		debugLogWarn('cache', `Offline cache miss ${path}`);
		throw new OfflineCacheMissError(path);
	}

	const cached = readRefCache<T>(path);
	if (cached !== null) {
		debugLogInfo('cache', `SWR cache hit ${path}`);
		scheduleRevalidate(path, fetcher);
		return cached;
	}

	try {
		const value = await fetcher();
		markServerOnline();
		writeRefCache(path, value);
		return value;
	} catch (err) {
		if (isOfflineFetchError(err)) {
			markServerOffline();
			const stale = readRefCache<T>(path);
			if (stale !== null) return stale;
		}
		throw err;
	}
}

export function clearRefCache(opts?: { preserveAuthMe?: boolean }): void {
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
		const meta =
			readRefCache<UIMeta>(UI_META_PATH) ??
			readStableOfflineCatalog() ??
			readRefCacheFromAnyStoredOrigin<UIMeta>(UI_META_PATH);
		if (meta) seedAccountsFromUIMetaIfEmpty(meta);
	} else {
		clearStableOfflineCatalog();
	}
}

export function resetRefCacheForTests(): void {
	clearRefCache();
	clearStableOfflineCatalog();
	memoryStore.clear();
	inflightRevalidate.clear();
	pendingDiskWrites.clear();
	lastRevalidatedAt.clear();
	for (const t of revalidateTimers.values()) clearTimeout(t);
	revalidateTimers.clear();
	if (diskFlushTimer !== null) {
		clearTimeout(diskFlushTimer);
		diskFlushTimer = null;
	}
	suppressNotifyDepth = 0;
	warmRefCacheActive = false;
	warmRefCacheGraceUntil = 0;
	stableCatalogMemory = null;
	refCacheTick.set(0);
	refCacheUpdate.set(null);
}

/** Flush deferred ref-cache writes to localStorage (warm sync, tests). */
export function flushRefCacheDisk(): void {
	if (diskFlushTimer !== null) {
		clearTimeout(diskFlushTimer);
		diskFlushTimer = null;
	}
	flushDiskWrites();
}

/** @deprecated Use flushRefCacheDisk */
export function flushRefCacheDiskForTests(): void {
	flushRefCacheDisk();
}

export function categoriesRefPath(type?: 'income' | 'expense'): string {
	const q = type ? `?type=${type}` : '';
	return `/api/v1/categories${q}`;
}

export function subcategoriesRefPath(categoryId: string): string {
	return `/api/v1/categories/${categoryId}/subcategories`;
}

/** ui/meta and list GETs share the same rows — overwrite local dictionaries (never wipe on write). */
export function seedCategoriesFromUIMeta(meta: {
	expense_categories: unknown[];
	income_categories: unknown[];
}): void {
	writeRefCache(categoriesRefPath('expense'), meta.expense_categories);
	writeRefCache(categoriesRefPath('income'), meta.income_categories);
	writeRefCache(categoriesRefPath(), [...meta.expense_categories, ...meta.income_categories]);
}

/** Full dictionary refresh from ui/meta — update-in-place on device. */
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
	persistStableOfflineCatalog(meta);
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

/** Offline fallback: typed list → full list → ui/meta. Empty cached lists are ignored. */
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

	return readCategoriesFromUIMetaCache<T>(type) ?? readCategoriesFromStableCatalog<T>(type);
}

function readCategoriesFromStableCatalog<T>(type?: 'income' | 'expense'): T[] | null {
	const stable = readStableOfflineCatalog();
	if (!stable) return null;
	if (type === 'expense') return stable.expense_categories as T[];
	if (type === 'income') return stable.income_categories as T[];
	return [...stable.expense_categories, ...stable.income_categories] as T[];
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

/**
 * Fill empty `/accounts` list keys from ui/meta (never overwrite a full GET snapshot).
 * Called after mutation-clear and when seeding dictionaries so the expense form
 * still has account ids offline.
 */
export function seedAccountsFromUIMetaIfEmpty(meta: {
	accounts?: UIMetaAccountRef[];
	banks?: Bank[] | unknown[];
}): void {
	const refs = meta.accounts;
	if (!refs?.length) return;
	const banks = Array.isArray(meta.banks) ? (meta.banks as Bank[]) : [];
	const dash = readRefCache<Dashboard>('/api/v1/dashboard');
	const summaryById = new Map((dash?.accounts ?? []).map((row) => [row.id, row]));
	const mapped = enrichAccountsWithCachedBalances(
		refs.map((ref) => accountFromUIMetaRef(ref, banks, summaryById.get(ref.id)))
	);
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

/** Overlay cached dashboard balances onto account rows. */
export function enrichAccountsWithCachedBalances(
	accounts: Account[],
	dashboard?: Dashboard | null
): Account[] {
	const dash = dashboard ?? readRefCache<Dashboard>('/api/v1/dashboard');
	if (!dash?.accounts?.length) return accounts;
	const summaryById = new Map(dash.accounts.map((row) => [row.id, row]));
	return accounts.map((acc) => applyBalanceSummary(acc, summaryById.get(acc.id)));
}

/** @deprecated Use enrichAccountsWithCachedBalances */
export const mergeAccountsWithDashboard = enrichAccountsWithCachedBalances;

export function enrichAccountWithCachedBalances(
	account: Account,
	dashboard?: Dashboard | null
): Account {
	const dash = dashboard ?? readRefCache<Dashboard>('/api/v1/dashboard');
	const summary = dash?.accounts.find((row) => row.id === account.id);
	return summary ? applyBalanceSummary(account, summary) : account;
}

function applyBalanceSummary(account: Account, summary?: AccountBalanceSummary): Account {
	if (!summary) return account;
	return {
		...account,
		balance: summary.balance,
		balance_display: summary.balance_display,
		is_primary: summary.is_primary ?? account.is_primary,
		credit_limit: summary.credit_limit ?? account.credit_limit,
		credit_limit_display: summary.credit_limit_display ?? account.credit_limit_display,
		auto_topup_enabled: summary.auto_topup_enabled ?? account.auto_topup_enabled,
		auto_topup_threshold: summary.auto_topup_threshold ?? account.auto_topup_threshold,
		auto_topup_threshold_display:
			summary.auto_topup_threshold_display ?? account.auto_topup_threshold_display,
		auto_topup_target: summary.auto_topup_target ?? account.auto_topup_target,
		auto_topup_target_display:
			summary.auto_topup_target_display ?? account.auto_topup_target_display,
		auto_topup_source_account_id:
			summary.auto_topup_source_account_id ?? account.auto_topup_source_account_id
	};
}

/** Keep preserved `/accounts*` list caches in sync when dashboard is refreshed. */
export function patchAccountListCachesFromDashboard(dashboard: Dashboard): void {
	if (!dashboard.accounts?.length) return;
	const paths = [
		'/api/v1/accounts',
		'/api/v1/accounts?status=active',
		'/api/v1/accounts?status=archived',
		'/api/v1/accounts?status=deleted'
	] as const;
	for (const path of paths) {
		const rows = readRefCache<Account[]>(path);
		if (!isNonEmptyList(rows)) continue;
		writeRefCache(path, enrichAccountsWithCachedBalances(rows, dashboard));
	}
	for (const summary of dashboard.accounts) {
		const acc = readRefCache<Account>(`/api/v1/accounts/${summary.id}`);
		if (acc) {
			writeRefCache(`/api/v1/accounts/${summary.id}`, applyBalanceSummary(acc, summary));
		}
		writeRefCache(`/api/v1/accounts/${summary.id}/balance`, summary);
	}
}

/** Offline fallback when GET /accounts was cleared but ui/meta or dashboard remain. */
export function readAccountsFromOfflineCache(
	status?: 'active' | 'archived' | 'deleted'
): Account[] | null {
	const path = accountsRefPath(status);
	const direct = readRefCache<Account[]>(path);
	if (isNonEmptyList(direct)) return enrichAccountsWithCachedBalances(direct);

	const all = readRefCache<Account[]>('/api/v1/accounts');
	if (isNonEmptyList(all)) {
		const filtered = status ? all.filter((row) => row.status === status) : all;
		if (isNonEmptyList(filtered)) return enrichAccountsWithCachedBalances(filtered);
	}

	if (!status || status === 'active') {
		const dash = readRefCache<Dashboard>('/api/v1/dashboard');
		if (isNonEmptyList(dash?.accounts)) {
			return dash.accounts.map((row) => accountFromBalanceSummary(row));
		}
	}

	const meta = readRefCache<UIMeta>(UI_META_PATH);
	if (!meta?.accounts?.length) {
		const stable = readStableOfflineCatalog();
		if (stable?.accounts?.length) {
			const banks = Array.isArray(stable.banks) ? (stable.banks as Bank[]) : [];
			const dash = readRefCache<Dashboard>('/api/v1/dashboard');
			const summaryById = new Map((dash?.accounts ?? []).map((row) => [row.id, row]));
			let refs = stable.accounts;
			if (status) refs = refs.filter((row) => row.status === status);
			if (refs.length) {
				return enrichAccountsWithCachedBalances(
					refs.map((ref) => accountFromUIMetaRef(ref, banks, summaryById.get(ref.id)))
				);
			}
		}
		return null;
	}
	const dash = readRefCache<Dashboard>('/api/v1/dashboard');
	const summaryById = new Map((dash?.accounts ?? []).map((row) => [row.id, row]));
	let refs = meta.accounts;
	if (status) refs = refs.filter((row) => row.status === status);
	if (!refs.length) return null;
	return enrichAccountsWithCachedBalances(
		refs.map((ref) => accountFromUIMetaRef(ref, meta.banks, summaryById.get(ref.id)))
	);
}
