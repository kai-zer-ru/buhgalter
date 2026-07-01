import { get } from 'svelte/store';
import { _ } from 'svelte-i18n';
import { ApiError } from '$lib/api/client';

/** Umbrella API codes: specific text is in `error.message` (localized by server). */
const UMBRELLA_CODES = new Set(['VALIDATION_ERROR', 'CONFLICT']);

function fieldLabel(field: string): string | null {
	const t = get(_);
	const key = `auth.errors.field.${field}`;
	const label = t(key);
	return label !== key ? label : null;
}

function withFieldHint(message: string, field?: string): string {
	if (!field) return message;
	const label = fieldLabel(field);
	return label ? `${label}: ${message}` : message;
}

/** Human-readable API errors for register, login and admin user management. */
export function formatAuthUserApiError(
	err: unknown,
	fallbackKey = 'common.error',
	attachFieldHint = true
): string {
	const t = get(_);
	if (!(err instanceof ApiError)) return t(fallbackKey);

	const msg = err.message?.trim() ?? '';

	if (UMBRELLA_CODES.has(err.code) && msg) {
		return attachFieldHint ? withFieldHint(msg, err.field) : msg;
	}

	const byCode = t(`errors.${err.code}`);
	if (byCode !== `errors.${err.code}`) {
		return attachFieldHint ? withFieldHint(byCode, err.field) : byCode;
	}

	if (msg) return attachFieldHint ? withFieldHint(msg, err.field) : msg;
	return t(fallbackKey);
}

export function authUserApiField(err: unknown): string | undefined {
	return err instanceof ApiError ? err.field : undefined;
}
