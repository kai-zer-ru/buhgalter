import { isNativeApp } from '$lib/platform/native';

/** Normalize user input to origin without trailing slash. */
export function normalizeServerUrl(raw: string): string {
	let s = raw.trim();
	if (!s) return '';
	if (!/^https?:\/\//i.test(s)) {
		s = `https://${s}`;
	}
	const url = new URL(s);
	return url.origin;
}

function isLoopbackHostname(hostname: string): boolean {
	return hostname === 'localhost' || hostname === '127.0.0.1' || hostname === '[::1]';
}

/**
 * Reject loopback on native (phone cannot reach its own 127.0.0.1 as the self-hosted server).
 * Allow loopback in browser / Playwright e2e so SPA on :9877 can talk to API on :9878.
 */
export function isUsableServerOrigin(origin: string): boolean {
	if (!origin) return false;
	try {
		const { hostname } = new URL(origin);
		if (isLoopbackHostname(hostname)) {
			return !isNativeApp();
		}
		return true;
	} catch {
		return false;
	}
}
