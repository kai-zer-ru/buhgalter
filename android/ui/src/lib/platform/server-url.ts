import {
	activeServerMode,
	clearServerProfile,
	getServerProfile,
	hasConfiguredServerProfile,
	resetServerProfileForTests,
	resolveActiveServerUrl,
	setServerProfile,
	type ActiveServerMode
} from '$lib/platform/server-profile';
import { isUsableServerOrigin, normalizeServerUrl } from '$lib/platform/server-origin';
import { getCurrentWifiSsid } from '$lib/platform/wifi-subnet';
import { syncTrustedOriginsToNative } from '$lib/platform/ssl-trust';
import { get } from 'svelte/store';

export { normalizeServerUrl, isUsableServerOrigin };

const LEGACY_URL_KEY = 'buhgalter.server_url';

let cachedActiveUrl = '';
let lastKnownSsid: string | null = null;

function applyResolved(url: string, mode: 'lan' | 'remote') {
	cachedActiveUrl = isUsableServerOrigin(url) ? url : '';
	activeServerMode.set(mode);
}

function syncResolveActiveUrl(): string {
	const resolved = resolveActiveServerUrl(lastKnownSsid, getServerProfile());
	applyResolved(resolved.url, resolved.mode);
	return cachedActiveUrl;
}

/** Re-read SSID and pick LAN vs remote URL. Call on startup and network changes. */
export async function refreshActiveServerUrl(): Promise<{
	url: string;
	ssid: string | null;
	mode: ActiveServerMode;
}> {
	const result = await getCurrentWifiSsid();
	if (result.ssid !== null) {
		lastKnownSsid = result.ssid;
	}
	const resolved = resolveActiveServerUrl(lastKnownSsid, getServerProfile());
	applyResolved(resolved.url, resolved.mode);
	void syncTrustedOriginsToNative(getServerProfile().trustedOrigins);
	return { url: cachedActiveUrl, ssid: lastKnownSsid, mode: resolved.mode };
}

/** Switch active URL after LAN probe failure (home SSID + lanFallbackRemote). */
export function setActiveServerUrlForProbe(url: string, mode: ActiveServerMode): void {
	applyResolved(url, mode);
}

export function getServerUrl(): string {
	if (cachedActiveUrl && isUsableServerOrigin(cachedActiveUrl)) return cachedActiveUrl;
	return syncResolveActiveUrl();
}

export function hasServerUrl(): boolean {
	return hasConfiguredServerProfile();
}

/** Sets LAN URL in profile (legacy helper). */
export function setServerUrl(raw: string) {
	const profile = setServerProfile({ lanUrl: raw });
	const resolved = resolveActiveServerUrl(lastKnownSsid, profile);
	applyResolved(resolved.url, resolved.mode);
}

export function clearServerUrl() {
	clearServerProfile();
	cachedActiveUrl = '';
	lastKnownSsid = null;
	activeServerMode.set(null);
}

export function getApiBase(): string {
	return getServerUrl();
}

export function resetServerUrlForTests(): void {
	resetServerProfileForTests();
	cachedActiveUrl = '';
	lastKnownSsid = null;
	if (typeof localStorage !== 'undefined') {
		try {
			localStorage.removeItem(LEGACY_URL_KEY);
		} catch {
			// ignore
		}
	}
}

export function getActiveServerModeForTests() {
	return get(activeServerMode);
}

export function setLastKnownSsidForTests(ssid: string | null) {
	lastKnownSsid = ssid;
}
