import { beforeEach, describe, expect, it, vi } from 'vitest';
import { assertOnline, OnlineOnlyError, requireOnline } from '$lib/offline/require-online';
import * as connectivity from '$lib/offline/server-connectivity';

const toastError = vi.fn();

vi.mock('$lib/toast', () => ({
	toast: { error: (...args: unknown[]) => toastError(...args) }
}));

vi.mock('svelte-i18n', () => ({
	_: {
		subscribe: (fn: (v: (k: string) => string) => void) => {
			fn((k) => k);
			return () => undefined;
		}
	}
}));

vi.mock('$lib/offline/server-connectivity', async (importOriginal) => {
	const actual = await importOriginal<typeof connectivity>();
	return {
		...actual,
		isServerOfflineMode: vi.fn()
	};
});

beforeEach(() => {
	toastError.mockReset();
	vi.mocked(connectivity.isServerOfflineMode).mockReturnValue(false);
});

describe('requireOnline', () => {
	it('returns true when online', () => {
		expect(requireOnline()).toBe(true);
		expect(toastError).not.toHaveBeenCalled();
	});

	it('toasts and returns false when offline', () => {
		vi.mocked(connectivity.isServerOfflineMode).mockReturnValue(true);
		expect(requireOnline('offline.onlineOnly.creditCreate')).toBe(false);
		expect(toastError).toHaveBeenCalledWith('offline.onlineOnly.creditCreate');
	});
});

describe('assertOnline', () => {
	it('throws OnlineOnlyError when offline', () => {
		vi.mocked(connectivity.isServerOfflineMode).mockReturnValue(true);
		expect(() => assertOnline('offline.onlineOnly.creditCreate')).toThrow(OnlineOnlyError);
	});
});
