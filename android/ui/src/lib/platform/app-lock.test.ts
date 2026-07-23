import { describe, expect, it, beforeEach, vi } from 'vitest';
import {
	clearAppLock,
	enableAppLock,
	getAppLockConfig,
	isValidPin,
	isWeakPin,
	lockSession,
	noteAppBackground,
	noteAppForeground,
	resetAppLockForTests,
	retryBlockedMs,
	setAppLockConfigForTests,
	setBackgroundLockMs,
	setPin,
	shouldShowLockScreen,
	unlockSession,
	validateNewPin,
	verifyPin
} from './app-lock';
import { resetSecureStoreForTests } from './secure-store';

beforeEach(() => {
	resetSecureStoreForTests();
	resetAppLockForTests();
	vi.useRealTimers();
});

describe('isValidPin', () => {
	it('accepts exactly 4 digits', () => {
		expect(isValidPin('2580')).toBe(true);
		expect(isValidPin('123')).toBe(false);
		expect(isValidPin('12345')).toBe(false);
		expect(isValidPin('12a4')).toBe(false);
	});
});

describe('isWeakPin', () => {
	it('rejects repeated digits', () => {
		expect(isWeakPin('1111')).toBe(true);
		expect(isWeakPin('0000')).toBe(true);
	});

	it('rejects ascending and descending sequences', () => {
		expect(isWeakPin('1234')).toBe(true);
		expect(isWeakPin('4321')).toBe(true);
		expect(isWeakPin('7890')).toBe(true);
		expect(isWeakPin('0987')).toBe(true);
	});

	it('rejects common trivial PINs', () => {
		expect(isWeakPin('1212')).toBe(true);
		expect(isWeakPin('6969')).toBe(true);
	});

	it('allows non-trivial PINs', () => {
		expect(isWeakPin('2580')).toBe(false);
		expect(isWeakPin('5927')).toBe(false);
	});
});

describe('validateNewPin', () => {
	it('classifies format, weak, and ok', () => {
		expect(validateNewPin('12')).toBe('format');
		expect(validateNewPin('1234')).toBe('weak');
		expect(validateNewPin('2580')).toBe('ok');
	});
});

describe('setPin', () => {
	it('rejects weak PIN', async () => {
		await expect(setPin('1234')).rejects.toThrow('WEAK_PIN');
	});
});

describe('shouldShowLockScreen', () => {
	it('is true when enabled and session locked', () => {
		setAppLockConfigForTests({ enabled: true, biometricEnabled: false });
		lockSession();
		expect(shouldShowLockScreen()).toBe(true);
	});

	it('is false when unlocked', () => {
		setAppLockConfigForTests({ enabled: true, biometricEnabled: false });
		unlockSession();
		expect(shouldShowLockScreen()).toBe(false);
	});
});

describe('enableAppLock', () => {
	it('stores enabled config and requires unlock', async () => {
		await enableAppLock('2580');
		expect(getAppLockConfig().enabled).toBe(true);
		expect(shouldShowLockScreen()).toBe(true);

		const bad = await verifyPin('0000');
		expect(bad.ok).toBe(false);
		expect(retryBlockedMs()).toBe(0);

		const good = await verifyPin('2580');
		expect(good.ok).toBe(true);
		expect(shouldShowLockScreen()).toBe(false);
	});
});

describe('clearAppLock', () => {
	it('removes PIN and biometric without verification', async () => {
		await enableAppLock('2580');
		await clearAppLock();
		expect(getAppLockConfig()).toEqual({
			enabled: false,
			biometricEnabled: false,
			backgroundLockMs: 60_000
		});
		expect(shouldShowLockScreen()).toBe(false);
	});
});

describe('setBackgroundLockMs', () => {
	it('persists timeout', async () => {
		await enableAppLock('2580');
		await setBackgroundLockMs(300_000);
		expect(getAppLockConfig().backgroundLockMs).toBe(300_000);
	});
});

describe('background lock', () => {
	it('locks after configured idle in background', () => {
		vi.useFakeTimers();
		setAppLockConfigForTests({
			enabled: true,
			backgroundLockMs: 30_000
		});
		unlockSession();
		noteAppBackground();
		vi.advanceTimersByTime(29_000);
		noteAppForeground();
		expect(shouldShowLockScreen()).toBe(false);

		noteAppBackground();
		vi.advanceTimersByTime(30_000);
		noteAppForeground();
		expect(shouldShowLockScreen()).toBe(true);
	});
});
