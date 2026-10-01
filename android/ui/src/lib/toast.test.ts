import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { get } from 'svelte/store';

vi.mock('svelte-i18n', () => ({
	_: {
		subscribe: (run: (v: (key: string) => string) => void) => {
			run((key: string) =>
				key === 'common.server_unavailable'
					? 'Сервер недоступен. Сессия сохранена — попробуйте обновить.'
					: key
			);
			return () => {};
		}
	}
}));

import { toast, toastStore } from './toast';

describe('toast', () => {
	beforeEach(() => {
		vi.useFakeTimers();
	});

	afterEach(() => {
		vi.runOnlyPendingTimers();
		vi.useRealTimers();
	});

	it('exposes typed helpers', () => {
		expect(typeof toast.success).toBe('function');
		expect(typeof toast.error).toBe('function');
		expect(typeof toast.warning).toBe('function');
		expect(typeof toast.fromError).toBe('function');
	});

	it('pushes and auto-dismisses success toast', () => {
		toast.success('Saved');
		let items = get(toastStore as Parameters<typeof get>[0]);
		expect(items).toHaveLength(1);
		expect(items[0]?.type).toBe('success');
		expect(items[0]?.message).toBe('Saved');
		vi.advanceTimersByTime(4000);
		items = get(toastStore as Parameters<typeof get>[0]);
		expect(items).toHaveLength(0);
	});

	it('fromError uses plain Error message', () => {
		toast.fromError(new Error('Validation failed'));
		const items = get(toastStore as Parameters<typeof get>[0]);
		expect(items[0]?.type).toBe('error');
		expect(items[0]?.message).toBe('Validation failed');
	});

	it('fromError does not show cache-miss URL', () => {
		const err = new Error('No cached data for /api/v1/transactions?page=2');
		err.name = 'OfflineCacheMissError';
		toast.fromError(err);
		const items = get(toastStore as Parameters<typeof get>[0]);
		expect(items[0]?.type).toBe('error');
		expect(items[0]?.message).toBe('Сервер недоступен. Сессия сохранена — попробуйте обновить.');
	});

	it('fromError swallows connection errors', () => {
		toast.fromError(new Error('Failed to connect to /192.168.0.10:8766'));
		const items = get(toastStore as Parameters<typeof get>[0]);
		expect(items).toHaveLength(0);
	});
});
