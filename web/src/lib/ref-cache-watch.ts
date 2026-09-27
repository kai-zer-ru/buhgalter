/** True when a ref-cache revalidate notification applies to watched API path(s). */
export function refCachePathMatches(updated: string, watch: string | string[]): boolean {
	if (updated === '*') return true;
	const paths = Array.isArray(watch) ? watch : [watch];
	return paths.some((p) => {
		if (updated === p) return true;
		const base = p.split('?')[0] ?? p;
		const updatedBase = updated.split('?')[0] ?? updated;
		return updatedBase === base;
	});
}
