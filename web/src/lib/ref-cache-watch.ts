/** Path-aware matching for ref-cache soft-reload notifications. */

/** True when a single updated path applies to watched API path(s). */
export function refCachePathMatches(updated: string, watch: string | string[]): boolean {
	if (updated === '*') return true;
	const paths = Array.isArray(watch) ? watch : [watch];
	const updatedBase = (updated.split('?')[0] ?? updated).replace(/\/$/, '');
	return paths.some((p) => {
		if (updated === p) return true;
		const base = (p.split('?')[0] ?? p).replace(/\/$/, '');
		if (updatedBase === base) return true;
		// hint /accounts → open /accounts/{id}; hint /accounts/{id} → list /accounts
		if (base.startsWith(updatedBase + '/')) return true;
		if (updatedBase.startsWith(base + '/')) return true;
		return false;
	});
}

/** True when any path in a realtime update touches the watched path(s). */
export function refCacheUpdateMatches(
	update: { path: string; paths?: string[] },
	watch: string | string[]
): boolean {
	const candidates = update.paths && update.paths.length > 0 ? update.paths : [update.path];
	return candidates.some((p) => refCachePathMatches(p, watch));
}
