import { describe, expect, it, vi } from 'vitest';

const translations: Record<string, string> = {
	'common.loadFailed': 'Не удалось загрузить данные',
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

vi.mock('./toast', () => ({
	toast: {
		fromError: vi.fn()
	}
}));

import { ApiError } from './api/client';
import { capturePageLoadError, reportPageLoadFailure } from './page-load';
import { toast } from './toast';

describe('page-load', () => {
	it('capturePageLoadError uses API message for generic codes', () => {
		const err = new ApiError('SERVICE_UNAVAILABLE', 'Сервер недоступен', 503);
		expect(capturePageLoadError(err)).toBe('Сервер недоступен');
	});

	it('capturePageLoadError falls back for unknown errors', () => {
		expect(capturePageLoadError(new Error('boom'))).toBe('Не удалось загрузить данные');
	});

	it('reportPageLoadFailure toasts on background refresh with data', () => {
		const err = new Error('network');
		const result = reportPageLoadFailure(err, { background: true, hasData: true });
		expect(result).toBeNull();
		expect(toast.fromError).toHaveBeenCalledWith(err);
	});

	it('reportPageLoadFailure returns message when no cached data', () => {
		vi.mocked(toast.fromError).mockClear();
		const err = new Error('network');
		const result = reportPageLoadFailure(err, { background: true, hasData: false });
		expect(result).toBe('Не удалось загрузить данные');
		expect(toast.fromError).not.toHaveBeenCalled();
	});
});
