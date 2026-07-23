import { writable } from 'svelte/store';

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

/** Credit detail embeds full payment schedule — too large for sync localStorage SWR. */
const CREDIT_DETAIL_PATH = /^\/api\/v1\/credits\/[^/]+$/;

export function shouldPersistRefCache(path: string): boolean {
	const pathOnly = path.split('?')[0] ?? path;
	if (REF_CACHE_SKIP.has(pathOnly)) return false;
	if (!pathOnly.startsWith('/api/v1/')) return false;
	if (pathOnly.includes('/preview')) return false;
	if (pathOnly.includes('/import/jobs/')) return false;
	if (pathOnly.startsWith('/api/v1/export')) return false;
	if (pathOnly.startsWith('/api/v1/version')) return false;
	if (CREDIT_DETAIL_PATH.test(pathOnly)) return false;
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
			const prev = readRefCache<T>(path);
			writeRefCache(path, value);
			if (JSON.stringify(prev) !== JSON.stringify(value)) {
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

export function clearRefCache(): void {
	cacheEpoch++;
	const prefix = `${REF_CACHE_VERSION}::`;
	if (typeof localStorage !== 'undefined') {
		try {
			const keys: string[] = [];
			for (let i = 0; i < localStorage.length; i++) {
				const key = localStorage.key(i);
				if (key?.startsWith(prefix)) keys.push(key);
			}
			for (const key of keys) localStorage.removeItem(key);
		} catch {
			// ignore
		}
	}
	for (const key of [...memoryStore.keys()]) {
		if (key.startsWith(prefix)) memoryStore.delete(key);
	}
	inflightRevalidate.clear();
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
