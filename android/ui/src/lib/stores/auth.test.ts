import { describe, expect, it, beforeEach, vi } from 'vitest';
import type { User } from '$lib/api/client';
import { ApiError, getMe } from '$lib/api/client';
import { writeRefCache, resetRefCacheForTests } from '$lib/offline/ref-cache';
import { markServerOffline, stopServerProbeLoopForTests } from '$lib/offline/server-connectivity';
import { setAuthToken, clearAuthToken } from '$lib/platform/auth-token';
import { setServerUrl, clearServerUrl } from '$lib/platform/server-url';
import {
	loadUser,
	restoreCachedUser,
	persistLastUser,
	clearLastUser,
	fallbackSessionUser,
	user,
	clearSessionHint
} from '$lib/stores/auth';
import { get } from 'svelte/store';

vi.mock('$lib/api/client', async (importOriginal) => {
	const actual = await importOriginal<typeof import('$lib/api/client')>();
	return {
		...actual,
		getMe: vi.fn()
	};
});

const me: User = {
	id: 'u1',
	login: 'test',
	display_name: 'Test',
	is_admin: false,
	status: 'active',
	language: 'ru',
	currency: 'RUB',
	timezone: 'Europe/Moscow',
	theme: 'system'
};

beforeEach(() => {
	clearServerUrl();
	clearAuthToken();
	clearSessionHint();
	clearLastUser();
	resetRefCacheForTests();
	stopServerProbeLoopForTests();
	user.set(null);
	setServerUrl('http://192.168.1.10:8765');
	setAuthToken('token');
	vi.mocked(getMe).mockReset();
});

describe('restoreCachedUser', () => {
	it('returns cached profile when token exists', () => {
		writeRefCache('/api/v1/auth/me', me);
		expect(restoreCachedUser()).toEqual(me);
	});

	it('returns null without token', () => {
		writeRefCache('/api/v1/auth/me', me);
		clearAuthToken();
		expect(restoreCachedUser()).toBeNull();
	});

	it('prefers stable last_user over wiped ref-cache', () => {
		persistLastUser(me);
		resetRefCacheForTests();
		expect(restoreCachedUser()).toEqual(me);
	});

	it('finds profile cached under the other configured origin', async () => {
		const { setServerProfile } = await import('$lib/platform/server-profile');
		setServerProfile({
			lanUrl: 'http://192.168.1.10:8765',
			remoteUrl: 'https://example.com',
			homeSsids: [],
			lanFallbackRemote: true,
			trustedOrigins: []
		});
		// Active URL is LAN; write under remote key via temporary switch.
		const { setActiveServerUrlForProbe } = await import('$lib/platform/server-url');
		setActiveServerUrlForProbe('https://example.com', 'remote');
		writeRefCache('/api/v1/auth/me', me);
		setActiveServerUrlForProbe('http://192.168.1.10:8765', 'lan');

		expect(restoreCachedUser()).toEqual(me);
	});
});

describe('fallbackSessionUser', () => {
	it('returns a minimal unlockable profile', () => {
		const stub = fallbackSessionUser();
		expect(stub.id).toBe('local-session');
		expect(stub.language).toBe('ru');
	});
});

describe('loadUser offline bootstrap', () => {
	it('restores cached user when server is unreachable', async () => {
		writeRefCache('/api/v1/auth/me', me);
		vi.mocked(getMe).mockRejectedValue(new TypeError('Failed to fetch'));

		const result = await loadUser();

		expect(result).toBe('ok');
		expect(get(user)).toEqual(me);
	});

	it('uses cache immediately when already in offline mode', async () => {
		writeRefCache('/api/v1/auth/me', me);
		markServerOffline();

		const result = await loadUser();

		expect(result).toBe('ok');
		expect(getMe).not.toHaveBeenCalled();
		expect(get(user)).toEqual(me);
	});

	it('uses last_user when offline and ref-cache empty', async () => {
		persistLastUser(me);
		resetRefCacheForTests();
		markServerOffline();

		const result = await loadUser();

		expect(result).toBe('ok');
		expect(getMe).not.toHaveBeenCalled();
		expect(get(user)).toEqual(me);
	});

	it('returns network when offline with no cache', async () => {
		markServerOffline();

		const result = await loadUser();

		expect(result).toBe('network');
		expect(getMe).not.toHaveBeenCalled();
		expect(get(user)).toBeNull();
	});

	it('returns unauthorized on 401 without cache fallback', async () => {
		vi.mocked(getMe).mockRejectedValue(new ApiError('UNAUTHORIZED', 'bad token', 401));

		const result = await loadUser();

		expect(result).toBe('unauthorized');
		expect(get(user)).toBeNull();
	});
});
