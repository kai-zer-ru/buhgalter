import { writable } from 'svelte/store';
import { getServerProfile } from '$lib/platform/server-profile';
import { getServerUrl } from '$lib/platform/server-url';
import {
	isConnectionError,
	isServerOfflineMode,
	markServerOffline,
	markServerOnline
} from '$lib/offline/server-connectivity';
import { debugLogInfo, debugLogWarn } from '$lib/platform/debug-log';

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

const memoryStore = new Map<string, string>();
const inflightRevalidate = new Map<string, Promise<void>>();

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
	return /^\/api\/v1\/categories\/[^/]+\/subcategories$/.test(pathOnly);
}

function isPreservedOfflineRefKey(key: string): boolean {
	return isPreservedOfflineRefPath(pathFromStorageKey(key));
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
	if (direct !== null) return direct;
	return readRefCacheAnyConfiguredServer<T>(path);
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
	for (const origin of candidates) {
		const cached = readRefCacheForServer<T>(origin, path);
		if (cached !== null) return cached;
	}
	return null;
}

export function refCacheReady(path: string): boolean {
	return readRefCache(path) !== null;
}

/** True when any of the paths has a cached GET response. */
export function refCacheReadyAny(paths: string[]): boolean {
	return paths.some(refCacheReady);
}

export function writeRefCache<T>(path: string, value: T): void {
	try {
		storageSet(storageKey(path), JSON.stringify(value));
	} catch {
		// ignore serialization / quota errors
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
	writeRefCache(path, value);
	notifyRefCacheUpdated(path);
}

function notifyRefCacheUpdated(path: string): void {
	refCacheUpdate.set({ path, seq: Date.now() });
	refCacheTick.update((n) => n + 1);
}

function scheduleRevalidate<T>(path: string, fetcher: () => Promise<T>): void {
	if (inflightRevalidate.has(path)) return;
	const job = (async () => {
		try {
			const value = await fetcher();
			markServerOnline();
			const prev = readRefCache<T>(path);
			writeRefCache(path, value);
			if (JSON.stringify(prev) !== JSON.stringify(value)) {
				debugLogInfo('cache', `SWR revalidated ${path}`);
				notifyRefCacheUpdated(path);
			}
		} catch (err) {
			if (isOfflineFetchError(err)) markServerOffline();
		} finally {
			inflightRevalidate.delete(path);
		}
	})();
	inflightRevalidate.set(path, job);
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
}

export function resetRefCacheForTests(): void {
	clearRefCache();
	memoryStore.clear();
	inflightRevalidate.clear();
	refCacheTick.set(0);
	refCacheUpdate.set(null);
}

export function categoriesRefPath(type?: 'income' | 'expense'): string {
	const q = type ? `?type=${type}` : '';
	return `/api/v1/categories${q}`;
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
