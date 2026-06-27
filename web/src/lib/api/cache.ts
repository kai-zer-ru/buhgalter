type CacheEntry = {
	value: unknown;
	expiresAt: number;
};

const cache = new Map<string, CacheEntry>();
const inflight = new Map<string, Promise<unknown>>();

const DEFAULT_TTL_MS = 5 * 60 * 1000;

export function invalidateApiCache(prefix?: string) {
	if (!prefix) {
		cache.clear();
		return;
	}
	for (const key of cache.keys()) {
		if (key.startsWith(prefix)) cache.delete(key);
	}
}

export async function cachedGet<T>(
	key: string,
	fetcher: () => Promise<T>,
	ttlMs = DEFAULT_TTL_MS
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
