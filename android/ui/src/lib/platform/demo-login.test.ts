import { beforeEach, describe, expect, it, vi } from 'vitest';
import { login, type User } from '$lib/api/client';
import {
	getAuthServerOrigin,
	getAuthToken,
	getAuthTokenKind,
	resetAuthTokenForTests
} from '$lib/platform/auth-token';
import { resetSecureStoreForTests } from '$lib/platform/secure-store';
import {
	getServerProfile,
	resetServerProfileForTests,
	setServerProfile,
	type ServerProfile
} from '$lib/platform/server-profile';
import { getServerUrl, hasServerUrl, resetServerUrlForTests } from '$lib/platform/server-url';
import {
	DEMO_LOGIN,
	DEMO_PASSWORD,
	DEMO_SERVER_ORIGIN,
	enterDemoMode,
	needsDemoReplaceConfirm
} from './demo-login';

vi.mock('$lib/api/client', async (importOriginal) => {
	const actual = await importOriginal<typeof import('$lib/api/client')>();
	return {
		...actual,
		login: vi.fn()
	};
});

const loginMock = vi.mocked(login);

const saved: ServerProfile = {
	lanUrl: 'http://192.168.1.10:8765',
	remoteUrl: 'https://home.example.com',
	homeSsids: ['HomeWiFi'],
	lanFallbackRemote: true,
	trustedOrigins: ['https://home.example.com']
};

const demoUser: User = {
	id: 'u-demo',
	login: 'demo',
	display_name: 'Demo',
	is_admin: false,
	status: 'active',
	language: 'ru',
	currency: 'RUB',
	timezone: 'Asia/Krasnoyarsk',
	theme: 'system'
};

describe('demo login', () => {
	beforeEach(() => {
		resetServerUrlForTests();
		resetServerProfileForTests();
		resetAuthTokenForTests();
		resetSecureStoreForTests();
		loginMock.mockReset();
	});

	it('uses the public stand origin and demo account', () => {
		expect(DEMO_SERVER_ORIGIN).toBe('https://buhgalter-demo.kai-zer.ru');
		expect(DEMO_LOGIN).toBe('demo');
		expect(DEMO_PASSWORD).toBe('demo_1_demo');
	});

	it('asks to replace a saved server that is not the demo origin', () => {
		expect(
			needsDemoReplaceConfirm({
				lanUrl: '',
				remoteUrl: '',
				homeSsids: [],
				lanFallbackRemote: false,
				trustedOrigins: []
			})
		).toBe(false);
		expect(
			needsDemoReplaceConfirm({
				lanUrl: DEMO_SERVER_ORIGIN,
				remoteUrl: '',
				homeSsids: ['HomeWiFi'],
				lanFallbackRemote: false,
				trustedOrigins: []
			})
		).toBe(false);
		expect(needsDemoReplaceConfirm(saved)).toBe(true);
	});

	it('signs in against the demo origin and keeps only that profile', async () => {
		setServerProfile(saved);
		loginMock.mockImplementation(async (name, password) => {
			expect(name).toBe(DEMO_LOGIN);
			expect(password).toBe(DEMO_PASSWORD);
			expect(getServerUrl()).toBe(DEMO_SERVER_ORIGIN);
			return { token: 'session-token', user: demoUser };
		});

		const signedIn = await enterDemoMode();

		expect(signedIn.login).toBe('demo');
		expect(getServerProfile()).toEqual({
			lanUrl: DEMO_SERVER_ORIGIN,
			remoteUrl: '',
			homeSsids: [],
			lanFallbackRemote: false,
			trustedOrigins: []
		});
		expect(getServerUrl()).toBe(DEMO_SERVER_ORIGIN);
		expect(getAuthToken()).toBe('session-token');
		expect(getAuthTokenKind()).toBe('session');
		expect(getAuthServerOrigin()).toBe(DEMO_SERVER_ORIGIN);
	});

	it('restores the previous profile when login fails', async () => {
		setServerProfile(saved);
		const before = getServerProfile();
		loginMock.mockImplementation(async () => {
			expect(getServerUrl()).toBe(DEMO_SERVER_ORIGIN);
			throw new Error('unreachable');
		});

		await expect(enterDemoMode()).rejects.toThrow('unreachable');

		expect(getServerProfile()).toEqual(before);
		expect(getServerUrl()).toBe(before.remoteUrl);
		expect(getAuthToken()).toBe('');
	});

	it('leaves an empty profile empty when login fails', async () => {
		expect(hasServerUrl()).toBe(false);
		loginMock.mockImplementation(async (name, password) => {
			expect(name).toBe(DEMO_LOGIN);
			expect(password).toBe(DEMO_PASSWORD);
			expect(getServerUrl()).toBe(DEMO_SERVER_ORIGIN);
			throw new Error('unreachable');
		});

		await expect(enterDemoMode()).rejects.toThrow('unreachable');

		expect(hasServerUrl()).toBe(false);
		expect(getServerProfile()).toEqual({
			lanUrl: '',
			remoteUrl: '',
			homeSsids: [],
			lanFallbackRemote: false,
			trustedOrigins: []
		});
		expect(getServerUrl()).toBe('');
	});
});
