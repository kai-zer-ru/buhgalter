import { writable } from 'svelte/store';
import type { User } from '$lib/api/client';
import { ApiError, getMe, isTransientHttpError, logout as apiLogout } from '$lib/api/client';
import { resetSessionExpiredSignal } from '$lib/auth/session-expired';

export const user = writable<User | null>(null);
export const authReady = writable(false);

export type LoadUserResult = 'ok' | 'unauthorized' | 'network';

const SESSION_HINT_KEY = 'buhgalter.session_hint';

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
	return hasSessionHint();
}

function hasSessionHint(): boolean {
	try {
		return sessionStorage.getItem(SESSION_HINT_KEY) === '1';
	} catch {
		return false;
	}
}

const LOAD_RETRIES = 4;
const RETRY_BASE_MS = 350;

export async function loadUser(): Promise<LoadUserResult> {
	for (let attempt = 0; attempt < LOAD_RETRIES; attempt++) {
		try {
			const me = await getMe();
			user.set(me);
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

			return 'network';
		}
	}
	return 'network';
}

export async function logout() {
	try {
		await apiLogout();
	} catch {
		// ignore
	}
	clearSessionHint();
	user.set(null);
}
