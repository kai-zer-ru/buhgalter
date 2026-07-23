import { get } from 'svelte/store';
import { _ } from 'svelte-i18n';
import { ApiError } from './client';

/**
 * API codes that group many cases; server always sends a specific localized `message`.
 * Do not replace it with a generic client label like «Конфликт данных».
 */
const GENERIC_API_CODES = new Set([
	'CONFLICT',
	'VALIDATION_ERROR',
	'INTERNAL_ERROR',
	'NOT_FOUND',
	'FORBIDDEN',
	'UNAUTHORIZED',
	'SERVICE_UNAVAILABLE'
]);

/** Errors expected before server URL / auth are configured — no user-facing toast. */
export function isSilentClientError(err: unknown): boolean {
	return err instanceof ApiError && (err.code === 'NO_SERVER' || err.code === 'INVALID_RESPONSE');
}

/** Map API error code to a client-side i18n message (falls back to server message). */
export function formatApiError(err: unknown, fallbackKey = 'common.error'): string {
	const t = get(_);
	if (err instanceof ApiError) {
		const message = err.message?.trim();
		if (message && GENERIC_API_CODES.has(err.code)) {
			return message;
		}
		const byCode = t(`errors.${err.code}`);
		if (byCode && byCode !== `errors.${err.code}`) {
			return byCode;
		}
		if (message) {
			return message;
		}
	}
	return t(fallbackKey);
}
