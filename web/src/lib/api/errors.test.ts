import { describe, expect, it, vi } from 'vitest';
import { ApiError } from './client';

const translations: Record<string, string> = {
	'errors.CONFLICT': 'Конфликт данных',
	'errors.PASSWORDS_MISMATCH': 'Пароли не совпадают',
	'common.error': 'Ошибка'
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

	it('does not show a raw ERR_ code when i18n and server message are missing', () => {
		const err = new ApiError(
			'ERR_CREDIT_CARD_PAYMENT_EXCEEDS_LIMIT',
			'ERR_CREDIT_CARD_PAYMENT_EXCEEDS_LIMIT',
			400
		);
		expect(formatApiError(err)).toBe('Ошибка');
	});

	it('does not show a generic code when server message repeats the code', () => {
		const err = new ApiError('CONFLICT', 'CONFLICT', 409);
		expect(formatApiError(err)).toBe('Конфликт данных');
	});
});
