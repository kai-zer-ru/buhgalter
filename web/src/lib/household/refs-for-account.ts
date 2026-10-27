export type NamedRef = { id: string; name: string; scope?: string };

export function catalogScopeOfAccount(
	account: { scope?: string } | null | undefined
): 'personal' | 'household' {
	return account?.scope === 'household' ? 'household' : 'personal';
}

export function refsForAccount<T extends { scope?: string }>(
	account: { scope?: string } | null | undefined,
	refs: readonly T[]
): T[] {
	const scope = catalogScopeOfAccount(account);
	return refs.filter((ref) => (ref.scope ?? 'personal') === scope);
}

export function matchRefByName<T extends NamedRef>(
	current: T | undefined,
	nextCatalog: readonly T[]
): T | undefined {
	if (!current) return undefined;
	const byId = nextCatalog.find((ref) => ref.id === current.id);
	if (byId) return byId;
	const name = current.name.trim().toLowerCase();
	if (!name) return undefined;
	return nextCatalog.find((ref) => ref.name.trim().toLowerCase() === name);
}

export function remapRefIds<T extends NamedRef>(
	current: readonly T[],
	nextCatalog: readonly T[]
): { next: T[]; dropped: boolean } {
	const next: T[] = [];
	let dropped = false;
	for (const item of current) {
		const matched = matchRefByName(item, nextCatalog);
		if (matched) next.push(matched);
		else dropped = true;
	}
	return { next, dropped };
}
