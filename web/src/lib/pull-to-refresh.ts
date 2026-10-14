import { invalidateRefCachePrefix } from '$lib/ref-cache';

/** Viewport width at or below which web PTR is allowed (with or without coarse pointer). */
export const PTR_VIEWPORT_MAX_PX = 768;

export type PtrActiveInput = {
	/** True inside Capacitor native shell. */
	isCapacitor: boolean;
	/** `matchMedia('(pointer: coarse)').matches`. */
	coarsePointer: boolean;
	/** `window.innerWidth` (CSS px). */
	viewportWidth: number;
};

/**
 * When the PTR gesture should be armed.
 * Capacitor: always. Web: coarse pointer and/or viewport ≲768px (not desktop fine + wide).
 */
export function isPullToRefreshActive(input: PtrActiveInput): boolean {
	if (input.isCapacitor) return true;
	return input.coarsePointer || input.viewportWidth <= PTR_VIEWPORT_MAX_PX;
}

/** Normalize pathname (strip query/hash, trailing slash except root). */
export function normalizePtrPathname(pathname: string): string {
	const bare = (pathname.split('?')[0] ?? pathname).split('#')[0] ?? pathname;
	const trimmed = bare.replace(/\/+$/, '');
	return trimmed === '' ? '/' : trimmed;
}

/**
 * Routes from surfaces.md §PTR: `/`, `/transactions`, `/accounts`, `/accounts/[id]`,
 * `/budget`, `/credits`, `/debts`. Excludes create/nested action paths.
 */
export function isPullToRefreshRoute(pathname: string): boolean {
	const path = normalizePtrPathname(pathname);
	switch (path) {
		case '/':
		case '/transactions':
		case '/accounts':
		case '/budget':
		case '/credits':
		case '/debts':
			return true;
	}
	const accountId = path.match(/^\/accounts\/([^/]+)$/);
	if (accountId && accountId[1] !== 'new') return true;
	return false;
}

/**
 * API path prefixes to soft-invalidate for a PTR route.
 * Empty when the pathname is not a PTR surface.
 */
export function apiPathsForPtrRoute(pathname: string): string[] {
	const path = normalizePtrPathname(pathname);
	if (path === '/') {
		return [
			'/api/v1/dashboard',
			'/api/v1/transactions',
			'/api/v1/budgets/summary',
			'/api/v1/transaction-templates'
		];
	}
	if (path === '/transactions') {
		return ['/api/v1/transactions', '/api/v1/ui/meta'];
	}
	if (path === '/accounts') {
		return ['/api/v1/accounts', '/api/v1/banks', '/api/v1/dashboard'];
	}
	const accountId = path.match(/^\/accounts\/([^/]+)$/);
	if (accountId && accountId[1] !== 'new') {
		return [`/api/v1/accounts/${accountId[1]}`, '/api/v1/transactions', '/api/v1/dashboard'];
	}
	if (path === '/budget') {
		return ['/api/v1/budgets', '/api/v1/ui/meta'];
	}
	if (path === '/credits') {
		return ['/api/v1/credits', '/api/v1/banks'];
	}
	if (path === '/debts') {
		return ['/api/v1/debts', '/api/v1/debts/summary'];
	}
	return [];
}

/**
 * Soft revalidate for the current route: path-aware ref-cache bust only
 * (no Sync All / warm / outbox; UI refresh is the caller's silent load).
 */
export function revalidateCurrentPath(pathname: string): void {
	const paths = apiPathsForPtrRoute(pathname);
	for (const p of paths) {
		invalidateRefCachePrefix((p.split('?')[0] ?? p).trim());
	}
}

/** True when a modal / drawer overlay should suppress PTR. */
export function isPtrBlockedByModal(root: ParentNode | null | undefined = undefined): boolean {
	const doc = root ?? (typeof document !== 'undefined' ? document : null);
	if (!doc) return false;
	if (doc.querySelector('[role="dialog"][aria-modal="true"]')) return true;
	if (doc.querySelector('.android-shell-layout.drawer-open')) return true;
	return false;
}

/**
 * Svelte 5 `ontouchmove` is passive, so `preventDefault` is a no-op.
 * Attach a non-passive listener so PTR can cancel scroll while pulling at top.
 * Returns a disposer for action destroy / onMount cleanup.
 */
export function attachNonPassiveTouchMove(
	node: HTMLElement,
	handler: (event: TouchEvent) => void
): () => void {
	node.addEventListener('touchmove', handler, { passive: false });
	return () => node.removeEventListener('touchmove', handler);
}
