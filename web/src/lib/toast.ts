import { writable } from 'svelte/store';

export type ToastType = 'success' | 'error' | 'info';

export type ToastItem = {
	id: number;
	message: string;
	type: ToastType;
};

const { subscribe, update } = writable<ToastItem[]>([]);

let nextId = 1;

export const toastStore = { subscribe };

export function toast(message: string, type: ToastType = 'success', durationMs = 3200) {
	const id = nextId++;
	update((items) => [...items, { id, message, type }]);
	window.setTimeout(() => {
		update((items) => items.filter((item) => item.id !== id));
	}, durationMs);
}
