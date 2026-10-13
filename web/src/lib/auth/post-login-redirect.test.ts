import { describe, expect, it } from 'vitest';
import { LOGIN_NEXT_PARAM, loginHrefFor, parsePostLoginPath } from './post-login-redirect';

describe('parsePostLoginPath', () => {
	it('accepts relative app paths with query', () => {
		expect(parsePostLoginPath('/budget')).toBe('/budget');
		expect(parsePostLoginPath('/budget?month=2026-11')).toBe('/budget?month=2026-11');
		expect(parsePostLoginPath('/subscriptions?edit=abc')).toBe('/subscriptions?edit=abc');
	});

	it('rejects open redirects and public routes', () => {
		expect(parsePostLoginPath('https://evil.example/')).toBe('/');
		expect(parsePostLoginPath('//evil.example/')).toBe('/');
		expect(parsePostLoginPath('/\\evil')).toBe('/');
		expect(parsePostLoginPath('/login')).toBe('/');
		expect(parsePostLoginPath('/register')).toBe('/');
		expect(parsePostLoginPath('/setup')).toBe('/');
		expect(parsePostLoginPath(null)).toBe('/');
		expect(parsePostLoginPath('', '/accounts')).toBe('/accounts');
	});
});

describe('loginHrefFor', () => {
	it('wraps protected path in next', () => {
		expect(loginHrefFor('/budget')).toBe(`/login?${LOGIN_NEXT_PARAM}=%2Fbudget`);
		expect(loginHrefFor('/budget', '?month=2026-11')).toBe(
			`/login?${LOGIN_NEXT_PARAM}=%2Fbudget%3Fmonth%3D2026-11`
		);
	});

	it('keeps existing login search and does not nest public routes', () => {
		expect(loginHrefFor('/login', `?${LOGIN_NEXT_PARAM}=%2Fbudget`)).toBe(
			`/login?${LOGIN_NEXT_PARAM}=%2Fbudget`
		);
		expect(loginHrefFor('/register')).toBe('/login');
		expect(loginHrefFor('/setup')).toBe('/login');
		expect(loginHrefFor('/')).toBe('/login');
	});
});
