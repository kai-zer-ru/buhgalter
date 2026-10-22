/** True when a ref-cache revalidate notification applies to watched API path(s). */
export function refCachePathMatches(updated: string, watch: string | string[]): boolean {
	const paths = Array.isArray(watch) ? watch : [watch];
	return paths.some((p) => {
		if (updated === p) return true;
		const base = p.split('?')[0] ?? p;
		const updatedBase = updated.split('?')[0] ?? updated;
		return updatedBase === base;
	});
}
