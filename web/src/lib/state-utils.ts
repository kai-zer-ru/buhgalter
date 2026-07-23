/** Skip $state assignment when serialized value is unchanged — fewer reactive runs on background refresh. */
export function assignIfChanged<T>(prev: T, next: T): T {
	if (prev === next) return prev;
	if (JSON.stringify(prev) === JSON.stringify(next)) return prev;
	return next;
}

/** IDs present in next but not in prev (for row enter animations). */
export function diffNewIds<T extends { id: string }>(prev: T[], next: T[]): Set<string> {
	const prevIds = new Set(prev.map((item) => item.id));
	const added = new Set<string>();
	for (const item of next) {
		if (!prevIds.has(item.id)) added.add(item.id);
	}
	return added;
}
