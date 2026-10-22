import { ApiError, pingServer } from '$lib/api/client';
import { isNativeApp } from '$lib/platform/native';
import {
	isHttpsOrigin,
	isOriginTrusted,
	isSslCertificateError,
	nativeHttpRequest,
	nativeResultToApiError
} from '$lib/platform/ssl-trust';
import { normalizeServerUrl } from '$lib/platform/server-origin';
import {
	normalizeProfile,
	normalizeTrustedOrigins,
	type ServerProfile
} from '$lib/platform/server-profile';

export class SslTrustRequiredError extends Error {
	readonly origin: string;

	constructor(origin: string) {
		super('SSL_CERTIFICATE');
		this.name = 'SslTrustRequiredError';
		this.origin = origin;
	}

	static is(err: unknown): err is SslTrustRequiredError {
		return err instanceof SslTrustRequiredError;
	}
}

async function pingOrigin(origin: string, profile: ServerProfile): Promise<void> {
	const normalized = normalizeServerUrl(origin);
	if (!normalized) return;

	if (isNativeApp() && isHttpsOrigin(normalized)) {
		const result = await nativeHttpRequest(`${normalized}/api/v1/health`, {
			method: 'GET',
			allowUntrusted: isOriginTrusted(normalized, profile)
		});
		const apiErr = nativeResultToApiError(result);
		if (apiErr) throw apiErr;
		if (!result.ok) {
			throw new ApiError('UNREACHABLE', 'Health check failed', result.status ?? 0);
		}
		let parsed: { status?: string; version?: string; db?: string };
		try {
			parsed = JSON.parse(result.body ?? '{}') as typeof parsed;
		} catch {
			throw new ApiError('INVALID_RESPONSE', 'Expected JSON from API', result.status ?? 0);
		}
		if (!parsed?.status || !parsed?.version || !parsed?.db) {
			throw new ApiError('INVALID_RESPONSE', 'Expected JSON from API', result.status ?? 0);
		}
		return;
	}

	await pingServer(normalized);
}

/** Ping LAN and remote URLs; throws SslTrustRequiredError when HTTPS cert is not trusted. */
export async function verifyServerProfile(
	profileInput: Partial<ServerProfile>
): Promise<ServerProfile> {
	const profile = normalizeProfile(profileInput);
	if (!profile.lanUrl) {
		throw new ApiError('VALIDATION', 'LAN URL required', 0);
	}

	for (const origin of [profile.lanUrl, profile.remoteUrl].filter(Boolean)) {
		try {
			await pingOrigin(origin, profile);
		} catch (err) {
			if (isSslCertificateError(err)) {
				throw new SslTrustRequiredError(origin);
			}
			throw err;
		}
	}

	return profile;
}

export function addTrustedOrigin(
	profile: ServerProfile,
	origin: string,
	trusted: boolean
): ServerProfile {
	const normalized = normalizeServerUrl(origin);
	if (!normalized.startsWith('https://')) return profile;
	const next = new Set(profile.trustedOrigins);
	if (trusted) next.add(normalized);
	else next.delete(normalized);
	return normalizeProfile({ ...profile, trustedOrigins: [...next] });
}

export function mergeTrustedOrigins(profile: ServerProfile, origins: string[]): ServerProfile {
	return normalizeProfile({
		...profile,
		trustedOrigins: normalizeTrustedOrigins([...profile.trustedOrigins, ...origins])
	});
}
