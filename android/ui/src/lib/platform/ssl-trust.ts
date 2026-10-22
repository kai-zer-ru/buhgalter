import { ApiError } from '$lib/api/client';
import { getServerProfile, type ServerProfile } from '$lib/platform/server-profile';
import { isNativeApp } from '$lib/platform/native';
import { normalizeServerUrl } from '$lib/platform/server-origin';
import { SslTrust, type NativeHttpResult } from '$lib/platform/ssl-trust-native';

export function isHttpsOrigin(origin: string): boolean {
	return origin.startsWith('https://');
}

export function isOriginTrusted(origin: string, profile?: ServerProfile): boolean {
	const normalized = normalizeServerUrl(origin);
	return (profile ?? getServerProfile()).trustedOrigins.includes(normalized);
}

export function isSslCertificateError(err: unknown): boolean {
	return err instanceof ApiError && err.code === 'SSL_CERTIFICATE';
}

export async function syncTrustedOriginsToNative(origins: string[]): Promise<void> {
	if (!isNativeApp()) return;
	try {
		await SslTrust.setTrustedOrigins({ origins });
	} catch {
		// native plugin unavailable in tests / web preview
	}
}

const NATIVE_HTTP_TIMEOUT_MS = 15_000;

export async function nativeHttpRequest(
	url: string,
	opts: {
		method?: string;
		headers?: Record<string, string>;
		body?: string;
		allowUntrusted?: boolean;
	} = {}
): Promise<NativeHttpResult> {
	if (!isNativeApp()) {
		return { errorCode: 'UNREACHABLE', message: 'native only' };
	}
	try {
		const request = SslTrust.request({
			url,
			method: opts.method ?? 'GET',
			headers: opts.headers,
			body: opts.body,
			allowUntrusted: opts.allowUntrusted ?? false
		});
		// Belt-and-suspenders: Capacitor plugin queue can stall; JS must not hang forever.
		let timer: ReturnType<typeof setTimeout> | undefined;
		const timeout = new Promise<NativeHttpResult>((resolve) => {
			timer = setTimeout(() => {
				resolve({ errorCode: 'UNREACHABLE', message: 'Request timed out' });
			}, NATIVE_HTTP_TIMEOUT_MS);
		});
		try {
			return await Promise.race([request, timeout]);
		} finally {
			if (timer !== undefined) clearTimeout(timer);
		}
	} catch (err) {
		return {
			errorCode: 'UNREACHABLE',
			message: err instanceof Error ? err.message : 'request failed'
		};
	}
}

export function nativeResultToApiError(result: NativeHttpResult): ApiError | null {
	if (result.errorCode === 'SSL_CERTIFICATE') {
		return new ApiError('SSL_CERTIFICATE', result.message ?? 'Untrusted certificate', 0);
	}
	if (result.errorCode) {
		return new ApiError('UNREACHABLE', result.message ?? 'Could not connect', 0);
	}
	return null;
}
