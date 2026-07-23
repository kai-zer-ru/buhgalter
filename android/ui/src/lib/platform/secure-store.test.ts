import { beforeEach, describe, expect, it, vi } from 'vitest';
import { resetSecureStoreForTests, secureGet, secureRemove, secureSet } from './secure-store';

vi.mock('$lib/platform/native', () => ({
	isNativeApp: () => false
}));

describe('secure-store (web / e2e)', () => {
	beforeEach(() => {
		resetSecureStoreForTests();
	});

	it('persists across get/set without Capacitor', async () => {
		await secureSet('token', 'abc');
		expect(await secureGet('token')).toBe('abc');
		await secureRemove('token');
		expect(await secureGet('token')).toBeNull();
	});

	it('reads values from localStorage prefix after memory miss', async () => {
		if (typeof localStorage === 'undefined') return;
		localStorage.setItem('buhgalter.secure.token', 'from-ls');
		expect(await secureGet('token')).toBe('from-ls');
	});
});
