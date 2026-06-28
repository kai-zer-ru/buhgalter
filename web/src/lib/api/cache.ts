type CacheEntry = {
	value: unknown;
	expiresAt: number;
};

const cache = new Map<string, CacheEntry>();
const inflight = new Map<string, Promise<unknown>>();

/** Справочники, меняющиеся только с релизом (банки и т.п.). */
export const STATIC_REF_TTL_MS = 24 * 60 * 60 * 1000;

export function invalidateApiCache(prefix?: string) {
	if (!prefix) {
		cache.clear();
		inflight.clear();
		return;
	}
	for (const key of cache.keys()) {
		if (key.startsWith(prefix)) cache.delete(key);
	}
}

export function seedStaticRef<T>(key: string, value: T, ttlMs = STATIC_REF_TTL_MS) {
	cache.set(key, { value, expiresAt: Date.now() + ttlMs });
}

export async function cachedGet<T>(
	key: string,
	fetcher: () => Promise<T>,
	ttlMs = STATIC_REF_TTL_MS
): Promise<T> {
	const now = Date.now();
	const hit = cache.get(key);
	if (hit && hit.expiresAt > now) {
		return hit.value as T;
	}

	let pending = inflight.get(key);
	if (!pending) {
		pending = fetcher()
			.then((value) => {
				cache.set(key, { value, expiresAt: Date.now() + ttlMs });
				inflight.delete(key);
				return value;
			})
			.catch((err) => {
				inflight.delete(key);
				throw err;
			});
		inflight.set(key, pending);
	}
	return pending as Promise<T>;
}
