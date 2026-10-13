import { isPublicAppRoute } from './session-expired';

/** Query param on `/login` with the path to open after successful auth. */
export const LOGIN_NEXT_PARAM = 'next';

/**
 * Safe in-app path for post-login navigation.
 * Rejects open redirects (protocol-relative, absolute URLs) and public auth routes.
 */
export function parsePostLoginPath(raw: string | null | undefined, fallback = '/'): string {
	if (!raw) return fallback;
	const path = raw.trim();
	if (!path.startsWith('/') || path.startsWith('//')) return fallback;
	if (path.includes('://') || path.includes('\\')) return fallback;
	const pathname = path.split(/[?#]/, 1)[0] ?? path;
	if (!pathname || isPublicAppRoute(pathname)) return fallback;
	return path;
}

/** Build `/login` or `/login?next=…` from the URL the user tried to open. */
export function loginHrefFor(pathname: string, search = ''): string {
	if (isPublicAppRoute(pathname)) {
		return pathname === '/login' ? `/login${search}` : '/login';
	}
	const next = parsePostLoginPath(`${pathname}${search}`, '');
	if (!next || next === '/') return '/login';
	return `/login?${LOGIN_NEXT_PARAM}=${encodeURIComponent(next)}`;
}
