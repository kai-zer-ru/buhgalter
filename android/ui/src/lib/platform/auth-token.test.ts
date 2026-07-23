import { beforeEach, describe, expect, it } from 'vitest';
import {
	clearAuthToken,
	getAuthToken,
	getAuthTokenKind,
	initAuthToken,
	resetAuthTokenForTests,
	setAuthToken
} from './auth-token';
import { resetSecureStoreForTests } from './secure-store';

describe('auth-token', () => {
	beforeEach(() => {
		resetAuthTokenForTests();
		resetSecureStoreForTests();
	});

	it('stores token in secure storage', async () => {
		await setAuthToken('secret-token');
		expect(getAuthToken()).toBe('secret-token');
		expect(getAuthTokenKind()).toBe('api_token');
		await clearAuthToken();
		expect(getAuthToken()).toBe('');
	});

	it('stores session token kind for password login', async () => {
		await setAuthToken('session-token', 'session');
		expect(getAuthToken()).toBe('session-token');
		expect(getAuthTokenKind()).toBe('session');
		await clearAuthToken();
		expect(getAuthTokenKind()).toBe('api_token');
	});

	it('migrates legacy localStorage token on init', async () => {
		if (typeof localStorage === 'undefined') return;
		localStorage.setItem('buhgalter.auth_token', 'legacy');
		resetAuthTokenForTests();
		resetSecureStoreForTests();
		await initAuthToken();
		expect(getAuthToken()).toBe('legacy');
		expect(getAuthTokenKind()).toBe('api_token');
		expect(localStorage.getItem('buhgalter.auth_token')).toBeNull();
	});
});
