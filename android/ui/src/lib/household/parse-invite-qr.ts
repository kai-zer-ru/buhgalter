import { APP_INVITE_SCHEME, compactInviteCode, isCompleteInviteCode } from './invite-code';

export type ParseInviteQrResult =
	| { ok: true; code: string }
	| { ok: false; reason: 'invalid' }
	| { ok: false; reason: 'foreign_host'; host: string };

export type InviteOriginProfile = {
	lanUrl?: string;
	remoteUrl?: string;
	trustedOrigins?: readonly string[];
};

function originHost(origin: string): string {
	const trimmed = origin.trim();
	if (!trimmed) return '';
	try {
		return new URL(trimmed).host.toLowerCase();
	} catch {
		try {
			return new URL(`https://${trimmed}`).host.toLowerCase();
		} catch {
			return '';
		}
	}
}

/** LAN, remote, current API URL and trusted HTTPS origins of this instance. */
export function inviteAllowedHosts(activeUrl: string, profile: InviteOriginProfile = {}): string[] {
	const hosts = new Set<string>();
	for (const raw of [
		activeUrl,
		profile.lanUrl ?? '',
		profile.remoteUrl ?? '',
		...(profile.trustedOrigins ?? [])
	]) {
		const host = originHost(raw);
		if (host) hosts.add(host);
	}
	return [...hosts];
}

function hostAllowed(urlHost: string, allowed: readonly string[]): boolean {
	if (allowed.length === 0) return true;
	const got = urlHost.toLowerCase();
	if (allowed.includes(got)) return true;
	const hostname = got.split(':')[0] ?? got;
	return allowed.some((entry) => (entry.split(':')[0] ?? entry) === hostname);
}

/** Without a saved remote URL, redeem on the connected instance (LAN-only). */
function shouldRejectForeignHost(profile?: InviteOriginProfile): boolean {
	if (!profile) return true;
	return Boolean(originHost(profile.remoteUrl ?? ''));
}

function codeFromInvitePath(pathname: string): string | null {
	const match = pathname.match(/\/invite\/([^/]+)/i);
	if (!match) return null;
	const compact = compactInviteCode(decodeURIComponent(match[1] ?? ''));
	return isCompleteInviteCode(compact) ? compact : null;
}

/**
 * Accept http(s) `/invite/{code}`, the app scheme, or a bare 9-character code.
 * A http(s) host that is not LAN, remote or the active API origin is rejected
 * only when a remote URL is saved; LAN-only profiles take the code as-is.
 */
export function parseInviteQr(
	text: string,
	expectedOrigin: string | readonly string[],
	profile?: InviteOriginProfile
): ParseInviteQrResult {
	const trimmed = text.trim();
	if (!trimmed) return { ok: false, reason: 'invalid' };

	const schemePrefix = `${APP_INVITE_SCHEME}://`;
	if (trimmed.toLowerCase().startsWith(schemePrefix.toLowerCase())) {
		let rest = trimmed.slice(schemePrefix.length);
		if (!rest.startsWith('/')) rest = `/${rest}`;
		const path = rest.split(/[?#]/, 1)[0] ?? rest;
		const code = codeFromInvitePath(path.startsWith('/invite') ? path : `/invite${path}`);
		if (!code) return { ok: false, reason: 'invalid' };
		return { ok: true, code };
	}

	try {
		const url = new URL(trimmed);
		if (url.protocol === 'http:' || url.protocol === 'https:') {
			const code = codeFromInvitePath(url.pathname);
			if (!code) return { ok: false, reason: 'invalid' };
			const allowed =
				typeof expectedOrigin === 'string'
					? inviteAllowedHosts(expectedOrigin, profile)
					: [...expectedOrigin].map(originHost).filter(Boolean);
			if (!hostAllowed(url.host, allowed) && shouldRejectForeignHost(profile)) {
				return { ok: false, reason: 'foreign_host', host: url.host };
			}
			return { ok: true, code };
		}
	} catch {
		/* bare code */
	}

	const compact = compactInviteCode(trimmed);
	if (isCompleteInviteCode(compact)) return { ok: true, code: compact };
	return { ok: false, reason: 'invalid' };
}
