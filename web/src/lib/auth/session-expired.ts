import { writable } from 'svelte/store';

export const sessionExpiredTick = writable(0);

let notifyLocked = false;

export function isPublicAppRoute(pathname: string): boolean {
	return (
		pathname === '/setup' ||
		pathname === '/login' ||
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
