import { get } from 'svelte/store';
import { _ } from 'svelte-i18n';
import { ApiError } from './client';

/** Map API error code to a client-side i18n message (falls back to server message). */
export function formatApiError(err: unknown, fallbackKey = 'common.error'): string {
	const t = get(_);
	if (err instanceof ApiError) {
		if (err.message && err.code === 'VALIDATION_ERROR') {
			return err.message;
		}
		const byCode = t(`errors.${err.code}`);
		if (byCode && byCode !== `errors.${err.code}`) {
			return byCode;
		}
		if (err.message) {
			return err.message;
		}
	}
	return t(fallbackKey);
}
