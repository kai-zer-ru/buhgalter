import { describe, expect, it } from 'vitest';
import { isPublicAppRoute } from './session-expired';

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
