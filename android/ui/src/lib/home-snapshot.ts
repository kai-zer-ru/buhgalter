import { capturePageLoadError } from '$lib/page-load';
import { readRefCache } from '$lib/offline/ref-cache';
import { isServerUnavailableForUser } from '$lib/offline/server-connectivity';
import { assignIfChanged } from '$lib/state-utils';
import { toast } from '$lib/toast';

export const HOME_DASHBOARD_PATH = '/api/v1/dashboard';

/** Shown dashboard, or `GET /api/v1/dashboard` already in ref-cache. Enough for the summary. */
export function hasHomeDashboardSnapshot(shown: unknown): boolean {
	if (shown != null) return true;
	return readRefCache(HOME_DASHBOARD_PATH) !== null;
}

/** Foreground `loadAll` hides the summary, so it runs only without a snapshot. */
export function homeNeedsForegroundLoad(shown: unknown): boolean {
	return !hasHomeDashboardSnapshot(shown);
}

export type HomeDashboardScreen<T> = {
	dashBase: T | null;
	loadError: string | null;
	loading: boolean;
};

/**
 * Read the cached dashboard onto the screen and clear the gate.
 * Same result when `refCacheUpdate` is silent (warm suppress and the 8s grace).
 * Returns `screen` unchanged when the cache has no dashboard.
 */
export function applyCachedDashboardSnapshot<T>(
	screen: HomeDashboardScreen<T>
): HomeDashboardScreen<T> {
	const cached = readRefCache<T>(HOME_DASHBOARD_PATH);
	if (cached == null) return screen;
	const dashBase = assignIfChanged<T | null>(screen.dashBase, cached);
	if (dashBase === screen.dashBase && screen.loadError === null && !screen.loading) return screen;
	return { dashBase, loadError: null, loading: false };
}

/**
 * Gate copy only when there is no snapshot. Does not toast — the gate is the only
 * «Сервер недоступен». A failed refresh while a snapshot exists returns null.
 */
export function homeDashboardGateMessage(err: unknown, shown: unknown): string | null {
	if (hasHomeDashboardSnapshot(shown)) return null;
	return capturePageLoadError(err);
}

/** Home transaction lists toast only when the error is not `common.server_unavailable`. */
export function reportHomeListLoadFailure(err: unknown): void {
	if (isServerUnavailableForUser(err)) return;
	toast.fromError(err);
}
