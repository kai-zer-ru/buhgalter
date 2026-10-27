import type { CatalogScope } from '$lib/api/client';

const KEY = 'buhgalter.catalogScope';

export function readStoredCatalogScope(): CatalogScope {
	try {
		const raw = localStorage.getItem(KEY);
		if (raw === 'personal' || raw === 'household' || raw === 'all') return raw;
	} catch {
		/* ignore */
	}
	return 'all';
}

export function writeStoredCatalogScope(scope: CatalogScope) {
	try {
		localStorage.setItem(KEY, scope);
	} catch {
		/* ignore quota / private mode */
	}
}

export function catalogScopeQuery(scope?: CatalogScope): string {
	if (!scope || scope === 'all') return '';
	return scope;
}

/** Ref-cache / GET path for the home dashboard at this catalog tab. */
export function dashboardRefPath(scope?: CatalogScope): string {
	const q = catalogScopeQuery(scope);
	return q ? `/api/v1/dashboard?scope=${encodeURIComponent(q)}` : '/api/v1/dashboard';
}
