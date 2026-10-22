import { afterEach, describe, expect, it, vi } from 'vitest';

afterEach(() => {
	vi.restoreAllMocks();
	vi.resetModules();
	if (typeof globalThis.window !== 'undefined') {
		delete (globalThis.window as Window & { Capacitor?: unknown }).Capacitor;
	}
});

describe('server-origin', () => {
	it('normalizes to origin', async () => {
		const { normalizeServerUrl } = await import('./server-origin');
		expect(normalizeServerUrl('https://buh.example.com/path')).toBe('https://buh.example.com');
		expect(normalizeServerUrl('buh.example.com')).toBe('https://buh.example.com');
	});

	it('allows loopback in browser', async () => {
		const { isUsableServerOrigin } = await import('./server-origin');
		expect(isUsableServerOrigin('http://127.0.0.1:9878')).toBe(true);
		expect(isUsableServerOrigin('http://localhost:8765')).toBe(true);
	});

	it('rejects loopback on native Capacitor', async () => {
		vi.stubGlobal('window', {
			Capacitor: { isNativePlatform: () => true }
		});
		const { isUsableServerOrigin } = await import('./server-origin');
		expect(isUsableServerOrigin('http://127.0.0.1:9878')).toBe(false);
		expect(isUsableServerOrigin('https://buh.example.com')).toBe(true);
	});
});
