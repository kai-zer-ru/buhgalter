import { describe, expect, it, vi } from 'vitest';
import { ApiError } from './client';

const translations: Record<string, string> = {
	'errors.CONFLICT': 'Конфликт данных',
	'errors.PASSWORDS_MISMATCH': 'Пароли не совпадают',
	'common.error': 'Ошибка',
	'common.server_unavailable': 'Сервер недоступен. Сессия сохранена — попробуйте обновить.'
};

vi.mock('svelte/store', async (importOriginal) => {
	const actual = await importOriginal<typeof import('svelte/store')>();
	return {
		...actual,
		get: () => (key: string) => translations[key] ?? key
	};
});

vi.mock('svelte-i18n', () => ({
	_: {}
}));

import { formatApiError } from './errors';

describe('formatApiError', () => {
	it('prefers server message for generic CONFLICT code', () => {
		const err = new ApiError(
			'CONFLICT',
			'Нельзя удалить операцию долга после погашения — удалите долг целиком',
			409
		);
		expect(formatApiError(err)).toBe(
			'Нельзя удалить операцию долга после погашения — удалите долг целиком'
		);
	});

	it('uses client i18n for specific error codes', () => {
		const err = new ApiError('PASSWORDS_MISMATCH', 'Passwords mismatch', 400);
		expect(formatApiError(err)).toBe('Пароли не совпадают');
	});

	it('falls back to generic CONFLICT label when server message is empty', () => {
		const err = new ApiError('CONFLICT', '', 409);
		expect(formatApiError(err)).toBe('Конфликт данных');
	});

	it('does not leak native connect URL for UNREACHABLE', () => {
		const err = new ApiError('UNREACHABLE', 'Failed to connect to /192.168.0.10:8766', 0);
		expect(formatApiError(err)).toBe('Сервер недоступен. Сессия сохранена — попробуйте обновить.');
	});

	it('does not leak cache-miss path', () => {
		const err = new Error('No cached data for /api/v1/transactions?page=2&limit=20');
		err.name = 'OfflineCacheMissError';
		expect(formatApiError(err)).toBe('Сервер недоступен. Сессия сохранена — попробуйте обновить.');
	});

	it('keeps SSL certificate text', () => {
		const err = new ApiError('SSL_CERTIFICATE', 'Untrusted certificate for https://example', 0);
		expect(formatApiError(err)).toBe('Untrusted certificate for https://example');
	});
});
