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

/** True when the string is an API code (`ERR_FOO`, `CONFLICT_BAR`) rather than a sentence. */
export function isRawErrorCode(text: string, code?: string): boolean {
	const t = text.trim();
	if (!t) return true;
	if (code && (t === code || t === `errors.${code}`)) return true;
	return /^[A-Z][A-Z0-9_]{2,}$/.test(t);
}

function labelForCode(t: (key: string) => string, code: string): string | null {
	if (!code) return null;
	const byCode = t(`errors.${code}`);
	if (byCode && !isRawErrorCode(byCode, code)) {
		return byCode;
	}
	return null;
}

/** Map API error code to a client-side i18n message (never the raw `ERR_…` code). */
export function formatApiError(err: unknown, fallbackKey = 'common.error'): string {
	const t = get(_);
	if (err instanceof ApiError) {
		const message = err.message?.trim() ?? '';
		if (GENERIC_API_CODES.has(err.code) && message && !isRawErrorCode(message, err.code)) {
			return message;
		}
		const labeled = labelForCode(t, err.code);
		if (labeled) return labeled;
		if (message && !isRawErrorCode(message, err.code)) {
			return message;
		}
		return t(fallbackKey);
	}
	if (err instanceof Error) {
		const message = err.message?.trim() ?? '';
		if (isRawErrorCode(message)) {
			return labelForCode(t, message) ?? t(fallbackKey);
		}
	}
	return t(fallbackKey);
}
