import { describe, expect, it } from 'vitest';
import { isPublicAppRoute, shouldNotifySessionExpired } from './session-expired';

describe('isPublicAppRoute', () => {
	it('treats login chooser and method screens as public', () => {
		expect(isPublicAppRoute('/login')).toBe(true);
		expect(isPublicAppRoute('/login/password')).toBe(true);
		expect(isPublicAppRoute('/login/token')).toBe(true);
		expect(isPublicAppRoute('/server-setup')).toBe(true);
	});

	it('does not treat app routes as public', () => {
		expect(isPublicAppRoute('/')).toBe(false);
		expect(isPublicAppRoute('/settings')).toBe(false);
		expect(isPublicAppRoute('/loginx')).toBe(false);
	});
});

describe('shouldNotifySessionExpired', () => {
	it('ignores 401 when no auth token was sent', () => {
		expect(shouldNotifySessionExpired('/api/v1/dashboard', false)).toBe(false);
		expect(shouldNotifySessionExpired('/api/v1/accounts', false)).toBe(false);
	});

	it('notifies on protected 401 when a token was sent', () => {
		expect(shouldNotifySessionExpired('/api/v1/dashboard', true)).toBe(true);
		expect(shouldNotifySessionExpired('/api/v1/accounts', true)).toBe(true);
	});

	it('keeps exempt paths quiet even with a token', () => {
		expect(shouldNotifySessionExpired('/api/v1/version/check', true)).toBe(false);
		expect(shouldNotifySessionExpired('/api/v1/auth/login', true)).toBe(false);
	});
});
