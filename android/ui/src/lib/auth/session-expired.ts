import { writable } from 'svelte/store';

export const sessionExpiredTick = writable(0);

let notifyLocked = false;

export function isPublicAppRoute(pathname: string): boolean {
	return (
		pathname === '/setup' ||
		pathname === '/login' ||
		pathname.startsWith('/login/') ||
		pathname === '/register' ||
		pathname === '/server-setup'
	);
}

const API_401_EXEMPT = [
	'/api/v1/auth/login',
	'/api/v1/auth/register',
	'/api/v1/auth/logout',
	'/api/v1/setup',
	'/api/v1/setup/status',
	'/api/v1/version/check'
];

export function shouldRedirectApi401(apiPath: string): boolean {
	return !API_401_EXEMPT.some((prefix) => apiPath === prefix || apiPath.startsWith(`${prefix}/`));
}

/** Only treat 401 as session expiry when a Bearer token was actually sent. */
export function shouldNotifySessionExpired(apiPath: string, hasAuthToken: boolean): boolean {
	return hasAuthToken && shouldRedirectApi401(apiPath);
}

/**
 * Logout on 401 only when the active server URL matches the origin the token was issued for.
 * Prevents a dual-app clone (demo server) from invalidating the main app's session and vice versa.
 */
export function shouldLogoutOnApi401(
	apiPath: string,
	hasAuthToken: boolean,
	authServerOrigin: string,
	activeServerUrl: string,
	normalizeOrigin: (url: string) => string
): boolean {
	if (!shouldNotifySessionExpired(apiPath, hasAuthToken)) return false;
	if (!authServerOrigin) return true;
	if (!activeServerUrl) return false;
	try {
		return normalizeOrigin(authServerOrigin) === normalizeOrigin(activeServerUrl);
	} catch {
		return authServerOrigin === activeServerUrl;
	}
}

export function notifySessionExpired(): void {
	if (notifyLocked) return;
	notifyLocked = true;
	sessionExpiredTick.update((n) => n + 1);
	setTimeout(() => {
		notifyLocked = false;
	}, 500);
}

export function resetSessionExpiredSignal(): void {
	sessionExpiredTick.set(0);
}
