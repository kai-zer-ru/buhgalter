import { writable } from 'svelte/store';
import { isUsableServerOrigin, normalizeServerUrl } from '$lib/platform/server-origin';
import { syncTrustedOriginsToNative } from '$lib/platform/ssl-trust';

export const MAX_HOME_SSIDS = 5;

const PROFILE_KEY = 'buhgalter.server_profile.v1';
const LEGACY_URL_KEY = 'buhgalter.server_url';

export type ServerProfile = {
	lanUrl: string;
	remoteUrl: string;
	homeSsids: string[];
	/** When on a home SSID and LAN is unreachable, try remote URL before going offline. */
	lanFallbackRemote: boolean;
	/** HTTPS origins with user-approved self-signed / invalid certificates. */
	trustedOrigins: string[];
};

export type ActiveServerMode = 'lan' | 'remote';

export type ResolvedActiveServer = {
	url: string;
	mode: ActiveServerMode;
};

export const activeServerMode = writable<ActiveServerMode | null>(null);

const emptyProfile = (): ServerProfile => ({
	lanUrl: '',
	remoteUrl: '',
	homeSsids: [],
	lanFallbackRemote: false,
	trustedOrigins: []
});

let memoryProfile: ServerProfile | null = null;

function readStorage(): ServerProfile {
	if (memoryProfile) {
		return {
			...memoryProfile,
			homeSsids: [...memoryProfile.homeSsids],
			trustedOrigins: [...memoryProfile.trustedOrigins]
		};
	}
	if (typeof localStorage === 'undefined') return emptyProfile();
	try {
		const raw = localStorage.getItem(PROFILE_KEY);
		if (!raw) return migrateLegacyProfile();
		const parsed = JSON.parse(raw) as Partial<ServerProfile>;
		return normalizeProfile({
			lanUrl: typeof parsed.lanUrl === 'string' ? parsed.lanUrl : '',
			remoteUrl: typeof parsed.remoteUrl === 'string' ? parsed.remoteUrl : '',
			homeSsids: Array.isArray(parsed.homeSsids) ? parsed.homeSsids : [],
			lanFallbackRemote: parsed.lanFallbackRemote === true,
			trustedOrigins: Array.isArray(parsed.trustedOrigins) ? parsed.trustedOrigins : []
		});
	} catch {
		return migrateLegacyProfile();
	}
}

function writeStorage(profile: ServerProfile) {
	const normalized = normalizeProfile(profile);
	memoryProfile = normalized;
	if (typeof localStorage === 'undefined') return;
	try {
		if (!normalized.lanUrl && !normalized.remoteUrl && normalized.homeSsids.length === 0) {
			localStorage.removeItem(PROFILE_KEY);
			return;
		}
		localStorage.setItem(PROFILE_KEY, JSON.stringify(normalized));
	} catch {
		// ignore quota / private mode
	}
}

function migrateLegacyProfile(): ServerProfile {
	if (typeof localStorage === 'undefined') return emptyProfile();
	try {
		const legacy = localStorage.getItem(LEGACY_URL_KEY);
		if (!legacy) return emptyProfile();
		const profile = normalizeProfile({ lanUrl: legacy, remoteUrl: '', homeSsids: [] });
		localStorage.removeItem(LEGACY_URL_KEY);
		writeStorage(profile);
		return profile;
	} catch {
		return emptyProfile();
	}
}

/** Trim, dedupe, cap at MAX_HOME_SSIDS. SSID matching is case-sensitive (Android). */
export function normalizeHomeSsids(values: string[]): string[] {
	const seen = new Set<string>();
	const result: string[] = [];
	for (const raw of values) {
		const ssid = raw.trim();
		if (!ssid || seen.has(ssid)) continue;
		seen.add(ssid);
		result.push(ssid);
		if (result.length >= MAX_HOME_SSIDS) break;
	}
	return result;
}

export function normalizeTrustedOrigins(values: string[]): string[] {
	const seen = new Set<string>();
	const result: string[] = [];
	for (const raw of values) {
		if (!raw?.trim()) continue;
		let origin: string;
		try {
			origin = normalizeServerUrl(raw);
		} catch {
			continue;
		}
		if (!origin.startsWith('https://') || !isUsableServerOrigin(origin) || seen.has(origin))
			continue;
		seen.add(origin);
		result.push(origin);
	}
	return result;
}

export function normalizeProfile(profile: Partial<ServerProfile>): ServerProfile {
	const lanUrl = profile.lanUrl ? normalizeServerUrl(profile.lanUrl) : '';
	const remoteUrl = profile.remoteUrl ? normalizeServerUrl(profile.remoteUrl) : '';
	return {
		lanUrl: isUsableServerOrigin(lanUrl) ? lanUrl : '',
		remoteUrl: isUsableServerOrigin(remoteUrl) ? remoteUrl : '',
		homeSsids: normalizeHomeSsids(profile.homeSsids ?? []),
		lanFallbackRemote: profile.lanFallbackRemote === true,
		trustedOrigins: normalizeTrustedOrigins(profile.trustedOrigins ?? [])
	};
}

export function getServerProfile(): ServerProfile {
	return readStorage();
}

export function setServerProfile(profile: Partial<ServerProfile>): ServerProfile {
	const next = normalizeProfile({ ...readStorage(), ...profile });
	writeStorage(next);
	void syncTrustedOriginsToNative(next.trustedOrigins);
	return next;
}

export function clearServerProfile(): void {
	memoryProfile = emptyProfile();
	if (typeof localStorage !== 'undefined') {
		try {
			localStorage.removeItem(PROFILE_KEY);
			localStorage.removeItem(LEGACY_URL_KEY);
		} catch {
			// ignore
		}
	}
}

export function hasConfiguredServerProfile(): boolean {
	const profile = getServerProfile();
	return Boolean(profile.lanUrl || profile.remoteUrl);
}

export function isHomeSsid(ssid: string | null | undefined, homeSsids: string[]): boolean {
	if (!ssid) return false;
	return homeSsids.includes(ssid);
}

/** Pick LAN vs remote URL from profile and current Wi‑Fi SSID. */
export function resolveActiveServerUrl(
	ssid: string | null | undefined,
	profile: ServerProfile
): ResolvedActiveServer {
	const { lanUrl, remoteUrl, homeSsids } = profile;

	if (!remoteUrl) {
		return { url: lanUrl, mode: 'lan' };
	}

	if (!lanUrl) {
		return { url: remoteUrl, mode: 'remote' };
	}

	if (isHomeSsid(ssid, homeSsids)) {
		return { url: lanUrl, mode: 'lan' };
	}

	return { url: remoteUrl, mode: 'remote' };
}

export function resetServerProfileForTests(): void {
	clearServerProfile();
	activeServerMode.set(null);
}
