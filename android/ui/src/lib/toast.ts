import { writable } from 'svelte/store';
import { formatApiError, isSilentClientError } from '$lib/api/errors';
import { ApiError } from '$lib/api/client';
import { isConnectionError } from '$lib/offline/server-connectivity';

export type ToastType = 'success' | 'error' | 'info' | 'warning';

export type ToastItem = {
	id: number;
	message: string;
	type: ToastType;
};

const DURATION_MS: Record<ToastType, number> = {
	success: 3200,
	info: 3200,
	error: 4500,
	warning: 4500
};

const { subscribe, update } = writable<ToastItem[]>([]);

let nextId = 1;
let lastPushedMessage = '';
let lastPushedAt = 0;

export const toastStore = { subscribe };

function push(message: string, type: ToastType, durationMs = DURATION_MS[type]) {
	const now = Date.now();
	if (message === lastPushedMessage && now - lastPushedAt < 3000) {
		return;
	}
	lastPushedMessage = message;
	lastPushedAt = now;

	const id = nextId++;
	update((items) => [...items, { id, message, type }]);
	globalThis.setTimeout(() => {
		update((items) => items.filter((item) => item.id !== id));
	}, durationMs);
}

function toastFn(message: string, type: ToastType = 'success', durationMs?: number) {
	push(message, type, durationMs ?? DURATION_MS[type]);
}

export const toast = Object.assign(toastFn, {
	success: (message: string) => push(message, 'success'),
	error: (message: string) => push(message, 'error'),
	warning: (message: string) => push(message, 'warning'),
	info: (message: string) => push(message, 'info'),
	fromError: (err: unknown, fallbackKey = 'common.error') => {
		if (isSilentClientError(err) || isConnectionError(err)) return;
		if (err instanceof Error && !(err instanceof ApiError) && err.message) {
			push(err.message, 'error');
			return;
		}
		push(formatApiError(err, fallbackKey), 'error');
	}
});
