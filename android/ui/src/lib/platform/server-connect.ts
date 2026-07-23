import { ApiError } from '$lib/api/client';

/** i18n key for inline server connection errors. */
export function serverConnectErrorKey(err: unknown): string {
	if (err instanceof ApiError) {
		if (err.code === 'SSL_CERTIFICATE') {
			return 'serverSetup.ssl.untrusted';
		}
		if (
			err.status === 403 ||
			err.code === 'FORBIDDEN' ||
			err.code === 'ERR_EXTERNAL_ACCESS_DENIED'
		) {
			return 'serverSetup.accessDenied';
		}
		if (err.code === 'INVALID_RESPONSE') {
			return 'serverSetup.wrongEndpoint';
		}
	}
	return 'serverSetup.unreachable';
}
