import { login, type User } from '$lib/api/client';
import { clearAuthToken, setAuthToken } from '$lib/platform/auth-token';
import {
	clearServerProfile,
	getServerProfile,
	setServerProfile,
	type ServerProfile
} from '$lib/platform/server-profile';
import { refreshActiveServerUrl } from '$lib/platform/server-url';

/** Public demo stand. Not a setting in the app. */
export const DEMO_SERVER_ORIGIN = 'https://buhgalter-demo.kai-zer.ru';
export const DEMO_LOGIN = 'demo';
export const DEMO_PASSWORD = 'demo_1_demo';

export function demoServerProfile(): ServerProfile {
	return {
		lanUrl: DEMO_SERVER_ORIGIN,
		remoteUrl: '',
		homeSsids: [],
		lanFallbackRemote: false,
		trustedOrigins: []
	};
}

/** Saved origin that is not the demo stand must be confirmed before replacement. */
export function needsDemoReplaceConfirm(profile: ServerProfile): boolean {
	const origins = [profile.lanUrl, profile.remoteUrl].filter(Boolean);
	if (origins.length === 0) return false;
	return origins.some((origin) => origin !== DEMO_SERVER_ORIGIN);
}

/**
 * Point the active URL at the demo stand and open a session Bearer.
 * `login()` talks to the active URL, so the profile is written for the attempt
 * and put back if the network or login fails.
 */
export async function enterDemoMode(): Promise<User> {
	const previous = getServerProfile();
	const hadProfile = Boolean(previous.lanUrl || previous.remoteUrl);
	try {
		setServerProfile(demoServerProfile());
		await refreshActiveServerUrl();
		await clearAuthToken();
		const res = await login(DEMO_LOGIN, DEMO_PASSWORD);
		await setAuthToken(res.token, 'session');
		return res.user;
	} catch (err) {
		if (hadProfile) setServerProfile(previous);
		else clearServerProfile();
		try {
			await refreshActiveServerUrl();
		} catch {
			// Profile is already the pre-attempt one. A failed refresh must not hide the login error.
		}
		await clearAuthToken();
		throw err;
	}
}
