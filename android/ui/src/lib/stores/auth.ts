import { writable } from 'svelte/store';
import type { User } from '$lib/api/client';
import { ApiError, getMe, isTransientHttpError, logout as apiLogout } from '$lib/api/client';
import { resetSessionExpiredSignal } from '$lib/auth/session-expired';
import {
	AUTH_ME_PATH,
	clearRefCache,
	readRefCache,
	readRefCacheAnyConfiguredServer,
	writeRefCache
} from '$lib/offline/ref-cache';
import { isServerOfflineMode, markServerOffline } from '$lib/offline/server-connectivity';
import { clearAuthToken, getAuthToken, getAuthTokenKind } from '$lib/platform/auth-token';
import { clearAppLock } from '$lib/platform/app-lock';

export const user = writable<User | null>(null);
export const authReady = writable(false);

export type LoadUserResult = 'ok' | 'unauthorized' | 'network';

const SESSION_HINT_KEY = 'buhgalter.session_hint';
/** Stable across LAN/remote URL switches — used for offline PIN unlock. */
const LAST_USER_KEY = 'buhgalter.last_user.v1';

/** In-memory mirror (vitest/node has no localStorage; also survives same-session wipe). */
let lastUserMemory: User | null = null;

function sleep(ms: number) {
	return new Promise((resolve) => setTimeout(resolve, ms));
}

export function markSessionHint() {
	try {
		sessionStorage.setItem(SESSION_HINT_KEY, '1');
		resetSessionExpiredSignal();
	} catch {
		// ignore private mode
	}
}

export function clearSessionHint() {
	try {
		sessionStorage.removeItem(SESSION_HINT_KEY);
	} catch {
		// ignore
	}
}

export function hasRecentSession(): boolean {
	return Boolean(getAuthToken());
}

function hasSessionHint(): boolean {
	try {
		return sessionStorage.getItem(SESSION_HINT_KEY) === '1';
	} catch {
		return false;
	}
}

function isUserShape(value: unknown): value is User {
	if (!value || typeof value !== 'object') return false;
	const u = value as Record<string, unknown>;
	return typeof u.id === 'string' && typeof u.language === 'string' && typeof u.theme === 'string';
}

/** Persist profile outside URL-keyed ref-cache (survives origin switch + mutation clears). */
export function persistLastUser(profile: User): void {
	lastUserMemory = profile;
	try {
		if (typeof localStorage !== 'undefined') {
			localStorage.setItem(LAST_USER_KEY, JSON.stringify(profile));
		}
	} catch {
		// ignore quota / private mode
	}
	try {
		writeRefCache(AUTH_ME_PATH, profile);
	} catch {
		// ignore
	}
}

export function clearLastUser(): void {
	lastUserMemory = null;
	try {
		if (typeof localStorage !== 'undefined') {
			localStorage.removeItem(LAST_USER_KEY);
		}
	} catch {
		// ignore
	}
}

export function readLastUser(): User | null {
	if (lastUserMemory && isUserShape(lastUserMemory)) return lastUserMemory;
	if (typeof localStorage === 'undefined') return null;
	try {
		const raw = localStorage.getItem(LAST_USER_KEY);
		if (!raw) return null;
		const parsed: unknown = JSON.parse(raw);
		return isUserShape(parsed) ? parsed : null;
	} catch {
		return null;
	}
}

/**
 * Profile for offline unlock: stable last_user, then ref-cache (any configured origin).
 */
export function restoreCachedUser(): User | null {
	if (!getAuthToken()) return null;
	return (
		readLastUser() ??
		readRefCacheAnyConfiguredServer<User>(AUTH_ME_PATH) ??
		readRefCache<User>(AUTH_ME_PATH)
	);
}

/** Minimal profile so lock UI can show when token exists but profile was never persisted. */
export function fallbackSessionUser(): User {
	return {
		id: 'local-session',
		login: '',
		display_name: '',
		is_admin: false,
		status: 'active',
		language: 'ru',
		currency: 'RUB',
		timezone: 'Europe/Moscow',
		theme: 'system'
	};
}

const LOAD_RETRIES = 2;
const RETRY_BASE_MS = 300;

export async function loadUser(): Promise<LoadUserResult> {
	const cached = restoreCachedUser();
	if (isServerOfflineMode()) {
		if (cached) {
			user.set(cached);
			persistLastUser(cached);
			markSessionHint();
			return 'ok';
		}
		return 'network';
	}

	for (let attempt = 0; attempt < LOAD_RETRIES; attempt++) {
		try {
			const me = await getMe();
			user.set(me);
			persistLastUser(me);
			markSessionHint();
			return 'ok';
		} catch (err) {
			const retryable =
				!(err instanceof ApiError) ||
				isTransientHttpError(err.status) ||
				(err.status === 401 && hasSessionHint() && attempt < LOAD_RETRIES - 1);

			if (retryable && attempt < LOAD_RETRIES - 1) {
				await sleep(RETRY_BASE_MS * (attempt + 1));
				continue;
			}

			if (err instanceof ApiError && err.status === 401) {
				clearSessionHint();
				user.set(null);
				return 'unauthorized';
			}

			break;
		}
	}

	const fallback = restoreCachedUser();
	if (fallback) {
		user.set(fallback);
		persistLastUser(fallback);
		markServerOffline();
		markSessionHint();
		return 'ok';
	}

	return 'network';
}

export async function logout() {
	clearSessionHint();
	if (getAuthTokenKind() === 'session' && getAuthToken()) {
		try {
			await apiLogout();
		} catch {
			// offline / already invalid — still clear locally
		}
	}
	const { clearWidgetsOnLogout } = await import('$lib/widgets/publish');
	await clearWidgetsOnLogout();
	await clearAuthToken();
	await clearAppLock();
	clearRefCache();
	clearLastUser();
	user.set(null);
}
