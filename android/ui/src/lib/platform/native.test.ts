import { afterEach, describe, expect, it, vi } from 'vitest';

afterEach(() => {
	vi.restoreAllMocks();
	vi.resetModules();
	if (typeof globalThis.window !== 'undefined') {
		delete (globalThis.window as Window & { Capacitor?: unknown }).Capacitor;
	}
});

describe('isNativeApp', () => {
	it('is false without Capacitor (browser / e2e / node tests)', async () => {
		const { isNativeApp, isMobileApp } = await import('./native');
		expect(isNativeApp()).toBe(false);
		expect(isMobileApp()).toBe(false);
	});

	it('follows Capacitor.isNativePlatform when present', async () => {
		vi.stubGlobal('window', {
			Capacitor: { isNativePlatform: () => true }
		});
		const { isNativeApp, isMobileApp } = await import('./native');
		expect(isNativeApp()).toBe(true);
		expect(isMobileApp()).toBe(true);
	});
});
